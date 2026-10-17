# =======================================================
# Pruebas de Integracion con Hurl para Servidor de Tareas
# =======================================================

# 1. POST /tasks - Crear una nueva tarea valida
POST http://localhost:8080/tasks
Content-Type: application/json
{
  "titulo": "Preparar presentacion del TP3",
  "descripcion": "Armar las diapositivas y verificar los endpoints con Hurl",
  "prioridad": "Alta",
  "vencimiento": "2026-12-31T23:59:59Z"
}
HTTP 201
[Captures]
task_id: jsonpath "$.id"
[Asserts]
jsonpath "$.id" > 0
jsonpath "$.titulo" == "Preparar presentacion del TP3"
jsonpath "$.estado" == "Pendiente"
jsonpath "$.prioridad" == "Alta"

# 2. POST /tasks - Error de validacion (Titulo vacio)
POST http://localhost:8080/tasks
Content-Type: application/json
{
  "titulo": "",
  "descripcion": "Esta prueba debe fallar por titulo vacio"
}
HTTP 400

# 3. GET /tasks - Listar todas las tareas
GET http://localhost:8080/tasks
HTTP 200
[Asserts]
header "Content-Type" contains "application/json"
jsonpath "$" count >= 1

# 4. GET /tasks/{id} - Obtener la tarea creada por ID
GET http://localhost:8080/tasks/{{task_id}}
HTTP 200
[Asserts]
jsonpath "$.id" == {{task_id}}
jsonpath "$.titulo" == "Preparar presentacion del TP3"

# 5. GET /tasks/{id} - Intentar obtener una tarea inexistente
GET http://localhost:8080/tasks/999999
HTTP 404

# 6. GET /tasks/{id} - Intentar obtener con formato de ID invalido
GET http://localhost:8080/tasks/invalido
HTTP 400

# 7. PUT /tasks/{id} - Actualizar la tarea
PUT http://localhost:8080/tasks/{{task_id}}
Content-Type: application/json
{
  "titulo": "Preparar presentacion del TP3 (Finalizado)",
  "descripcion": "Diapositivas y pruebas completadas con exito",
  "estado": "En Progreso",
  "prioridad": "Alta"
}
HTTP 200
[Asserts]
jsonpath "$.id" == {{task_id}}
jsonpath "$.titulo" == "Preparar presentacion del TP3 (Finalizado)"
jsonpath "$.estado" == "En Progreso"
jsonpath "$.prioridad" == "Alta"

# 8. PUT /tasks/{id} - Intentar actualizar una tarea inexistente
PUT http://localhost:8080/tasks/999999
Content-Type: application/json
{
  "titulo": "Tarea que no existe",
  "estado": "Pendiente",
  "prioridad": "Media"
}
HTTP 404

# 9. POST /tasks - Crear una segunda tarea para pruebas avanzadas
POST http://localhost:8080/tasks
Content-Type: application/json
{
  "titulo": "Redactar documentacion de API",
  "descripcion": "Documentar todos los endpoints en el README.md",
  "prioridad": "Baja",
  "vencimiento": "2026-11-15T12:00:00Z"
}
HTTP 201
[Captures]
task_id_2: jsonpath "$.id"
[Asserts]
jsonpath "$.id" > 0

# 10. GET /tasks?prioridad=Alta - Filtrar por prioridad
GET http://localhost:8080/tasks?prioridad=Alta
HTTP 200
[Asserts]
header "Content-Type" contains "application/json"
jsonpath "$" count >= 1

# 11. GET /tasks?q=documentacion - Busqueda de palabra clave
GET http://localhost:8080/tasks?q=documentacion
HTTP 200
[Asserts]
header "Content-Type" contains "application/json"
jsonpath "$" count >= 1

# 12. GET /tasks?orden=vencimiento - Listar pendientes ordenadas por vencimiento
GET http://localhost:8080/tasks?orden=vencimiento
HTTP 200
[Asserts]
header "Content-Type" contains "application/json"

# 13. GET /tasks/metrics - Obtener resumen de metricas
GET http://localhost:8080/tasks/metrics
HTTP 200
[Asserts]
header "Content-Type" contains "application/json"
jsonpath "$.total" >= 2

# 14. GET /tasks/overdue - Listar tareas vencidas
GET http://localhost:8080/tasks/overdue
HTTP 200
[Asserts]
header "Content-Type" contains "application/json"

# 15. PATCH /tasks/{id} - Accion rapida: marcar como completada
PATCH http://localhost:8080/tasks/{{task_id}}
HTTP 200
[Asserts]
jsonpath "$.id" == {{task_id}}
jsonpath "$.estado" == "Completada"

# 16. DELETE /tasks/{id} - Eliminar tarea 2
DELETE http://localhost:8080/tasks/{{task_id_2}}
HTTP 204

# 17. GET /tasks/{id} - Confirmar que tarea 2 fue eliminada
GET http://localhost:8080/tasks/{{task_id_2}}
HTTP 404

# 18. DELETE /tasks?estado=Completada - Borrado en lote de completadas
DELETE http://localhost:8080/tasks?estado=Completada
HTTP 200
[Asserts]
header "Content-Type" contains "application/json"
jsonpath "$.mensaje" == "tareas completadas eliminadas"

# 19. DELETE /tasks/{id} - Intentar eliminar una tarea inexistente
DELETE http://localhost:8080/tasks/999999
HTTP 404