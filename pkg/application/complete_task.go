package application

import (
	"context"
	"servidor/pkg/models"
)

// CompleteTaskUseCase orquesta el caso de uso "marcar tarea como completada".
type CompleteTaskUseCase struct {
	taskRepo models.TaskRepository
}

// NewCompleteTaskUseCase inyecta el repositorio por constructor.
func NewCompleteTaskUseCase(taskRepo models.TaskRepository) *CompleteTaskUseCase {
	return &CompleteTaskUseCase{taskRepo: taskRepo}
}

// CompleteTask marca la tarea como completada y la devuelve actualizada.
// Devuelve models.ErrTaskNotFound si la tarea no existe.
func (uc *CompleteTaskUseCase) CompleteTask(ctx context.Context, id int32) (models.Task, error) {
	return uc.taskRepo.MarkCompleted(ctx, id)
}
