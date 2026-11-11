package handlers

// ssr.go (TP5) maneja los handlers de renderizado correspondientes a la implementacion del lado del servidor

import (
	"errors"
	"log"
	"net/http"
	"strings"
	"time"

	db "servidor/db/sqlc"
	"servidor/pkg/models"
	"servidor/views"
)

// IndexHandler maneja la petición principal GET /:
// Obtiene todas las tareas desde PostgreSQL mediante sqlc y renderiza
// la página completa en el http.ResponseWriter utilizando Templ (IndexPage).
func (h *Handler) IndexHandler(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}
	if r.Method != http.MethodGet {
		http.Error(w, "Método no permitido", http.StatusMethodNotAllowed)
		return
	}

	tasks, err := h.queries.ListTasks(r.Context())
	if err != nil {
		log.Printf("Error al listar tareas: %v", err)
		http.Error(w, "Error interno al obtener tareas", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := views.IndexPage(tasks).Render(r.Context(), w); err != nil {
		log.Printf("Error al renderizar vista IndexPage: %v", err)
		http.Error(w, "Error interno al renderizar la página", http.StatusInternalServerError)
		return
	}
}

// CreateTaskHandler procesa el envío de formulario vía POST /tasks:
// Parsea los campos del formulario HTML, aplica las reglas de negocio de pkg/models (ApplyDefaults y ValidateTask)
// Persiste la nueva tarea con sqlc y redirige a / mediante el patrón Post/Redirect/Get (PRG)
func (h *Handler) CreateTaskHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Método no permitido", http.StatusMethodNotAllowed)
		return
	}

	// Presentación: leer los campos del formulario HTML
	if err := r.ParseForm(); err != nil {
		http.Error(w, "Error al procesar el formulario: "+err.Error(), http.StatusBadRequest)
		return
	}

	vencimiento, err := parseVencimiento(r.FormValue("vencimiento"))
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	newTask := models.Task{
		Titulo:      r.FormValue("titulo"),
		Descripcion: r.FormValue("descripcion"),
		Estado:      r.FormValue("estado"),
		Prioridad:   r.FormValue("prioridad"),
		Vencimiento: vencimiento,
	}

	// Lógica de negocio: aplicar reglas y validar la tarea (pkg/models)
	newTask = models.ApplyDefaults(newTask)
	if err := models.ValidateTask(newTask); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Datos: persistir con el método generado por sqlc
	_, err = h.queries.CreateTask(r.Context(), db.CreateTaskParams{
		Titulo:      newTask.Titulo,
		Descripcion: newTask.Descripcion,
		Estado:      newTask.Estado,
		Prioridad:   newTask.Prioridad,
		Vencimiento: toNullTime(newTask.Vencimiento),
	})
	if err != nil {
		log.Printf("Error al crear tarea: %v", err)
		http.Error(w, "Error interno al guardar la tarea", http.StatusInternalServerError)
		return
	}

	// Patrón Post/Redirect/Get (PRG): Redirige a GET /
	http.Redirect(w, r, "/", http.StatusSeeOther)
}

// Formatos aceptados para la fecha de vencimiento. El primero es el que envía
// <input type="datetime-local">; los otros dos se aceptan por compatibilidad.
var formatosVencimiento = []string{
	"2006-01-02T15:04",
	"2006-01-02T15:04:05",
	"2006-01-02",
}

// parseVencimiento convierte el texto del formulario en *time.Time
// Campo vacío: devuelve nil (tarea sin vencimiento)
// Formato inválido: devuelve error, en lugar de descartar la fecha sin avisar
func parseVencimiento(valor string) (*time.Time, error) {
	valor = strings.TrimSpace(valor)
	if valor == "" {
		return nil, nil
	}
	for _, formato := range formatosVencimiento {
		if t, err := time.Parse(formato, valor); err == nil {
			return &t, nil
		}
	}
	return nil, errors.New("la fecha de vencimiento no tiene un formato valido")
}
