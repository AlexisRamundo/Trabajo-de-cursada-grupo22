package views

// TaskDTO es la representación de una tarea que necesita la vista.
// La vista no conoce el struct de la base de datos ni la entidad del dominio (models.Task), el handler (pkg/handlers/ssr.go)
// convierte cada entidad en un TaskDTO antes de renderizar, con las fechas ya formateadas como texto.
type TaskDTO struct {
	ID          int32
	Titulo      string
	Descripcion string
	Estado      string
	Prioridad   string
	CreadoEn    string
	Vencimiento string // vacío si la tarea no tiene fecha de vencimiento
	Completada  bool   // true si el estado es Completada (oculta el botón "Completar")
}

// MetricsDTO representa el resumen cuantitativo de tareas para mostrar en la vista.
type MetricsDTO struct {
	Total       int32
	Pendientes  int32
	EnProgreso  int32
	Completadas int32
	Vencidas    int32
}

// HasCompleted indica si en la lista de tareas hay alguna con estado Completada.
func HasCompleted(tasks []TaskDTO) bool {
	for _, t := range tasks {
		if t.Completada {
			return true
		}
	}
	return false
}
