package handlers

import (
	"database/sql"
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

	db "servidor/db/sqlc"
	"servidor/pkg/models"
)

// Handler guarda la dependencia de los handlers: el repositorio generado por sqlc.
// la concurrencia la resuelve PostgreSQL y *sql.DB es seguro para uso concurrente.
type Handler struct {
	queries *db.Queries
}

func New(queries *db.Queries) *Handler {
	return &Handler{queries: queries}
}

// Manejador para /tasks/{id}
func (h *Handler) TaskHandler(w http.ResponseWriter, r *http.Request) {
	// Extraer ID del path
	parts := strings.Split(r.URL.Path, "/")
	if len(parts) != 3 {
		http.Error(w, "URL invalida", http.StatusBadRequest)
		return
	}
	id64, err := strconv.ParseInt(parts[2], 10, 32)
	if err != nil || id64 <= 0 {
		http.Error(w, "ID de tarea invalido", http.StatusBadRequest)
		return
	}
	id := int32(id64)

	switch r.Method {
	case http.MethodGet:
		h.getTask(w, r, id)
	case http.MethodPut:
		h.updateTask(w, r, id)
	case http.MethodPatch:
		h.markTaskCompleted(w, r, id)
	case http.MethodDelete:
		h.deleteTask(w, r, id)
	default:
		http.Error(w, "Metodo no permitido", http.StatusMethodNotAllowed)
	}
}

// Manejador para /tasks
func (h *Handler) TasksHandler(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		h.getTasks(w, r)
	case http.MethodPost:
		h.createTask(w, r)
	case http.MethodDelete:
		h.deleteTasks(w, r)
	default:
		http.Error(w, "Metodo no permitido", http.StatusMethodNotAllowed)
	}
}

// GET /tasks - Listar todas las tareas con soporte para filtros y búsquedas avanzadas
func (h *Handler) getTasks(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()

	var (
		tasks []db.Task
		err   error
	)

	switch {
	case q.Get("q") != "":
		// Búsqueda por palabra clave en título o descripción
		tasks, err = h.queries.SearchTasks(r.Context(), q.Get("q"))
	case q.Get("prioridad") != "":
		// Filtrado por nivel de prioridad (Baja, Media, Alta)
		tasks, err = h.queries.ListTasksByPriority(r.Context(), q.Get("prioridad"))
	case q.Get("orden") == "vencimiento":
		// Tareas pendientes ordenadas por fecha de vencimiento próxima
		tasks, err = h.queries.ListPendingTasksOrderByDueDate(r.Context())
	case q.Get("vence_en") != "":
		// Tareas que vencen en los siguientes X días
		dias, parseErr := strconv.Atoi(q.Get("vence_en"))
		if parseErr != nil || dias < 0 {
			http.Error(w, "parametro 'vence_en' debe ser un numero de dias valido", http.StatusBadRequest)
			return
		}
		tasks, err = h.queries.ListTasksDueInNextDays(r.Context(), int32(dias))
	case q.Get("completadas_ultimos_dias") != "":
		// Tareas completadas en los últimos X días
		dias, parseErr := strconv.Atoi(q.Get("completadas_ultimos_dias"))
		if parseErr != nil || dias < 0 {
			http.Error(w, "parametro 'completadas_ultimos_dias' debe ser un numero de dias valido", http.StatusBadRequest)
			return
		}
		rows, qErr := h.queries.ListCompletedTasksInLastDays(r.Context(), int32(dias))
		if qErr != nil {
			internalError(w, qErr)
			return
		}
		result := make([]models.Task, 0, len(rows))
		for _, row := range rows {
			t := models.Task{
				ID:          row.ID,
				Titulo:      row.Titulo,
				Descripcion: row.Descripcion,
				Estado:      row.Estado,
				Prioridad:   row.Prioridad,
				CreadoEn:    row.CreadoEn,
			}
			if row.Vencimiento.Valid {
				v := row.Vencimiento.Time
				t.Vencimiento = &v
			}
			result = append(result, t)
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(result)
		return
	default:
		// Listado estándar de todas las tareas
		tasks, err = h.queries.ListTasks(r.Context())
	}

	if err != nil {
		internalError(w, err)
		return
	}

	// make(...) para que una lista vacia se devuelva como [] y no como null
	result := make([]models.Task, 0, len(tasks))
	for _, t := range tasks {
		result = append(result, toModel(t))
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(result)
}

// GET /tasks/{id} - Obtener una tarea especifica
func (h *Handler) getTask(w http.ResponseWriter, r *http.Request, id int32) {
	task, err := h.queries.GetTaskByID(r.Context(), id)
	if errors.Is(err, sql.ErrNoRows) {
		http.Error(w, "tarea no encontrada", http.StatusNotFound)
		return
	}
	if err != nil {
		internalError(w, err)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(toModel(task))
}

// POST /tasks - Crear una nueva tarea
func (h *Handler) createTask(w http.ResponseWriter, r *http.Request) {
	// Contenedor vacío para el JSON recibido
	var newTask models.Task

	// Decodificamos el JSON
	if err := json.NewDecoder(r.Body).Decode(&newTask); err != nil {
		http.Error(w, "JSON inválido: "+err.Error(), http.StatusBadRequest)
		return
	}

	// Completamos valores por defecto y validamos con la lógica de negocio
	newTask = models.ApplyDefaults(newTask)
	if err := models.ValidateTask(newTask); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Guardamos en la base de datos con el metodo generado por sqlc
	created, err := h.queries.CreateTask(r.Context(), db.CreateTaskParams{
		Titulo:      newTask.Titulo,
		Descripcion: newTask.Descripcion,
		Estado:      newTask.Estado,
		Prioridad:   newTask.Prioridad,
		Vencimiento: toNullTime(newTask.Vencimiento),
	})
	if err != nil {
		internalError(w, err)
		return
	}

	// Respondemos con el 201 y la tarea creada
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(toModel(created))
}

// PUT /tasks/{id} - Actualizar una tarea
func (h *Handler) updateTask(w http.ResponseWriter, r *http.Request, id int32) {
	var updated models.Task

	if err := json.NewDecoder(r.Body).Decode(&updated); err != nil {
		http.Error(w, "JSON inválido: "+err.Error(), http.StatusBadRequest)
		return
	}

	updated = models.ApplyDefaults(updated)
	if err := models.ValidateTask(updated); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	task, err := h.queries.UpdateTask(r.Context(), db.UpdateTaskParams{
		ID:          id,
		Titulo:      updated.Titulo,
		Descripcion: updated.Descripcion,
		Estado:      updated.Estado,
		Prioridad:   updated.Prioridad,
		Vencimiento: toNullTime(updated.Vencimiento),
	})
	if errors.Is(err, sql.ErrNoRows) {
		http.Error(w, "tarea no encontrada", http.StatusNotFound)
		return
	}
	if err != nil {
		internalError(w, err)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(toModel(task))
}

// DELETE /tasks/{id} - Eliminar una tarea
func (h *Handler) deleteTask(w http.ResponseWriter, r *http.Request, id int32) {
	// DeleteTask no falla si el id no existe, por eso buscamos primero para poder devolver 404
	_, err := h.queries.GetTaskByID(r.Context(), id)
	if errors.Is(err, sql.ErrNoRows) {
		http.Error(w, "tarea no encontrada", http.StatusNotFound)
		return
	}
	if err != nil {
		internalError(w, err)
		return
	}

	if err := h.queries.DeleteTask(r.Context(), id); err != nil {
		internalError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// GET /tasks/metrics - Obtener metricas y resumen estadistico de tareas
func (h *Handler) MetricsHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Metodo no permitido", http.StatusMethodNotAllowed)
		return
	}

	m, err := h.queries.GetTaskMetrics(r.Context())
	if err != nil {
		internalError(w, err)
		return
	}

	res := models.TaskMetrics{
		Total:       m.Total,
		Pendientes:  m.Pendientes,
		EnProgreso:  m.EnProgreso,
		Completadas: m.Completadas,
		Vencidas:    m.Vencidas,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(res)
}

// GET /tasks/overdue - Listar tareas vencidas (vencimiento < NOW() y no completadas)
func (h *Handler) OverdueHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Metodo no permitido", http.StatusMethodNotAllowed)
		return
	}

	tasks, err := h.queries.ListOverdueTasks(r.Context())
	if err != nil {
		internalError(w, err)
		return
	}

	result := make([]models.Task, 0, len(tasks))
	for _, t := range tasks {
		result = append(result, toModel(t))
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(result)
}

// PATCH /tasks/{id} - Accion rapida: marcar tarea directamente como completada
func (h *Handler) markTaskCompleted(w http.ResponseWriter, r *http.Request, id int32) {
	task, err := h.queries.MarkTaskCompleted(r.Context(), id)
	if errors.Is(err, sql.ErrNoRows) {
		http.Error(w, "tarea no encontrada", http.StatusNotFound)
		return
	}
	if err != nil {
		internalError(w, err)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(toModel(task))
}

// DELETE /tasks?estado=Completada - Limpieza en lote de tareas completadas
func (h *Handler) deleteTasks(w http.ResponseWriter, r *http.Request) {
	if r.URL.Query().Get("estado") != models.EstadoCompletada {
		http.Error(w, "Debe especificar ?estado=Completada para borrado en lote", http.StatusBadRequest)
		return
	}

	res, err := h.queries.DeleteCompletedTasks(r.Context())
	if err != nil {
		internalError(w, err)
		return
	}

	affected, _ := res.RowsAffected()
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]any{
		"mensaje":         "tareas completadas eliminadas",
		"filas_afectadas": affected,
	})
}

// Funcion para convertir el struct generado por sqlc (db.Task) en models.Task.
// Necesaria porque sql.NullTime se serializaría como {"Time":...,"Valid":...}.
func toModel(t db.Task) models.Task {
	task := models.Task{
		ID:          t.ID,
		Titulo:      t.Titulo,
		Descripcion: t.Descripcion,
		Estado:      t.Estado,
		Prioridad:   t.Prioridad,
		CreadoEn:    t.CreadoEn,
	}
	if t.Vencimiento.Valid {
		v := t.Vencimiento.Time
		task.Vencimiento = &v
	}
	return task
}

// Funcion inversa: *time.Time (nil = sin fecha) a sql.NullTime
func toNullTime(t *time.Time) sql.NullTime {
	if t == nil {
		return sql.NullTime{}
	}
	return sql.NullTime{Time: *t, Valid: true}
}

// Registra el error real en el log y responde 500 sin exponer detalles internos
func internalError(w http.ResponseWriter, err error) {
	log.Printf("error interno: %v", err)
	http.Error(w, "error interno del servidor", http.StatusInternalServerError)
}