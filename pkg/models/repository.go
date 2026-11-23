package models

import (
	"context"
	"errors"
)

// ErrTaskNotFound indica que no existe una tarea con el ID indicado.
var ErrTaskNotFound = errors.New("tarea no encontrada")

// TaskRepository es el puerto de salida del dominio
// La implementación concreta (PostgreSQL con sqlc) vive en la capa de infraestructura (pkg/infrastructure) y se inyecta desde main.go

type TaskRepository interface {
	// FindAll devuelve todas las tareas.
	FindAll(ctx context.Context) ([]Task, error)
	// Create guarda una tarea nueva y la devuelve con los datos que asigna la base de datos (ID y fecha de creación).
	Create(ctx context.Context, task Task) (Task, error)
	// Delete borra la tarea con el ID indicado. Devuelve ErrTaskNotFound si no existe.
	Delete(ctx context.Context, id int32) error
	// MarkCompleted cambia el estado de la tarea a Completada y la devuelve actualizada. Devuelve ErrTaskNotFound si la tarea no existe.
	MarkCompleted(ctx context.Context, id int32) (Task, error)
	// UpdateStatus modifica el estado de la tarea (Pendiente, En Progreso, Completada) y la devuelve actualizada.
	UpdateStatus(ctx context.Context, id int32, status string) (Task, error)
	// Search busca tareas por término clave en título o descripción.
	Search(ctx context.Context, query string) ([]Task, error)
	// GetMetrics calcula el resumen cuantitativo de tareas por estado y vencimiento.
	GetMetrics(ctx context.Context) (TaskMetrics, error)
	// DeleteCompleted elimina en lote todas las tareas en estado Completada.
	DeleteCompleted(ctx context.Context) (int64, error)
}
