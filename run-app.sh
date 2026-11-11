#!/usr/bin/env bash

# Script de Ejecución (TP5)
# Renderizado del Lado del Servidor (SSR)

set -euo pipefail

# Funcion para limpieza posterior garantizada (al salir o presionar Ctrl+C)
cleanup() {
    echo ""
    echo "==> [3/3] Tareas posteriores: Deteniendo aplicación, contenedores y volumenes..."
    docker compose down -v --remove-orphans > /dev/null 2>&1 || true
    rm -f tmp_app tmp_app.exe
    echo "==> Entorno limpio."
}

trap cleanup EXIT INT TERM

echo "============================================================"
echo "    TP5 - Renderizado del Lado del Servidor con Templ       "
echo "============================================================"

echo "==> [1/3] Tareas previas..."

# 0. Verificacion y creacion automatica de .env a partir de .env.example
if [ ! -f .env ]; then
    echo " -> Archivo .env no encontrado. Creando automáticamente a partir de .env.example..."
    cp .env.example .env
else
    echo " -> Archivo .env detectado."
fi

# Cargar variables de entorno desde .env
set -a
# shellcheck disable=SC1091
source .env
set +a

# 1. Limpieza preventiva de contenedores o volumenes huerfanos anteriores
echo " -> Limpieza preventiva de contenedores y volumenes previos..."
docker compose down -v --remove-orphans > /dev/null 2>&1 || true

# 2. Ejecucion de sqlc generate para generar codigo Go de persistencia
echo " -> Ejecutando sqlc generate..."
if command -v sqlc >/dev/null 2>&1; then
    sqlc generate
elif [ -f "$(go env GOPATH)/bin/sqlc" ]; then
    "$(go env GOPATH)/bin/sqlc" generate
elif [ -f "$(go env GOPATH)/bin/sqlc.exe" ]; then
    "$(go env GOPATH)/bin/sqlc.exe" generate
else
    echo "    sqlc no encontrado en PATH, ejecutando via go run..."
    go run github.com/sqlc-dev/sqlc/cmd/sqlc@v1.31.1 generate
fi

# 3. Ejecucion de templ generate para compilar componentes de vistas a codigo Go
echo " -> Ejecutando templ generate..."
if command -v templ >/dev/null 2>&1; then
    templ generate
elif [ -f "$(go env GOPATH)/bin/templ" ]; then
    "$(go env GOPATH)/bin/templ" generate
elif [ -f "$(go env GOPATH)/bin/templ.exe" ]; then
    "$(go env GOPATH)/bin/templ.exe" generate
else
    echo "    templ no encontrado en PATH, ejecutando via go run..."
    go run github.com/a-h/templ/cmd/templ@v0.3.1100 generate
fi

# 4. Descarga de dependencias y compilacion del servidor
echo " -> Verificando dependencias y compilando aplicacion..."
go mod tidy
go build -o tmp_app main.go

# 5. Levantar contenedor de base de datos PostgreSQL
echo " -> Levantando contenedor de base de datos PostgreSQL..."
docker compose up -d db

# 6. Esperar a que PostgreSQL este listo
echo " -> Esperando a que PostgreSQL este listo..."
MAX_ATTEMPTS=30
ATTEMPT=1
until docker compose exec -T db pg_isready -U "${DB_USER:-postgres}" -d "${DB_NAME:-todolist_test}" > /dev/null 2>&1; do
    if [ "$ATTEMPT" -ge "$MAX_ATTEMPTS" ]; then
        echo "ERROR: Tiempo de espera agotado esperando a PostgreSQL."
        docker compose logs db
        exit 1
    fi
    echo "    Intento $ATTEMPT/$MAX_ATTEMPTS: esperando servicio db..."
    sleep 1
    ATTEMPT=$((ATTEMPT + 1))
done
sleep 1
echo " -> Base de datos PostgreSQL lista y respondiendo."

echo ""
echo "==> [2/3] Ejecutando la aplicacion..."
PORT="${PORT:-:8080}"
if [[ "$PORT" != :* ]]; then
    PORT=":$PORT"
fi
DATABASE_URL="${DATABASE_URL:-postgres://postgres:postgres@localhost:5432/todolist_test?sslmode=disable}"
export PORT DATABASE_URL

echo " -> Abrir tu navegador en: http://localhost${PORT}"
echo " -> Presionar Ctrl+C para detener el servidor y realizar la limpieza automatica."
echo ""

if [ -f "./tmp_app" ]; then
    ./tmp_app
elif [ -f "./tmp_app.exe" ]; then
    ./tmp_app.exe
fi
