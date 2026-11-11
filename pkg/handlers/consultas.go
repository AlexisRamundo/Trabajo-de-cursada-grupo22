package handlers

// consultas.go (TP3) agrupa las consultas avanzadas de la API:
//   - filtros y búsquedas sobre GET /tasks
//   - GET /tasks/metrics (resumen estadístico)
//   - GET /tasks/overdue (tareas vencidas)
// Las operaciones CRUD y las funciones auxiliares están en CRUD.go.

import (
	"encoding/json"
	"net/http"
	"strconv"

	db "servidor/db/sqlc"
	"servidor/pkg/models"
)

// filterTasks resuelve GET /tasks cuando la URL trae un parámetro de filtro o búsqueda.
// Devuelve true si atendió la petición (con éxito o con error) y false si la URL no trae ningun filtro
func (h *Handler) filterTasks(w http.ResponseWriter, r *http.Request) bool {
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
			return true
		}
		tasks, err = h.queries.ListTasksDueInNextDays(r.Context(), int32(dias))
	case q.Get("completadas_ultimos_dias") != "":
		// Tareas completadas en los últimos X días
		h.completedInLastDays(w, r, q.Get("completadas_ultimos_dias"))
		return true
	default:
		// Sin filtros: no se atiende acá
		return false
	}

	if err != nil {
		internalError(w, err)
		return true
	}
	writeTasks(w, tasks)
	return true
}

// GET /tasks?completadas_ultimos_dias=X - Tareas completadas en los últimos X días.
// Va aparte porque la consulta devuelve un tipo propio (ListCompletedTasksInLastDaysRow)
// con una columna extra, en lugar de db.Task.
func (h *Handler) completedInLastDays(w http.ResponseWriter, r *http.Request, param string) {
	dias, parseErr := strconv.Atoi(param)
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
	writeTasks(w, tasks)
}