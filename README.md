# Trabajo de Cursada — Entrega 5: Renderizado del Lado del Servidor con Templ

## App de Gestión de Tareas (To-Do List)

Seguimos desarrollando el TP de cursada y continuamos con la entrega, esta vez con la **parte 5 (TP5)**. En esta etapa se adopta la arquitectura de **Renderizado del Lado del Servidor (Server-Side Rendering - SSR)** integrando la vista directamente en el servidor Go mediante **Templ** y dandole estilo con **Pico.css**.

> **Organización de ramas Git:**
>
> - Rama `tp1`: Entrega 1 (Definición del dominio y servidor HTTP básico).
> - Rama `tp2`: Entrega 2 (Persistencia con PostgreSQL, sqlc, pruebas de base de datos y automatización).
> - Rama `tp3`: Entrega 3 (API REST completa con endpoints CRUD y filtros, validaciones, middleware, contenedorización Docker multi-stage y pruebas con Hurl)
> - **Rama `tp5`**: Entrega 5 (Arquitectura SSR con componentes Templ, modularización en `pkg/handlers/`, formulario clásico con patrón PRG, estilos Pico.css y ejecución automatizada).

---

## 1. Arquitectura de la aplicación (SSR con Go + Templ)

El servidor Go asume la responsabilidad completa de construir la interfaz de usuario (HTML) y entregarla al cliente:

**Capa de vistas / componentes (`views/`):**

- **`layout.templ`**: Define la estructura base HTML5 (`<html>`, `<head>`, `<body>`), enlazando la hoja de estilos de **Pico.css** vía CDN y definiendo el contenedor principal semántico mediante `{ children... }`.
- **`entity_form.templ`**: Componente `EntityForm()` que renderiza el formulario para la creación de tareas (`POST /tasks`) con validaciones HTML nativas y controles de fecha/hora.
- **`entity_list.templ`**: Componente `EntityList(tasks []db.Task)` que recibe las entidades directamente de la base de datos y renderiza una tabla con formato y estados estilizados, contenida dentro de `<div id="tasks-container">`.
- **`index.templ`**: Componente de página principal `IndexPage(tasks []db.Task)` que compone el `Layout`, encabezado, formulario y listado.

**Capa de controladores y handlers (`pkg/handlers/` y `main.go`):**

- **`pkg/handlers/ssr.go`**: Métodos asociados al struct `Handler` para la entrega SSR:
  - `IndexHandler (GET /)`: Consulta la base de datos mediante `h.queries.ListTasks(ctx)` y renderiza el componente `views.IndexPage(tasks)`.
  - `CreateTaskHandler (POST /tasks)`: Parsea los datos del formulario (`r.ParseForm()`), aplica las reglas de negocio de `pkg/models` (`ApplyDefaults` completa los valores por defecto y `ValidateTask` valida título, estado y prioridad), guarda el nuevo registro con `h.queries.CreateTask(ctx, ...)` e implementa el patrón **Post/Redirect/Get (PRG)** redirigiendo a `/` para refrescar la lista evitando reenvíos duplicados.
- **`pkg/handlers/CRUD.go` y `pkg/handlers/consultas.go`**: Mantenidos en el paquete para preservar el histórico de controladores de persistencia y consultas (TP anteriores).
- **`main.go`**: Responsable de la inicialización y carga de variables desde `.env`, apertura y verificación de conexión a PostgreSQL, inyección de dependencias (`handlers.New(queries)`), registro de rutas en `http.ServeMux` e inicio del servidor HTTP con middleware de logging.

**Capa de Datos y Persistencia (`db/sqlc/`, PostgreSQL):**

- Repositorio generado por **sqlc**, y consultas parametrizadas sin riesgo de inyecciones SQL.

**Variables de entorno (`.env.example` y automatización de `.env`):**

- Se provee una plantilla de configuración `.env.example`.
- Tanto `run-app.sh`, el `Makefile` como la propia aplicación Go (`main.go`) **detectan automáticamente** si `.env` existe; si no está presente, **lo generan automáticamente a partir de `.env.example`**.

---

## 2. Flujo de Navegación: Patrón Post/Redirect/Get (PRG)

```
[ Navegador ] ---- GET / -----------------------------------> [ Handler.IndexHandler ]
             <--- 200 OK (HTML renderizado con Templ) ------- (Consulta DB)

[ Formulario ] --- POST /tasks (datos del form) ------------> [ Handler.CreateTaskHandler ]
                                                              (Guarda en DB)
             <--- 303 See Other (Location: /) --------------

[ Navegador ] ---- GET / -----------------------------------> [ Handler.IndexHandler ]
             <--- 200 OK (HTML con la lista actualizada) ---- (Consulta DB)
```

---

## 3. Estructura del Proyecto

```text
├── .dockerignore            # Exclusión de binarios y temporales en Docker
├── .env.example             # Plantilla base para variables de entorno (PORT, DATABASE_URL)
├── .gitignore               # Exclusión de binarios, scripts locales y temporales
├── Dockerfile               # Multi-stage build optimizado (golang:alpine -> alpine)
├── docker-compose.yml       # Stack completo: PostgreSQL (db) + aplicación SSR (app)
├── requests.hurl            # Archivo de pruebas reproducibles con HURL (TP3)
├── Makefile                 # Automatización de tareas
├── run-app.sh               # Script Bash para ejecutar la aplicación (TP5)
├── schema.sql               # Esquema DDL de PostgreSQL (TP2)
├── queries.sql              # Consultas SQL con anotaciones sqlc (TP2)
├── sqlc.yaml                # Configuración de sqlc (TP2)
├── README.md                # Documentación de la Entrega 5
├── go.mod                   # Definición del módulo y dependencias Go
├── go.sum                   # Checksums de dependencias Go
├── index.html               # Frontend básico (TP1)
├── main.go                  # Bootstrap del servidor, conexión DB y registro de rutas
├── pkg/
│   ├── handlers/
│   │   ├── ssr.go           # Handlers SSR con Templ (IndexHandler y CreateTaskHandler)
│   │   ├── CRUD.go          # Handlers CRUD (TP3)
│   │   └── consultas.go     # Consultas avanzadas (TP3)
│   └── models/
│       ├── models.go        # Modelos del dominio (TP3)
│       └── models_test.go   # Pruebas unitarias de modelos (TP3)
├── views/                   # Componentes de interfaz de usuario con Templ
│   ├── layout.templ         # Estructura base HTML con link a Pico.css
│   ├── entity_form.templ    # Formulario para crear tareas
│   ├── entity_list.templ    # Tabla de listado de tareas
│   └── index.templ          # Vista principal IndexPage
└── db/
    ├── schema/schema.sql    # DDL relacional de tasks (TP2)
    ├── queries/queries.sql  # Consultas CRUD y avanzadas (TP2)
    └── sqlc/                # Repositorio Go generado en tiempo de compilación
         └── tasks_test.go   # Suite de pruebas unitarias/integración (TP2)

```

---

## 4. Requisitos Previos

- **[Docker](https://docs.docker.com/get-docker/)** y **Docker Compose**.
- **[Go](https://go.dev/dl/)** (versión 1.22 o superior).
- **Bash** (disponible en Linux/macOS y Git Bash en Windows) o **Make**.
- _(Opcional)_ Herramientas `templ` y `sqlc` CLI. Si no están en `PATH`, los scripts las ejecutan automáticamente mediante `go run`.

---

## 5. Ejecución de la Aplicación (TP5)

Para probar el proyecto tras clonar el repositorio y situarse en la rama `tp5`:

```bash
git checkout tp5
```

Se provee el script `run-app.sh` y el `Makefile` que realizan **todas las tareas de forma 100% automatizada y desatendida**:

### Opción A (Recomendada con Bash)

```bash
./run-app.sh
```

_(o también: `bash run-app.sh`)_

### Opción B (Con Make)

```bash
make run-app
```

Una vez iniciado, abre tu navegador web en:
**[http://localhost:8080](http://localhost:8080)**

Para detener la aplicación, presiona `Ctrl+C` en la terminal. El script limpiará automáticamente todos los contenedores y volúmenes creados.

---

## 6. Fases de la Automatización (`run-app.sh`)

El script coordina las 3 etapas exigidas:

1. **Tareas Previas:**
   - Creación automática de `.env` a partir de `.env.example` si no existe.
   - Limpieza preventiva de contenedores o volúmenes huérfanos anteriores (`docker compose down -v`).
   - Ejecución de `sqlc generate` para sincronizar la persistencia.
   - Ejecución de `templ generate` para compilar los componentes de vista a Go.
   - Verificación de dependencias y compilación del binario del servidor (`go mod tidy`, `go build`).
   - Levantamiento del contenedor de base de datos PostgreSQL (`docker compose up -d db`).
   - Espera activa con `pg_isready` hasta que PostgreSQL acepte conexiones.

2. **Ejecución de la aplicación:**
   - Inicia el servidor SSR en el puerto configurado (por defecto `:8080`), listo para responder peticiones desde cualquier navegador.

3. **Tareas Posteriores:**
   - Mediante el manejador de señales `trap cleanup EXIT INT TERM`, al presionar `Ctrl+C` o cerrar la aplicación, detiene el proceso y borra automáticamente los contenedores y volúmenes de Docker (`docker compose down -v --remove-orphans`).

---

## 7. Ejecución con Docker Compose (Stack Completo)

Si se desea levantar todo el entorno (aplicación + base de datos) dentro de contenedores:

```bash
docker compose up --build
```

_(o también: `make run`)_
