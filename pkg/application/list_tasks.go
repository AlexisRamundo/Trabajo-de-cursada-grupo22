package application

// Capa de aplicación Onion: Application Services / casos de uso
// Orquesta los casos de uso usando las entidades y los puertos del dominio (pkg/models). No conoce HTTP, HTML ni la base de datos concreta.

import (
	"context"
	"servidor/pkg/models"
)

// ListTasksUseCase orquesta el caso de uso "listar tareas".
type ListTasksUseCase struct {
	taskRepo models.TaskRepository
}

// NewListTasksUseCase inyecta el repositorio por constructor.
func NewListTasksUseCase(taskRepo models.TaskRepository) *ListTasksUseCase {
	return &ListTasksUseCase{taskRepo: taskRepo}
}

// ListTasks devuelve todas las tareas del dominio.
func (uc *ListTasksUseCase) ListTasks(ctx context.Context) ([]models.Task, error) {
	return uc.taskRepo.FindAll(ctx)
}
