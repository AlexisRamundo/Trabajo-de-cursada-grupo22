package application

import (
	"context"
	"servidor/pkg/models"
)

// ClearCompletedTasksUseCase orquesta el caso de uso "eliminar en lote tareas completadas".
type ClearCompletedTasksUseCase struct {
	taskRepo models.TaskRepository
}

// NewClearCompletedTasksUseCase inyecta el repositorio por constructor.
func NewClearCompletedTasksUseCase(taskRepo models.TaskRepository) *ClearCompletedTasksUseCase {
	return &ClearCompletedTasksUseCase{taskRepo: taskRepo}
}

// ClearCompleted elimina todas las tareas completadas y devuelve la cantidad afectada.
func (uc *ClearCompletedTasksUseCase) ClearCompleted(ctx context.Context) (int64, error) {
	return uc.taskRepo.DeleteCompleted(ctx)
}
