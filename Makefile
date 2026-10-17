.PHONY: all generate compile up wait-db test test-api run-tests down clean run

all: test-api

# 1. Generacion de codigo con sqlc
generate:
	@echo "==> Generando codigo Go con sqlc..."
	@sqlc generate

# 2. Compilacion y dependencias
compile:
	@echo "==> Verificando dependencias y compilacion..."
	@go mod tidy
	@go test -c ./db/sqlc -o tmp_test.bin
	@rm -f tmp_test.bin tmp_test.bin.exe

# 3. Levantar base de datos PostgreSQL aislada
up:
	@echo "==> Levantando contenedor PostgreSQL..."
	@docker compose down -v --remove-orphans > /dev/null 2>&1 || true
	@docker compose up -d db

wait-db:
	@echo "==> Esperando a que PostgreSQL este listo..."
	@bash -c 'until docker compose exec -T db pg_isready -U postgres -d todolist_test > /dev/null 2>&1; do sleep 1; done'
	@echo "==> PostgreSQL esta listo."

# 4. Pruebas unitarias y de persistencia (TP2)
run-tests:
	@echo "==> Ejecutando suite de tests con go test..."
	@DATABASE_URL="postgres://postgres:postgres@localhost:5432/todolist_test?sslmode=disable" go test -v -count=1 ./...

test:
	@bash test.sh

# 5. Pruebas de API REST (TP3)
test-api:
	@bash test-api.sh

# 6. Levantar stack completo en Docker (API + DB)
run: generate
	@echo "==> Levantando stack completo con Docker Compose..."
	@docker compose up --build

# 7. Tareas posteriores y limpieza
down:
	@echo "==> Limpiando contenedores y volumenes de Docker..."
	@docker compose down -v --remove-orphans

clean: down
	@rm -f tmp_test.bin tmp_test.bin.exe tmp_server tmp_server.exe
