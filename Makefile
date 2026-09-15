.PHONY: all generate compile up wait-db test run-tests down clean

all: test

# 1. Tareas previas
generate:
	@echo "==> Generando codigo Go con sqlc..."
	@sqlc generate

compile:
	@echo "==> Verificando dependencias y compilacion..."
	@go mod tidy
	@go test -c ./db/sqlc -o tmp_test.bin
	@rm -f tmp_test.bin tmp_test.bin.exe

up:
	@echo "==> Levantando contenedor PostgreSQL..."
	@docker compose down -v --remove-orphans > /dev/null 2>&1 || true
	@docker compose up -d db

wait-db:
	@echo "==> Esperando a que PostgreSQL este listo..."
	@bash -c 'until docker compose exec -T db pg_isready -U postgres -d todolist_test > /dev/null 2>&1; do sleep 1; done'
	@echo "==> PostgreSQL esta listo."

# 2. Ejecucion de tests
run-tests:
	@echo "==> Ejecutando suite de tests con go test..."
	@DATABASE_URL="postgres://postgres:postgres@localhost:5432/todolist_test?sslmode=disable" go test -v -count=1 ./...

# 3. Tareas posteriores
down:
	@echo "==> Limpiando contenedores y volumenes de Docker..."
	@docker compose down -v --remove-orphans

clean: down
	@rm -f tmp_test.bin tmp_test.bin.exe

# Flujo orquestado completo
test:
	@bash test.sh
