# Trabajo de Cursada — Entrega 3: Exponiendo la Lógica como API

## App de Gestión de Tareas (To-Do List)

Tercera entrega del TP de cursada. En esta etapa se construye y expone la **API REST** de la aplicación, conectando los endpoints HTTP con la capa de datos generada por **sqlc** e implementando las validaciones y reglas del dominio en Go.

> **Organización de ramas Git:**
>
> - Rama `tp1`: Entrega 1 inicial (definición del dominio y servidor HTTP básico).
> - Rama `tp2`: Entrega 2 (persistencia en PostgreSQL, sqlc, pruebas de base de datos y automatización).
> - Rama `tp3`: Entrega 3 (API REST completa con endpoints CRUD y filtros, validaciones, middleware, contenedorización Docker multi-stage y pruebas con Hurl).

---

## 1. Arquitectura de la aplicación (Three-Tier)

El proyecto (que ahora crecio bastante) sigue una separación clara de responsabilidades en tres capas:

1. **Capa de Presentación / API REST (`main.go`, `pkg/handlers/`):**  
   Maneja el protocolo HTTP, enrutamiento, deserialización de peticiones JSON, invocación de la lógica y envío de respuestas tipadas con códigos de estado HTTP (`200`, `201`, `204`, `400`, `404`, `405`, `500`).
2. **Capa de Lógica del Negocio (`pkg/models/`):**  
   Define la entidad `Task` y funciones puras desacopladas de la web (`ValidateTask`, `ApplyDefaults`) que validan restricciones de negocio y valores por defecto.
3. **Capa de Datos y Persistencia (`db/sqlc/`, PostgreSQL):**  
   Acceso la base de datos generado por `sqlc`, transacciones y ejecución de consultas SQL parametrizadas sin riesgo de inyecciones SQL.

---

## 2. Endpoints de la API REST

Todos los endpoints reciben y emiten datos en formato JSON (`Content-Type: application/json`). La API incluye además soporte completo para **CORS** (`Access-Control-Allow-Origin: *`, `OPTIONS preflight`) facilitando el consumo directo desde interfaces web en el navegador (para su consumo en el TP4).

### Endpoints CRUD principales

| Método   | Endpoint                   | Descripción                                              | Código Éxito     | Códigos de Error                   |
| :------- | :------------------------- | :------------------------------------------------------- | :--------------- | :--------------------------------- |
| `POST`   | `/tasks`                   | Crear una nueva tarea con validación de datos            | `201 Created`    | `400 Bad Request`                  |
| `GET`    | `/tasks`                   | Listar tareas (con soporte de filtros por query params)  | `200 OK`         | `500 Internal Server Error`        |
| `GET`    | `/tasks/{id}`              | Obtener el detalle de una tarea por su ID                | `200 OK`         | `400 Bad Request`, `404 Not Found` |
| `PUT`    | `/tasks/{id}`              | Actualizar todos los campos de una tarea existente       | `200 OK`         | `400 Bad Request`, `404 Not Found` |
| `PATCH`  | `/tasks/{id}`              | Acción rápida: marcar tarea directamente como Completada | `200 OK`         | `400 Bad Request`, `404 Not Found` |
| `DELETE` | `/tasks/{id}`              | Eliminar una tarea por su ID                             | `204 No Content` | `400 Bad Request`, `404 Not Found` |
| `DELETE` | `/tasks?estado=Completada` | Limpieza masiva en lote de todas las tareas completadas  | `200 OK`         | `400 Bad Request`                  |
| `GET`    | `/health`                  | Chequeo de salud del servicio (liveness/readiness)       | `200 OK`         | —                                  |

### Endpoints especializados y métricas (Dashboard)

| Método | Endpoint         | Descripción                                                     | Código Éxito |
| :----- | :--------------- | :-------------------------------------------------------------- | :----------- |
| `GET`  | `/tasks/metrics` | Resumen: total, pendientes, en progreso, completadas y vencidas | `200 OK`     |
| `GET`  | `/tasks/overdue` | Listar tareas vencidas (`vencimiento < NOW()` y no completadas) | `200 OK`     |

### Parámetros de Consulta (Query Params) en `GET /tasks`

El endpoint `GET /tasks` permite filtrar, buscar y ordenar las tareas para alimentar vistas dinámicas del frontend:

- `GET /tasks?q=palabra`: Búsqueda textual por palabra clave en título y descripción (`ILIKE`).
- `GET /tasks?prioridad=Alta`: Filtra tareas por prioridad (`Alta`, `Media`, `Baja`).
- `GET /tasks?orden=vencimiento`: Retorna tareas pendientes ordenadas ascendentemente por fecha de vencimiento más próxima.
- `GET /tasks?vence_en=7`: Lista tareas que vencen en los próximos _N_ días.
- `GET /tasks?completadas_ultimos_dias=30`: Lista tareas completadas dentro de la ventana de los últimos _N_ días.
- `GET /tasks` (sin parámetros): Retorna todas las tareas ordenadas por ID.

---

## 3. Validaciones de Dominio y Middleware

- **Validaciones (`pkg/models/models.go`):**
  - El título es obligatorio y no puede superar los 255 caracteres.
  - El estado debe ser uno de: `"Pendiente"`, `"En Progreso"`, `"Completada"`.
  - La prioridad debe ser una de: `"Baja"`, `"Media"`, `"Alta"`.
  - `ApplyDefaults`: Asigna `"Pendiente"` y `"Media"` automáticamente si no se especifican.
- **Middleware de Observabilidad (`main.go`):**
  - Implementa el patrón `loggingMiddleware(next http.Handler)` enseñado en la cátedra.
  - Registra cada petición entrante con timestamp, método, URL, IP remota y duración de procesamiento.

---

## 4. Estructura del Proyecto

```text
├── .dockerignore            # Exclusión de archivos para el contexto de build Docker
├── .gitignore               # Exclusión de binarios, scripts locales y temporales
├── Dockerfile               # Multi-stage build optimizado (golang:alpine -> alpine)
├── docker-compose.yml       # Stack completo: PostgreSQL (db) + API REST (api)
├── Makefile                 # Automatización de tareas (test-api, test, run, clean)
├── test-api.sh              # Script Bash para probar la API (TP3)
├── test.sh                  # Script Bash para suite de pruebas unitarias (TP2)
├── requests.hurl            # Archivo de pruebas reproducibles con HURL
├── schema.sql               # Esquema DDL de PostgreSQL
├── queries.sql              # Consultas SQL con anotaciones sqlc
├── sqlc.yaml                # Configuración de sqlc
├── README.md                # Documentación de la Entrega 3
├── go.mod                   # Definición del módulo y dependencias Go
├── go.sum                   # Checksums de dependencias Go
├── index.html               # Frontend básico (TP1)
├── main.go                  # Servidor web (TP1). Punto de entrada del servidor HTTP y middleware
├── pkg/
│   ├── handlers/
│   │   └── handler.go       # Handlers HTTP CRUD y queries
│   └── models/
│       ├── models.go        # Entidades del dominio, defaults y validaciones
│       └── models_test.go   # Pruebas unitarias de la lógica pura (Table-Driven)
└── db/
    ├── schema/ schema.sql    # DDL relacional de tasks
    ├── queries/ queries.sql  # Consultas CRUD y avanzadas
    └── sqlc/ tasks_test.go   # Suite de pruebas unitarias/integración
```

---

## 5. Requisitos previos

Para ejecutar y probar el proyecto se requiere:

- **[Docker](https://docs.docker.com/get-docker/)** y **Docker Compose**.
- **[Go](https://go.dev/dl/)** (versión 1.22 o superior).
- **Bash** para ejecutar los scripts (disponible en Linux/macOS y en Git Bash para Windows) o **Make**.
- **[Hurl](https://hurl.dev/)** para ejecutar directamente el archivo `requests.hurl`.

---

## 6. Ejecución de pruebas de la API (Consigna TP3)

Para probar el proyecto tras clonar el repositorio y posicionarse en la rama `tp3`:

```bash
git checkout tp3
```

Se provee el script `test-api.sh` y el `Makefile` que realizan **todas las tareas de forma 100% desatendida**:

### Opción A (Recomendada con Bash):

```bash
./test-api.sh
```

_(o también: `bash test-api.sh`)_

### Opción B (Con Make):

```bash
make test-api
```

---

## 7. Fases de la automatización (`test-api.sh`)

El script coordina automáticamente las tres etapas:

1. **Tareas Previas:**
   - Limpieza preventiva de contenedores o volúmenes huérfanos anteriores (`docker compose down -v`).
   - Ejecución de `sqlc generate` para asegurar que el código Go esté sincronizado con el esquema.
   - Verificación de dependencias y compilación del ejecutable (`go mod tidy`, `go build`).
   - Levantamiento del contenedor de base de datos PostgreSQL (`docker compose up -d db`).
   - Espera activa (_polling_ con `pg_isready`) hasta que PostgreSQL acepte conexiones.
   - Levantamiento del servidor API en segundo plano y verificación de liveness (`/health`).

2. **Prueba de cada uno de los Endpoints:**
   - Si **Hurl** está instalado en el sistema, ejecuta `hurl --test requests.hurl`.
   - Se valida:
     1. `POST /tasks` (Crear tarea válida): verifica HTTP `201` y captura el `id` asignado.
     2. `POST /tasks` (Título vacío): verifica HTTP `400` por error de validación.
     3. `GET /tasks` (Listar tareas): verifica HTTP `200` y arreglo JSON no nulo.
     4. `GET /tasks/{id}` (Obtener por ID): verifica HTTP `200` y concordancia de datos.
     5. `GET /tasks/999999` (ID inexistente): verifica HTTP `404`.
     6. `GET /tasks/invalido` (ID alfanumérico): verifica HTTP `400`.
     7. `PUT /tasks/{id}` (Actualizar tarea): verifica HTTP `200` y persistencia de cambios.
     8. `PUT /tasks/999999` (Actualizar ID inexistente): verifica HTTP `404`.
     9. `POST /tasks` (Crear segunda tarea para pruebas avanzadas): verifica HTTP `201`.
     10. `GET /tasks?prioridad=Alta` (Filtrar por prioridad): verifica HTTP `200`.
     11. `GET /tasks?q=documentacion` (Búsqueda por palabra clave): verifica HTTP `200`.
     12. `GET /tasks?orden=vencimiento` (Ordenar pendientes por vencimiento): verifica HTTP `200`.
     13. `GET /tasks/metrics` (Dashboard de métricas): verifica HTTP `200` y conteos numéricos.
     14. `GET /tasks/overdue` (Listado de tareas vencidas): verifica HTTP `200`.
     15. `PATCH /tasks/{id}` (Acción rápida: marcar completada): verifica HTTP `200`.
     16. `DELETE /tasks/{id}` (Eliminar tarea 2): verifica HTTP `204`.
     17. `GET /tasks/{id}` (Verificar borrado de tarea 2): verifica HTTP `404`.
     18. `DELETE /tasks?estado=Completada` (Borrado masivo en lote): verifica HTTP `200`.
     19. `DELETE /tasks/999999` (Eliminar ID inexistente): verifica HTTP `404`.

3. **Tareas Posteriores:**
   - Mediante `trap cleanup EXIT`, detiene el proceso del servidor API y destruye los contenedores y volúmenes de Docker (`docker compose down -v --remove-orphans`), garantizando un entorno limpio tanto si las pruebas finalizan con éxito como ante cualquier error.

---

## 8. Ejecución del Servidor en Desarrollo

### Con Docker Compose (Stack Completo):

```bash
docker compose up --build
```

_(o con `make run`)_

### De forma local:

```bash
# 1. Levantar base de datos
docker compose up -d db

# 2. Iniciar servidor Go
go run main.go
```
