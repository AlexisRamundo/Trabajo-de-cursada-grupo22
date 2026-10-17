#!/usr/bin/env bash

# Script de Automatización de Pruebas de API REST (TP3)

set -euo pipefail

SERVER_PID=""

# Función para limpieza posterior garantizada
cleanup() {
    echo ""
    echo "==> [3/3] Tareas posteriores: Deteniendo servidor, contenedores y volumenes..."
    if [ -n "$SERVER_PID" ] && kill -0 "$SERVER_PID" 2>/dev/null; then
        echo " -> Deteniendo proceso del servidor (PID: $SERVER_PID)..."
        kill "$SERVER_PID" 2>/dev/null || true
        wait "$SERVER_PID" 2>/dev/null || true
    fi
    docker compose down -v --remove-orphans > /dev/null 2>&1 || true
    rm -f tmp_server tmp_server.exe
    echo "==> Entorno limpio."
}

trap cleanup EXIT

echo "============================================================"
echo "          TP3 - Exponiendo la Logica como API REST          "
echo "============================================================"

echo "==> [1/3] Tareas previas..."

# 1. Limpieza preventiva de contenedores o volumenes huerfanos anteriores
echo " -> Limpieza preventiva de contenedores y volumenes previos..."
docker compose down -v --remove-orphans > /dev/null 2>&1 || true

# 2. Ejecucion de sqlc para generar codigo Go
echo " -> Ejecutando sqlc generate..."
if command -v sqlc >/dev/null 2>&1; then
    sqlc generate
elif [ -f "$(go env GOPATH)/bin/sqlc" ]; then
    "$(go env GOPATH)/bin/sqlc" generate
elif [ -f "$(go env GOPATH)/bin/sqlc.exe" ]; then
    "$(go env GOPATH)/bin/sqlc.exe" generate
else
    echo "sqlc no encontrado en PATH, ejecutando via go run..."
    go run github.com/sqlc-dev/sqlc/cmd/sqlc@v1.31.1 generate
fi

# 3. Descarga de dependencias y compilacion del servidor
echo " -> Verificando dependencias y compilando servidor..."
go mod tidy
go build -o tmp_server main.go

# 4. Levantar contenedor de base de datos PostgreSQL
echo " -> Levantando contenedor de base de datos PostgreSQL..."
docker compose up -d db

# 5. Esperar a que PostgreSQL este listo
echo " -> Esperando a que PostgreSQL este listo..."
MAX_ATTEMPTS=30
ATTEMPT=1
until docker compose exec -T db pg_isready -U postgres -d todolist_test > /dev/null 2>&1; do
    if [ "$ATTEMPT" -ge "$MAX_ATTEMPTS" ]; then
        echo "ERROR: Tiempo de espera agotado esperando a PostgreSQL."
        docker compose logs db
        exit 1
    fi
    echo "    Intento $ATTEMPT/$MAX_ATTEMPTS: esperando servicio db..."
    sleep 1
    ATTEMPT=$((ATTEMPT + 1))
done
echo " -> Base de datos PostgreSQL lista y respondiendo."

# 6. Levantar la aplicacion
echo " -> Iniciando servidor API en http://localhost:8080..."
DATABASE_URL="postgres://postgres:postgres@localhost:5432/todolist_test?sslmode=disable" \
PORT=":8080" \
./tmp_server &
SERVER_PID=$!

# Esperar a que el servidor HTTP responda
echo " -> Esperando a que la API este lista para recibir peticiones..."
MAX_API_ATTEMPTS=20
API_ATTEMPT=1
until curl -s http://localhost:8080/health > /dev/null 2>&1 || curl -s http://localhost:8080/tasks > /dev/null 2>&1; do
    if [ "$API_ATTEMPT" -ge "$MAX_API_ATTEMPTS" ]; then
        echo "ERROR: El servidor API no respondio a tiempo."
        exit 1
    fi
    sleep 0.5
    API_ATTEMPT=$((API_ATTEMPT + 1))
done
echo " -> Servidor API activo y respondiendo."

echo ""
echo "==> [2/3] Prueba de cada uno de los endpoints de la API con HURL..."

# Verificar si HURL está disponible
if ! command -v hurl >/dev/null 2>&1; then
    echo "ERROR: 'hurl' no se encuentra instalado en el sistema."
    echo "Para ejecutar la suite de pruebas se requiere Hurl (requests.hurl)."
    echo "Instrucciones de instalación oficial: https://hurl.dev"
    exit 1
fi

echo " -> Ejecutando pruebas con HURL (requests.hurl)..."
hurl --test requests.hurl
echo " -> ¡Todas las pruebas con HURL (requests.hurl) finalizaron exitosamente!"