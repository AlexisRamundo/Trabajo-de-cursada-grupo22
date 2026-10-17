package models

import (
	"errors"
	"strings"
	"time"
	"unicode/utf8"
)

// Valores permitidos para Estado y Prioridad
const (
	EstadoPendiente  = "Pendiente"
	EstadoEnProgreso = "En Progreso"
	EstadoCompletada = "Completada"

	PrioridadBaja  = "Baja"
	PrioridadMedia = "Media"
	PrioridadAlta  = "Alta"
)

// Task es la entidad del dominio tal como se recibe y se devuelve por la API
// Vencimiento es un puntero: nil significa "sin fecha de vencimiento"
type Task struct {
	ID          int32      `json:"id"`
	Titulo      string     `json:"titulo"`
	Descripcion string     `json:"descripcion"`
	Estado      string     `json:"estado"`
	Prioridad   string     `json:"prioridad"`
	CreadoEn    time.Time  `json:"creado_en"`
	Vencimiento *time.Time `json:"vencimiento"`
}

// TaskMetrics representa el resumen estadístico de tareas para dashboards y métricas.
type TaskMetrics struct {
	Total       int32 `json:"total"`
	Pendientes  int32 `json:"pendientes"`
	EnProgreso  int32 `json:"en_progreso"`
	Completadas int32 `json:"completadas"`
	Vencidas    int32 `json:"vencidas"`
}

// ApplyDefaults limpia espacios y completa los valores por defecto (los mismos que tiene el schema: Pendiente y Media)
func ApplyDefaults(t Task) Task {
	t.Titulo = strings.TrimSpace(t.Titulo)
	t.Descripcion = strings.TrimSpace(t.Descripcion)
	if t.Estado == "" {
		t.Estado = EstadoPendiente
	}
	if t.Prioridad == "" {
		t.Prioridad = PrioridadMedia
	}
	return t
}

// ValidateTask verifica las reglas del dominio.
func ValidateTask(t Task) error {
	if t.Titulo == "" {
		return errors.New("el titulo es obligatorio")
	}
	if utf8.RuneCountInString(t.Titulo) > 255 {
		return errors.New("el titulo no puede superar los 255 caracteres")
	}
	switch t.Estado {
	case EstadoPendiente, EstadoEnProgreso, EstadoCompletada:
	default:
		return errors.New("el estado debe ser Pendiente, En Progreso o Completada")
	}
	switch t.Prioridad {
	case PrioridadBaja, PrioridadMedia, PrioridadAlta:
	default:
		return errors.New("la prioridad debe ser Baja, Media o Alta")
	}
	return nil
}