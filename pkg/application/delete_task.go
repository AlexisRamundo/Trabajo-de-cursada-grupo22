package application

import (
	"context"
	"servidor/pkg/models"
)

// DeleteTaskUseCase orquesta el caso de uso "borrar tarea"
type DeleteTaskUseCase struct {
	taskRepo models.TaskRepository
}

// NewDeleteTaskUseCase inyecta el repositorio por constructor
func NewDeleteTaskUseCase(taskRepo models.TaskRepository) *DeleteTaskUseCase {
	return &DeleteTaskUseCase{taskRepo: taskRepo}
}

// DeleteTask borra la tarea con el ID indicado a través del repositorio.
func (uc *DeleteTaskUseCase) DeleteTask(ctx context.Context, id int32) error {
	return uc.taskRepo.Delete(ctx, id)
}
