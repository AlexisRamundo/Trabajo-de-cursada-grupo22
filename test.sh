#!/usr/bin/env bash
set -euo pipefail

# Función para limpieza posterior de contenedores y volúmenes
cleanup() {
    echo ""
    echo "==> [3/3] Tareas posteriores: Deteniendo y borrando contenedores y volumenes..."
    docker compose down -v --remove-orphans > /dev/null 2>&1 || true
    echo "==> Entorno limpio."
}

# Registrar trap para asegurar ejecución de tareas posteriores ante éxito o error
trap cleanup EXIT

echo "==> [1/3] Tareas previas..."

# 1. Limpieza preventiva de contenedores o volúmenes huérfanos anteriores
echo " -> Limpieza preventiva de contenedores y volumenes previos..."
docker compose down -v --remove-orphans > /dev/null 2>&1 || true

# 2. Ejecución de sqlc para generar código Go
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

# 3. Descarga de dependencias y verificación de compilación
echo " -> Verificando dependencias y compilacion..."
go mod tidy
go test -c ./db/sqlc -o /dev/null 2>/dev/null || go test -c ./db/sqlc -o tmp_test.bin
rm -f tmp_test.bin tmp_test.bin.exe

# 4. Levantar contenedores con Docker Compose
echo " -> Levantando contenedor de base de datos PostgreSQL..."
docker compose up -d db

# 5. Esperar a que la base de datos esté lista para aceptar conexiones
echo " -> Esperando a que PostgreSQL esté listo..."
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

echo ""
echo "==> [2/3] Ejecucion de tests unitarios y de persistencia..."
DATABASE_URL="postgres://postgres:postgres@localhost:5432/todolist_test?sslmode=disable" \
go test -v -count=1 ./...

echo ""
echo "==> Todos los tests se ejecutaron exitosamente!"