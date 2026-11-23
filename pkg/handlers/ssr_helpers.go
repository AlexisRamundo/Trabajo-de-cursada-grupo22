package handlers

// ssr_helpers.go agrupa funciones de soporte, HTMX y mappers entre las entidades del dominio (models.Task, models.TaskMetrics)
// y los DTOs de presentación (views.TaskDTO, views.MetricsDTO).

import (
	"errors"
	"net/http"
	"servidor/pkg/models"
	"servidor/views"
	"strings"
	"time"
)

// htmxError responde un error sin que HTMX lo inserte en la página. Mediante el encabezado HX-Reswap none se evita alterar el DOM ante códigos 4xx o 5xx
func htmxError(w http.ResponseWriter, mensaje string, codigo int) {
	w.Header().Set("HX-Reswap", "none")
	http.Error(w, mensaje, codigo)
}

// Formatos aceptados para la fecha de vencimiento. El primero es el estándar que envía, los otros dos se aceptan por compatibilidad.
var formatosVencimiento = []string{
	"2006-01-02T15:04",
	"2006-01-02T15:04:05",
	"2006-01-02",
}

// parseVencimiento convierte el texto del formulario HTML en *time.Time
// Si el valor está vacío devuelve nil (tarea sin vencimiento).
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

// formatoFecha es el formato legible con el que las vistas renderizan las fechas
const formatoFecha = "02/01/2006 15:04"

// toTaskDTO convierte una entidad del dominio (models.Task) en el DTO de la vista (views.TaskDTO)
func toTaskDTO(t models.Task) views.TaskDTO {
	dto := views.TaskDTO{
		ID:          t.ID,
		Titulo:      t.Titulo,
		Descripcion: t.Descripcion,
		Estado:      t.Estado,
		Prioridad:   t.Prioridad,
		CreadoEn:    t.CreadoEn.Format(formatoFecha),
		Completada:  t.Estado == models.EstadoCompletada,
	}
	if t.Vencimiento != nil {
		dto.Vencimiento = t.Vencimiento.Format(formatoFecha)
	}
	return dto
}

// toTaskDTOs convierte un slice de entidades del dominio en un slice de DTOs para la vista
func toTaskDTOs(tasks []models.Task) []views.TaskDTO {
	dtos := make([]views.TaskDTO, 0, len(tasks))
	for _, t := range tasks {
		dtos = append(dtos, toTaskDTO(t))
	}
	return dtos
}

// toMetricsDTO convierte la entidad TaskMetrics en MetricsDTO para la vista
func toMetricsDTO(m models.TaskMetrics) views.MetricsDTO {
	return views.MetricsDTO{
		Total:       m.Total,
		Pendientes:  m.Pendientes,
		EnProgreso:  m.EnProgreso,
		Completadas: m.Completadas,
		Vencidas:    m.Vencidas,
	}
}