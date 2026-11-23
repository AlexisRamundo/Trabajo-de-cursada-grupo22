package handlers

// ssr.go maneja los handlers de renderizado del lado del servidor (SSR) con HTMX

// Capa UI de la arquitectura Onion: solo traduce HTTP <-> casos de uso.
// Lee el formulario y parámetros HTTP, delega en los casos de uso de pkg/application y convierte las entidades del dominio a DTOs
// de la vista (views.TaskDTO y views.MetricsDTO) para renderizar con templ. No conoce directamente ni la base de datos.

import (
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"servidor/pkg/application"
	"servidor/pkg/models"
	"servidor/views"
	"strconv"
	"strings"
)

// SSRHandler agrupa los handlers de la aplicación web. Sus dependencias son los casos de uso, que se inyectan por constructor desde main.go
type SSRHandler struct {
	listTasks        *application.ListTasksUseCase
	createTask       *application.CreateTaskUseCase
	deleteTask       *application.DeleteTaskUseCase
	completeTask     *application.CompleteTaskUseCase
	searchTasks      *application.SearchTasksUseCase
	getMetrics       *application.GetMetricsUseCase
	clearCompleted   *application.ClearCompletedTasksUseCase
	updateTaskStatus *application.UpdateTaskStatusUseCase
}

// NewSSRHandler inyecta los casos de uso en el handler.
func NewSSRHandler(
	listTasks *application.ListTasksUseCase,
	createTask *application.CreateTaskUseCase,
	deleteTask *application.DeleteTaskUseCase,
	completeTask *application.CompleteTaskUseCase,
	searchTasks *application.SearchTasksUseCase,
	getMetrics *application.GetMetricsUseCase,
	clearCompleted *application.ClearCompletedTasksUseCase,
	updateTaskStatus *application.UpdateTaskStatusUseCase,
) *SSRHandler {
	return &SSRHandler{
		listTasks:        listTasks,
		createTask:       createTask,
		deleteTask:       deleteTask,
		completeTask:     completeTask,
		searchTasks:      searchTasks,
		getMetrics:       getMetrics,
		clearCompleted:   clearCompleted,
		updateTaskStatus: updateTaskStatus,
	}
}

// IndexHandler maneja la petición principal GET /
// - Petición completa del navegador: obtiene métricas y tareas, renderiza IndexPage
// - Petición reactiva de HTMX (live search por q): filtra por término y devuelve solo EntityList
func (h *SSRHandler) IndexHandler(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}
	if r.Method != http.MethodGet {
		http.Error(w, "Método no permitido", http.StatusMethodNotAllowed)
		return
	}

	query := strings.TrimSpace(r.URL.Query().Get("q"))
	var tasks []models.Task
	var err error

	if query != "" {
		tasks, err = h.searchTasks.SearchTasks(r.Context(), query)
	} else {
		tasks, err = h.listTasks.ListTasks(r.Context())
	}
	if err != nil {
		log.Printf("Error al obtener tareas: %v", err)
		http.Error(w, "Error interno al obtener tareas", http.StatusInternalServerError)
		return
	}

	// Si es una petición reactiva de HTMX, devolvemos únicamente el fragmento EntityList para no recargar la página.
	if r.Header.Get("HX-Request") == "true" {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		if err := views.EntityList(toTaskDTOs(tasks)).Render(r.Context(), w); err != nil {
			log.Printf("Error al renderizar fragmento EntityList: %v", err)
		}
		return
	}

	metrics, err := h.getMetrics.GetMetrics(r.Context())
	if err != nil {
		log.Printf("Error al obtener métricas: %v", err)
		metrics = models.TaskMetrics{}
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := views.IndexPage(toTaskDTOs(tasks), toMetricsDTO(metrics)).Render(r.Context(), w); err != nil {
		log.Printf("Error al renderizar vista IndexPage: %v", err)
		http.Error(w, "Error interno al renderizar la página", http.StatusInternalServerError)
		return
	}
}

// MetricsHandler procesa GET /metrics: Devuelve el fragmento HTML del bloque de métricas actualizado (para eventos taskChanged)
func (h *SSRHandler) MetricsHandler(w http.ResponseWriter, r *http.Request) {
	metrics, err := h.getMetrics.GetMetrics(r.Context())
	if err != nil {
		log.Printf("Error al obtener métricas: %v", err)
		metrics = models.TaskMetrics{}
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := views.MetricsCards(toMetricsDTO(metrics), false).Render(r.Context(), w); err != nil {
		log.Printf("Error al renderizar MetricsCards: %v", err)
	}
}

// TasksToolbarHandler procesa GET /tasks/toolbar
// Devuelve el fragmento de la barra de acciones superior (Mostrando X tareas y botón Limpiar completadas)
func (h *SSRHandler) TasksToolbarHandler(w http.ResponseWriter, r *http.Request) {
	tasks, err := h.listTasks.ListTasks(r.Context())
	if err != nil {
		log.Printf("Error al obtener tareas para toolbar: %v", err)
		tasks = nil
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := views.TasksToolbar(toTaskDTOs(tasks)).Render(r.Context(), w); err != nil {
		log.Printf("Error al renderizar TasksToolbar: %v", err)
	}
}

// CreateTaskHandler procesa el envío de formulario vía POST /tasks enviado por HTMX
// Valida la entrada, delega en CreateTaskUseCase, emite evento taskChanged y devuelve EntityList actualizado
func (h *SSRHandler) CreateTaskHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Método no permitido", http.StatusMethodNotAllowed)
		return
	}

	if err := r.ParseForm(); err != nil {
		htmxError(w, "Error al procesar el formulario: "+err.Error(), http.StatusBadRequest)
		return
	}

	vencimiento, err := parseVencimiento(r.FormValue("vencimiento"))
	if err != nil {
		htmxError(w, err.Error(), http.StatusBadRequest)
		return
	}

	newTask := models.Task{
		Titulo:      r.FormValue("titulo"),
		Descripcion: r.FormValue("descripcion"),
		Estado:      r.FormValue("estado"),
		Prioridad:   r.FormValue("prioridad"),
		Vencimiento: vencimiento,
	}

	_, err = h.createTask.CreateTask(r.Context(), newTask)
	if errors.Is(err, application.ErrTareaInvalida) {
		htmxError(w, err.Error(), http.StatusBadRequest)
		return
	}
	if err != nil {
		log.Printf("Error al crear tarea: %v", err)
		htmxError(w, "Error interno al guardar la tarea", http.StatusInternalServerError)
		return
	}

	tasks, err := h.listTasks.ListTasks(r.Context())
	if err != nil {
		log.Printf("Error al listar tareas: %v", err)
		htmxError(w, "Error interno al obtener tareas", http.StatusInternalServerError)
		return
	}

	// Notificar evento a clientes para sincronizar métricas y toolbar
	w.Header().Set("HX-Trigger", "taskChanged")
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := views.EntityList(toTaskDTOs(tasks)).Render(r.Context(), w); err != nil {
		log.Printf("Error al renderizar vista EntityList: %v", err)
	}
}

// UpdateTaskStatusHandler procesa PATCH /tasks/{id}/status
// Modifica el estado de una tarea (Pendiente a En Progreso, o viceversa)
// notifica evento taskChanged y devuelve la fila actualizada (TaskRow)
func (h *SSRHandler) UpdateTaskStatusHandler(w http.ResponseWriter, r *http.Request) {
	id64, err := strconv.ParseInt(r.PathValue("id"), 10, 32)
	if err != nil || id64 <= 0 {
		htmxError(w, "ID de tarea invalido", http.StatusBadRequest)
		return
	}

	_ = r.ParseForm()
	estado := strings.TrimSpace(r.FormValue("estado"))
	if estado == "" {
		estado = strings.TrimSpace(r.URL.Query().Get("estado"))
	}
	if estado == "" && r.Body != nil {
		var jsonBody struct {
			Estado string `json:"estado"`
		}
		if err := json.NewDecoder(r.Body).Decode(&jsonBody); err == nil {
			estado = strings.TrimSpace(jsonBody.Estado)
		}
	}
	if estado == "" {
		htmxError(w, "El estado es requerido", http.StatusBadRequest)
		return
	}

	task, err := h.updateTaskStatus.UpdateStatus(r.Context(), int32(id64), estado)
	if errors.Is(err, models.ErrTaskNotFound) {
		log.Printf("Modificar estado de tarea inexistente: ID %d", id64)
		w.WriteHeader(http.StatusNotFound)
		return
	}
	if errors.Is(err, application.ErrEstadoInvalido) {
		htmxError(w, err.Error(), http.StatusBadRequest)
		return
	}
	if err != nil {
		log.Printf("Error al actualizar estado de tarea: %v", err)
		htmxError(w, "Error interno al actualizar estado", http.StatusInternalServerError)
		return
	}

	// Sincronizar métricas y toolbar via evento HTMX
	w.Header().Set("HX-Trigger", "taskChanged")
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := views.TaskRow(toTaskDTO(task)).Render(r.Context(), w); err != nil {
		log.Printf("Error al renderizar vista TaskRow: %v", err)
	}
}

// CompleteTaskHandler procesa PATCH /tasks/{id}
// Marca la tarea como completada, notifica taskChanged y devuelve la fila actualizada
func (h *SSRHandler) CompleteTaskHandler(w http.ResponseWriter, r *http.Request) {
	id64, err := strconv.ParseInt(r.PathValue("id"), 10, 32)
	if err != nil || id64 <= 0 {
		htmxError(w, "ID de tarea invalido", http.StatusBadRequest)
		return
	}

	task, err := h.completeTask.CompleteTask(r.Context(), int32(id64))
	if errors.Is(err, models.ErrTaskNotFound) {
		log.Printf("Completar tarea inexistente: ID %d", id64)
		w.WriteHeader(http.StatusNotFound)
		return
	}
	if err != nil {
		log.Printf("Error al completar tarea: %v", err)
		htmxError(w, "Error interno al completar la tarea", http.StatusInternalServerError)
		return
	}

	// Notificar evento a clientes para sincronizar métricas y toolbar
	w.Header().Set("HX-Trigger", "taskChanged")
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := views.TaskRow(toTaskDTO(task)).Render(r.Context(), w); err != nil {
		log.Printf("Error al renderizar vista TaskRow: %v", err)
	}
}

// DeleteTaskHandler procesa DELETE /tasks/{id} desde el botón Borrar
// - Obtiene el ID y borra la tarea con DeleteTaskUseCase
// - Si quedan tareas en el sistema, responde con código 200 OK y cuerpo vacío. HTMX reemplaza closest tr con nada, eliminando la fila de la vista de inmediato
// - Notifica taskChanged para refrescar métricas y la barra de tareas
// - Si ya no quedan tareas, retargetea a #tasks-container para mostrar el mensaje de lista vacía
func (h *SSRHandler) DeleteTaskHandler(w http.ResponseWriter, r *http.Request) {
	id64, err := strconv.ParseInt(r.PathValue("id"), 10, 32)
	if err != nil || id64 <= 0 {
		htmxError(w, "ID de tarea invalido", http.StatusBadRequest)
		return
	}

	status := http.StatusOK
	err = h.deleteTask.DeleteTask(r.Context(), int32(id64))
	if errors.Is(err, models.ErrTaskNotFound) {
		log.Printf("Borrado de tarea inexistente: ID %d", id64)
		status = http.StatusNotFound
	} else if err != nil {
		log.Printf("Error al borrar tarea: %v", err)
		htmxError(w, "Error interno al borrar la tarea", http.StatusInternalServerError)
		return
	}

	tasks, err := h.listTasks.ListTasks(r.Context())
	if err != nil {
		log.Printf("Error al listar tareas despues del borrado: %v", err)
	}

	w.Header().Set("HX-Trigger", "taskChanged")

	// Si ya no quedan tareas, mostrar el contenedor vacío en vez de una tabla sin filas
	if len(tasks) == 0 {
		w.Header().Set("HX-Retarget", "#tasks-container")
		w.Header().Set("HX-Reswap", "outerHTML")
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.WriteHeader(status)
		if err := views.EntityList(toTaskDTOs(tasks)).Render(r.Context(), w); err != nil {
			log.Printf("Error al renderizar vista EntityList: %v", err)
		}
		return
	}

	// Respuesta 200 OK con cuerpo vacío para que HTMX elimine 'closest tr' del DOM
	w.WriteHeader(status)
}

// ClearCompletedHandler procesa DELETE /tasks/completed
// Elimina todas las tareas con estado 'Completada', notifica taskChanged y devuelve EntityList actualizado a #tasks-container
func (h *SSRHandler) ClearCompletedHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodDelete {
		http.Error(w, "Método no permitido", http.StatusMethodNotAllowed)
		return
	}

	_, err := h.clearCompleted.ClearCompleted(r.Context())
	if err != nil {
		log.Printf("Error al limpiar tareas completadas: %v", err)
		htmxError(w, "Error interno al eliminar tareas completadas", http.StatusInternalServerError)
		return
	}

	tasks, err := h.listTasks.ListTasks(r.Context())
	if err != nil {
		log.Printf("Error al listar tareas despues de limpiar completadas: %v", err)
		htmxError(w, "Error interno al obtener tareas", http.StatusInternalServerError)
		return
	}

	w.Header().Set("HX-Trigger", "taskChanged")
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := views.EntityList(toTaskDTOs(tasks)).Render(r.Context(), w); err != nil {
		log.Printf("Error al renderizar vista EntityList: %v", err)
	}
}
