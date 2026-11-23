package application

import (
	"context"
	"servidor/pkg/models"
)

// SearchTasksUseCase orquesta el caso de uso "buscar tareas por texto".
type SearchTasksUseCase struct {
	taskRepo models.TaskRepository
}

// NewSearchTasksUseCase inyecta el repositorio por constructor.
func NewSearchTasksUseCase(taskRepo models.TaskRepository) *SearchTasksUseCase {
	return &SearchTasksUseCase{taskRepo: taskRepo}
}

// SearchTasks busca tareas por palabra clave a través del repositorio.
func (uc *SearchTasksUseCase) SearchTasks(ctx context.Context, query string) ([]models.Task, error) {
	return uc.taskRepo.Search(ctx, query)
}
