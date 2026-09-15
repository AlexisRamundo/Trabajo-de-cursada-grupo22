# Trabajo de Cursada — Entrega 2: Persistiendo el Dominio

## App de Gestión de Tareas (To-Do List)

Segunda entrega del Trabajo Práctico de cursada para la materia **Programación Web**. En esta etapa se establece la capa de persistencia permanente para la aplicación definida en la Entrega 1, utilizando una base de datos relacional **PostgreSQL**, consultas SQL tipadas y la herramienta de generación de código **sqlc**.

> **Estrategia de ramas Git:**
>
> - Rama `tp1`: Contiene la Entrega 1 inicial (definición del dominio y servidor HTTP nativo).
> - Rama `tp2`: Contiene la Entrega 2 con toda la infraestructura de persistencia, consultas SQL, código generado por sqlc, tests de integración y automatización con Docker.

---

## 1. Diseño de la Base de Datos y Persistencia

Continuando con el dominio de **Gestión de Tareas (_To-Do List_)**, se definió la tabla principal `tasks` en PostgreSQL.

### Modelo Relacional (`schema.sql` / `db/schema/schema.sql`)

### Justificación de Atributos y Tipos de Datos:

- **`id` (`SERIAL PRIMARY KEY`):** Identificador numérico único autoincremental de 32 bits (mapeado a `int32` en Go).
- **`titulo` (`VARCHAR(255) NOT NULL`):** Nombre o título obligatorio de la tarea.
- **`descripcion` (`TEXT NOT NULL DEFAULT ''`):** Descripción detallada de la tarea; admite texto sin límite rígido.
- **`estado` (`VARCHAR(50) NOT NULL DEFAULT 'Pendiente'`):** Estado del ciclo de vida de la tarea (_"Pendiente"_, _"En Progreso"_, _"Completada"_).
- **`prioridad` (`VARCHAR(50) NOT NULL DEFAULT 'Media'`):** Nivel de urgencia (_"Baja"_, _"Media"_, _"Alta"_).
- **`creado_en` (`TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP`):** Marca temporal con zona horaria generada automáticamente por el motor de base de datos.
- **`vencimiento` (`TIMESTAMP WITH TIME ZONE`):** Fecha límite opcional/nullable (mapeada a `sql.NullTime` en Go).

---

## 2. Definición de Consultas CRUD (`queries.sql` / `db/queries/queries.sql`)

Se escribieron las consultas SQL para las operaciones CRUD fundamentales utilizando las anotaciones requeridas por **sqlc**:

- **Crear registro (`CreateTask`):**
- **Obtener por ID (`GetTaskByID`):**
- **Listar registros (`ListTasks`):**
- **Actualizar registro (`UpdateTask`):**
- **Eliminar registro (`DeleteTask`):**

### Consultas Avanzadas:

- **1. Ordenar por estado (Pendiente) y fecha de vencimiento (`ListPendingTasksOrderByDueDate`):**
- **2. Filtrado por prioridad (`ListTasksByPriority`):**
- **3. Tareas completadas en los últimos X días con conteo total (`ListCompletedTasksInLastDays`):**

- **4. Tareas que vencen en los siguientes X días (`ListTasksDueInNextDays`):**
- **5. Búsqueda global de palabra clave en título y descripción (`SearchTasks`):**
- **6. Métricas y estadísticas de tareas para dashboard (`GetTaskMetrics`):**
- **7. Detección de tareas vencidas atrasadas (`ListOverdueTasks`):**
- **8. Acción rápida: Marcar tarea como completada (`MarkTaskCompleted`):**
- **9. Limpieza en lote de tareas completadas (`DeleteCompletedTasks`):**

### Ventajas del enfoque con `sqlc`:

1. **Seguridad estricta contra inyecciones SQL:** Todas las consultas son parametrizadas (`$1`, `$2`, etc.).
2. **Tipado estático en tiempo de compilación:** `sqlc` valida el esquema contra las consultas y genera structs e interfaces Go fuertemente tipadas.
3. **Sin sobrecarga en tiempo de ejecución:** A diferencia de un ORM pesado, `sqlc` genera código nativo con `database/sql` sin reflection en runtime.

---

## 3. Generación de Código con `sqlc`

El archivo de configuración [`sqlc.yaml`](./sqlc.yaml) define el motor PostgreSQL y las rutas de entrada/salida:

```yaml
version: "2"
sql:
  - engine: "postgresql"
    schema: "db/schema/schema.sql"
    queries: "db/queries/queries.sql"
    gen:
      go:
        package: "db"
        out: "db/sqlc"
        sql_package: "database/sql"
```

Al ejecutar `sqlc generate`, se producen en `db/sqlc/`:

- `models.go`: Estructura Go `Task` representativa de la tabla.
- `queries.sql.go`: Métodos Go (`CreateTask`, `GetTaskByID`, `ListTasks`, `UpdateTask`, `DeleteTask`) listos para usar con `*sql.DB` o transacciones.
- `db.go`: Constructor `New(db)` e interfaz común `DBTX`.

---

## 4. Estructura del Proyecto

```text
├── .gitignore               # Exclusión de binarios, archivos temporales y entornos
├── docker-compose.yml       # Contenedor de PostgreSQL aislado con healthcheck
├── Makefile                 # Automatización de tareas (test, generate, up, down)
├── test.sh                  # Script Bash con ciclo de vida completo de pruebas
├── schema.sql               # Esquema DDL en la raíz
├── queries.sql              # Consultas CRUD con anotaciones sqlc en la raíz
├── sqlc.yaml                # Configuración de sqlc
├── README.md                # Documentación de la 2da Entrega
├── go.mod                   # Definición del módulo y dependencias Go
├── go.sum                   # Checksums de dependencias Go
├── main.go                  # Servidor web (TP1)
├── index.html               # Frontend básico (TP1)
└── db/
    ├── schema/
    │   └── schema.sql       # Esquema DDL de PostgreSQL
    ├── queries/
    │   └── queries.sql      # Consultas SQL CRUD
    └── sqlc/
        ├── db.go            # Inicialización de queries generado por sqlc
        ├── models.go        # Structs Go de modelos generados
        ├── queries.sql.go   # Implementación de métodos CRUD generados
        └── tasks_test.go    # Suite de pruebas unitarias/integración
```

---

## 5. Requisitos Previos

Para ejecutar y probar este proyecto se requiere contar con:

- **[Go](https://go.dev/dl/)** (versión 1.22 o superior).
- **[Docker](https://docs.docker.com/get-docker/)** y **Docker Compose**.
- **Bash** (disponible de forma nativa en Linux/macOS, o mediante Git Bash en Windows) o **Make**.

---

## 6. Ejecución de Tests y Automatización

Para probar el proyecto tras clonar el repositorio y posicionarse en la rama `tp2`:

```bash
git checkout tp2
```

Se provee un script bash y un `Makefile` que realizan **todas las tareas necesarias de forma totalmente desatendida**:

### Opción A (Recomendada con Bash):

```bash
./test.sh
```

_(o también: `bash test.sh`)_

### Opción B (Con Make):

```bash
make test
```

---

## 7. Fases de la Automatización (`test.sh` / `Makefile`)

El script coordina de manera automatizada las tres etapas requeridas:

1. **Tareas Previas:**
   - Limpieza preventiva de contenedores o volúmenes huérfanos anteriores (`docker compose down -v`).
   - Ejecución de `sqlc generate` para regenerar el código Go a partir del esquema y consultas.
   - Verificación de dependencias y compilación Go (`go mod tidy`, `go test -c`).
   - Levantamiento del contenedor de PostgreSQL mediante Docker Compose (`docker compose up -d db`).
   - Espera activa (_polling_ con `pg_isready`) hasta que el motor de PostgreSQL esté listo para aceptar conexiones e inicialice la tabla `tasks`.

2. **Ejecución de Tests:**
   - Ejecución de las pruebas con `go test -v -count=1 ./...`.
   - Se corre la prueba integral `TestQueries_CRUD` (`db/sqlc/tasks_test.go`) que valida:
     1. `CreateTask`: Creación de registro y aserción de ID autoincremental, título, prioridad y fechas.
     2. `GetTaskByID`: Lectura por ID y comparación de atributos.
     3. `UpdateTask`: Modificación de campos y verificación de persistencia real en la base.
     4. `ListTasks`: Comprobación de que la tarea actualizada se encuentra en la lista general.
     5. `DeleteTask`: Eliminación del registro y validación de que consultas posteriores retornen `sql.ErrNoRows`.
   - Se ejecuta además las `TestQueries_Avanzadas` (estructurada con el patrón _Table-Driven Tests_ de Go) que verifica:
     1. `OrderByStatusAndDueDate`: Listado de tareas pendientes ordenadas cronológicamente por vencimiento y exclusión de completadas.
     2. `FilterByPriority`: Filtrado exclusivo por prioridad (ej. "Alta") y manejo de prioridades inexistentes.
     3. `CompletedTasksInLastDays`: Filtrado por estado completado dentro de una ventana temporal relativa de días, retornando las entidades junto al conteo numérico total del período (`COUNT(*) OVER()`).
     4. `TasksDueInNextDays`: Identificación de tareas con vencimiento próximo en los siguientes X días ordenadas de forma ascendente.
     5. `SearchTasks`: Búsqueda global insensible a mayúsculas/minúsculas (`ILIKE`) tanto en título como en descripción.
     6. `GetTaskMetrics`: Cálculo de métricas y contadores de estado y vencidas para dashboard mediante agregaciones con `FILTER`.
     7. `ListOverdueTasks`: Detección precisa de tareas con vencimiento superado que continúan sin completar.
     8. `MarkTaskCompleted`: Transición de estado rápida a "Completada" y verificación de persistencia en disco.
     9. `DeleteCompletedTasks`: Eliminación masiva de tareas finalizadas comprobando filas afectadas.

3. **Tareas Posteriores:**
   - Registro de `trap` en Bash que asegura la detención y borrado total de contenedores y volúmenes de Docker (`docker compose down -v --remove-orphans`) tanto si las pruebas finalizan con éxito como si ocurre algún fallo.
