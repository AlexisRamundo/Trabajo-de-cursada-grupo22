package application

import (
	"context"
	"servidor/pkg/models"
)

// GetMetricsUseCase orquesta el caso de uso "obtener métricas cuantitativas de tareas"
type GetMetricsUseCase struct {
	taskRepo models.TaskRepository
}

// NewGetMetricsUseCase inyecta el repositorio por constructor
func NewGetMetricsUseCase(taskRepo models.TaskRepository) *GetMetricsUseCase {
	return &GetMetricsUseCase{taskRepo: taskRepo}
}

// GetMetrics consulta el resumen estadístico de tareas en el repositorio
func (uc *GetMetricsUseCase) GetMetrics(ctx context.Context) (models.TaskMetrics, error) {
	return uc.taskRepo.GetMetrics(ctx)
}
