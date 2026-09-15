-- name: CreateTask :one
INSERT INTO tasks (
    titulo,
    descripcion,
    estado,
    prioridad,
    vencimiento
) VALUES (
    $1, $2, $3, $4, $5
)
RETURNING *;

-- name: GetTaskByID :one
SELECT * FROM tasks
WHERE id = $1 LIMIT 1;

-- name: ListTasks :many
SELECT * FROM tasks
ORDER BY id ASC;

-- name: UpdateTask :one
UPDATE tasks
SET
    titulo = $2,
    descripcion = $3,
    estado = $4,
    prioridad = $5,
    vencimiento = $6
WHERE id = $1
RETURNING *;

-- name: DeleteTask :exec
DELETE FROM tasks
WHERE id = $1;

-- Ordenar por estado (pendiente) y fecha vencimiento
-- name: ListPendingTasksOrderByDueDate :many
SELECT * FROM tasks
WHERE estado = 'Pendiente'
ORDER BY vencimiento ASC NULLS LAST;


-- Filtrado por prioridad (Alta)
-- name: ListTasksByPriority :many
SELECT * FROM tasks
WHERE prioridad = $1
ORDER BY id ASC;

-- Tareas completadas en los últimos x dias con conteo total de ventana
-- name: ListCompletedTasksInLastDays :many
SELECT
    id,
    titulo,
    descripcion,
    estado,
    prioridad,
    creado_en,
    vencimiento,
    COUNT(*) OVER()::int AS total_completadas
FROM tasks
WHERE estado = 'Completada'
  AND creado_en >= NOW() - ($1::int * INTERVAL '1 day')
ORDER BY creado_en DESC;

-- Tareas que vencen en los siguientes x dias
-- name: ListTasksDueInNextDays :many
SELECT * FROM tasks
WHERE vencimiento IS NOT NULL
  AND vencimiento BETWEEN NOW() AND NOW() + ($1::int * INTERVAL '1 day')
ORDER BY vencimiento ASC;

-- Búsqueda global de palabra clave en título y descripción
-- name: SearchTasks :many
SELECT * FROM tasks
WHERE titulo ILIKE '%' || $1::text || '%'
   OR descripcion ILIKE '%' || $1::text || '%'
ORDER BY id ASC;

-- Métricas y estadísticas de tareas (cantidad de tareas por estado, pendientes, vencidas, etc)
-- name: GetTaskMetrics :one
SELECT
    COUNT(*)::int AS total,
    COUNT(*) FILTER (WHERE estado = 'Pendiente')::int AS pendientes,
    COUNT(*) FILTER (WHERE estado = 'En Progreso')::int AS en_progreso,
    COUNT(*) FILTER (WHERE estado = 'Completada')::int AS completadas,
    COUNT(*) FILTER (WHERE vencimiento < NOW() AND estado != 'Completada')::int AS vencidas
FROM tasks;

-- Detección de tareas vencidas
-- name: ListOverdueTasks :many
SELECT * FROM tasks
WHERE vencimiento IS NOT NULL
  AND vencimiento < NOW()
  AND estado != 'Completada'
ORDER BY vencimiento ASC;

-- Acción rápida: Marcar tarea como completada
-- name: MarkTaskCompleted :one
UPDATE tasks
SET estado = 'Completada'
WHERE id = $1
RETURNING *;

-- Limpieza en lote de tareas completadas
-- name: DeleteCompletedTasks :execresult
DELETE FROM tasks
WHERE estado = 'Completada';