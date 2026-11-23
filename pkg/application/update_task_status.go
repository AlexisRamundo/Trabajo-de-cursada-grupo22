package application

import (
	"context"
	"errors"
	"servidor/pkg/models"
)

// ErrEstadoInvalido indica que se intentó asignar un estado no permitido por el dominio
var ErrEstadoInvalido = errors.New("estado de tarea no valido")

// UpdateTaskStatusUseCase orquesta la modificación del estado de una tarea
type UpdateTaskStatusUseCase struct {
	taskRepo models.TaskRepository
}

// NewUpdateTaskStatusUseCase inyecta el repositorio por constructor
func NewUpdateTaskStatusUseCase(taskRepo models.TaskRepository) *UpdateTaskStatusUseCase {
	return &UpdateTaskStatusUseCase{taskRepo: taskRepo}
}

// UpdateStatus valida el nuevo estado y delega en el repositorio la actualización
func (uc *UpdateTaskStatusUseCase) UpdateStatus(ctx context.Context, id int32, estado string) (models.Task, error) {
	if estado != models.EstadoPendiente && estado != models.EstadoEnProgreso && estado != models.EstadoCompletada {
		return models.Task{}, ErrEstadoInvalido
	}
	return uc.taskRepo.UpdateStatus(ctx, id, estado)
}
