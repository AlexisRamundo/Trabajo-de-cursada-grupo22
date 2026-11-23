package application

import (
	"context"
	"errors"
	"fmt"
	"servidor/pkg/models"
)

// ErrTareaInvalida indica que la tarea no cumple las reglas del negocio.
// El handler la detecta con errors.Is y responde 400; cualquier otro error,(por ejemplode la base de datos) se responde como 500.
var ErrTareaInvalida = errors.New("tarea invalida")

// CreateTaskUseCase orquesta el caso de uso "crear tarea".
type CreateTaskUseCase struct {
	taskRepo models.TaskRepository
}

// NewCreateTaskUseCase inyecta el repositorio por constructor.
func NewCreateTaskUseCase(taskRepo models.TaskRepository) *CreateTaskUseCase {
	return &CreateTaskUseCase{taskRepo: taskRepo}
}

// CreateTask aplica las reglas del negocio del dominio (valores por defecto y validación) y, si la tarea es válida, la guarda a través del repositorio.
func (uc *CreateTaskUseCase) CreateTask(ctx context.Context, task models.Task) (models.Task, error) {
	task = models.ApplyDefaults(task)
	if err := models.ValidateTask(task); err != nil {
		return models.Task{}, fmt.Errorf("%w: %v", ErrTareaInvalida, err)
	}
	return uc.taskRepo.Create(ctx, task)
}
