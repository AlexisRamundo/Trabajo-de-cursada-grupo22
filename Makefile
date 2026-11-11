.PHONY: all env generate compile up wait-db run-app run down clean

all: run-app

# 1. Variables de entorno: creacion automatica a partir de .env.example si no existe
env:
	@if [ ! -f .env ]; then echo "==> Creando .env a partir de .env.example..."; cp .env.example .env; fi

# 2. Generacion de codigo con sqlc y templ
generate: env
	@echo "==> Generando codigo con sqlc..."
	@if command -v sqlc >/dev/null 2>&1; then sqlc generate; elif [ -f "$$(go env GOPATH)/bin/sqlc" ]; then "$$(go env GOPATH)/bin/sqlc" generate; else go run github.com/sqlc-dev/sqlc/cmd/sqlc@v1.31.1 generate; fi
	@echo "==> Generando componentes con templ..."
	@if command -v templ >/dev/null 2>&1; then templ generate; elif [ -f "$$(go env GOPATH)/bin/templ" ]; then "$$(go env GOPATH)/bin/templ" generate; else go run github.com/a-h/templ/cmd/templ@v0.3.1100 generate; fi

# 3. Compilacion y dependencias
compile: env generate
	@echo "==> Verificando dependencias y compilacion..."
	@go mod tidy
	@go build -o tmp_app.bin main.go
	@rm -f tmp_app.bin tmp_app.bin.exe

# 4. Levantar base de datos PostgreSQL aislada
up:
	@echo "==> Levantando contenedor PostgreSQL..."
	@docker compose down -v --remove-orphans > /dev/null 2>&1 || true
	@docker compose up -d db

wait-db:
	@echo "==> Esperando a que PostgreSQL este listo..."
	@bash -c 'until docker compose exec -T db pg_isready -U postgres -d todolist_test > /dev/null 2>&1; do sleep 1; done; sleep 1'
	@echo "==> PostgreSQL esta listo."

# 5. Ejecutar aplicacion completa SSR (TP5)
run-app: env
	@bash run-app.sh

# 6. Levantar stack completo en Docker
run: env generate
	@echo "==> Levantando stack completo con Docker Compose..."
	@docker compose up --build

# 7. Tareas posteriores y limpieza
down:
	@echo "==> Limpiando contenedores y volumenes de Docker..."
	@docker compose down -v --remove-orphans

clean: down
	@echo "==> Limpiando binarios y archivos generados..."
	@rm -f tmp_app tmp_app.exe tmp_app.bin tmp_app.bin.exe tmp_test.bin tmp_test.bin.exe
