package application_test

// Pruebas de los casos de uso aisladas de la infraestructura, como los casos de uso dependen de la interfaz
// models.TaskRepository, se les inyecta un repositorio falso y no hace falta levantar PostgreSQL.

import (
	"context"
	"errors"
	"servidor/pkg/application"
	"servidor/pkg/models"
	"strings"
	"testing"
)

// MockTaskRepository es una implementación falsa de models.TaskRepository.
type MockTaskRepository struct {
	tasks   []models.Task
	created []models.Task // tareas que recibió Create
	deleted []int32       // IDs que recibió Delete
}

func (m *MockTaskRepository) FindAll(ctx context.Context) ([]models.Task, error) {
	return m.tasks, nil
}

func (m *MockTaskRepository) Create(ctx context.Context, task models.Task) (models.Task, error) {
	m.created = append(m.created, task)
	task.ID = int32(len(m.created))
	return task, nil
}

func (m *MockTaskRepository) Delete(ctx context.Context, id int32) error {
	for i, t := range m.tasks {
		if t.ID == id {
			m.tasks = append(m.tasks[:i], m.tasks[i+1:]...)
			m.deleted = append(m.deleted, id)
			return nil
		}
	}
	return models.ErrTaskNotFound
}

func (m *MockTaskRepository) MarkCompleted(ctx context.Context, id int32) (models.Task, error) {
	for i, t := range m.tasks {
		if t.ID == id {
			m.tasks[i].Estado = models.EstadoCompletada
			return m.tasks[i], nil
		}
	}
	return models.Task{}, models.ErrTaskNotFound
}

func (m *MockTaskRepository) UpdateStatus(ctx context.Context, id int32, status string) (models.Task, error) {
	for i, t := range m.tasks {
		if t.ID == id {
			m.tasks[i].Estado = status
			return m.tasks[i], nil
		}
	}
	return models.Task{}, models.ErrTaskNotFound
}

func (m *MockTaskRepository) Search(ctx context.Context, query string) ([]models.Task, error) {
	if query == "" {
		return m.tasks, nil
	}
	var res []models.Task
	for _, t := range m.tasks {
		if strings.Contains(strings.ToLower(t.Titulo), strings.ToLower(query)) ||
			strings.Contains(strings.ToLower(t.Descripcion), strings.ToLower(query)) {
			res = append(res, t)
		}
	}
	return res, nil
}

func (m *MockTaskRepository) GetMetrics(ctx context.Context) (models.TaskMetrics, error) {
	var metrics models.TaskMetrics
	metrics.Total = int32(len(m.tasks))
	for _, t := range m.tasks {
		switch t.Estado {
		case models.EstadoPendiente:
			metrics.Pendientes++
		case models.EstadoEnProgreso:
			metrics.EnProgreso++
		case models.EstadoCompletada:
			metrics.Completadas++
		}
	}
	return metrics, nil
}

func (m *MockTaskRepository) DeleteCompleted(ctx context.Context) (int64, error) {
	var count int64
	var remaining []models.Task
	for _, t := range m.tasks {
		if t.Estado == models.EstadoCompletada {
			count++
		} else {
			remaining = append(remaining, t)
		}
	}
	m.tasks = remaining
	return count, nil
}

func TestListTasksUseCase_ListTasks(t *testing.T) {
	mockRepo := &MockTaskRepository{tasks: []models.Task{{ID: 1, Titulo: "Estudiar"}}}
	listTasks := application.NewListTasksUseCase(mockRepo)

	tasks, err := listTasks.ListTasks(context.Background())

	if err != nil {
		t.Fatalf("no se esperaba error: %v", err)
	}
	if len(tasks) != 1 || tasks[0].Titulo != "Estudiar" {
		t.Fatalf("se esperaba la tarea del repositorio, se obtuvo %+v", tasks)
	}
}

func TestCreateTaskUseCase_AplicaValoresPorDefecto(t *testing.T) {
	mockRepo := &MockTaskRepository{}
	createTask := application.NewCreateTaskUseCase(mockRepo)

	created, err := createTask.CreateTask(context.Background(), models.Task{Titulo: "  Comprar insumos  "})

	if err != nil {
		t.Fatalf("no se esperaba error: %v", err)
	}
	if len(mockRepo.created) != 1 {
		t.Fatalf("se esperaba que la tarea se guarde una vez, se guardó %d veces", len(mockRepo.created))
	}
	if created.Titulo != "Comprar insumos" {
		t.Errorf("titulo = %q, se esperaba sin espacios", created.Titulo)
	}
	if created.Estado != models.EstadoPendiente || created.Prioridad != models.PrioridadMedia {
		t.Errorf("estado/prioridad = %q/%q, se esperaban los valores por defecto", created.Estado, created.Prioridad)
	}
}

func TestCreateTaskUseCase_RechazaTareaInvalida(t *testing.T) {
	casos := []struct {
		nombre string
		task   models.Task
	}{
		{"titulo vacio", models.Task{Titulo: "   "}},
		{"estado invalido", models.Task{Titulo: "X", Estado: "Cancelada"}},
		{"prioridad invalida", models.Task{Titulo: "X", Prioridad: "Urgente"}},
	}

	for _, tc := range casos {
		t.Run(tc.nombre, func(t *testing.T) {
			mockRepo := &MockTaskRepository{}
			createTask := application.NewCreateTaskUseCase(mockRepo)

			_, err := createTask.CreateTask(context.Background(), tc.task)

			if !errors.Is(err, application.ErrTareaInvalida) {
				t.Fatalf("se esperaba ErrTareaInvalida, se obtuvo %v", err)
			}
			if len(mockRepo.created) != 0 {
				t.Fatalf("una tarea invalida no debe llegar al repositorio")
			}
		})
	}
}

func TestDeleteTaskUseCase_BorraLaTarea(t *testing.T) {
	mockRepo := &MockTaskRepository{tasks: []models.Task{{ID: 1, Titulo: "A"}, {ID: 2, Titulo: "B"}}}
	deleteTask := application.NewDeleteTaskUseCase(mockRepo)

	err := deleteTask.DeleteTask(context.Background(), 2)

	if err != nil {
		t.Fatalf("no se esperaba error: %v", err)
	}
	if len(mockRepo.deleted) != 1 || mockRepo.deleted[0] != 2 {
		t.Fatalf("se esperaba borrar el ID 2, se borró %v", mockRepo.deleted)
	}
	if len(mockRepo.tasks) != 1 {
		t.Fatalf("se esperaba que quede 1 tarea, quedaron %d", len(mockRepo.tasks))
	}
}

func TestDeleteTaskUseCase_TareaInexistente(t *testing.T) {
	mockRepo := &MockTaskRepository{tasks: []models.Task{{ID: 1, Titulo: "A"}}}
	deleteTask := application.NewDeleteTaskUseCase(mockRepo)

	err := deleteTask.DeleteTask(context.Background(), 99)

	if !errors.Is(err, models.ErrTaskNotFound) {
		t.Fatalf("se esperaba ErrTaskNotFound, se obtuvo %v", err)
	}
}

func TestCompleteTaskUseCase_MarcaComoCompletada(t *testing.T) {
	mockRepo := &MockTaskRepository{tasks: []models.Task{{ID: 1, Titulo: "A", Estado: models.EstadoPendiente}}}
	completeTask := application.NewCompleteTaskUseCase(mockRepo)

	task, err := completeTask.CompleteTask(context.Background(), 1)

	if err != nil {
		t.Fatalf("no se esperaba error: %v", err)
	}
	if task.Estado != models.EstadoCompletada {
		t.Fatalf("estado = %q, se esperaba %q", task.Estado, models.EstadoCompletada)
	}
}

func TestCompleteTaskUseCase_TareaInexistente(t *testing.T) {
	mockRepo := &MockTaskRepository{}
	completeTask := application.NewCompleteTaskUseCase(mockRepo)

	_, err := completeTask.CompleteTask(context.Background(), 99)

	if !errors.Is(err, models.ErrTaskNotFound) {
		t.Fatalf("se esperaba ErrTaskNotFound, se obtuvo %v", err)
	}
}

func TestSearchTasksUseCase_FiltraPorTexto(t *testing.T) {
	mockRepo := &MockTaskRepository{tasks: []models.Task{
		{ID: 1, Titulo: "Aprender Go", Descripcion: "Basico"},
		{ID: 2, Titulo: "Estudiar HTMX", Descripcion: "Frontend reactivo"},
		{ID: 3, Titulo: "Repasar SQL", Descripcion: "Bases de datos"},
	}}
	searchUC := application.NewSearchTasksUseCase(mockRepo)

	res, err := searchUC.SearchTasks(context.Background(), "htmx")
	if err != nil {
		t.Fatalf("no se esperaba error: %v", err)
	}
	if len(res) != 1 || res[0].ID != 2 {
		t.Fatalf("se esperaba 1 tarea coincidente, se obtuvo %+v", res)
	}

	resVacio, err := searchUC.SearchTasks(context.Background(), "")
	if err != nil {
		t.Fatalf("no se esperaba error: %v", err)
	}
	if len(resVacio) != 3 {
		t.Fatalf("con query vacia se esperaban todas las tareas, se obtuvo %d", len(resVacio))
	}
}

func TestGetMetricsUseCase_CalculaResumen(t *testing.T) {
	mockRepo := &MockTaskRepository{tasks: []models.Task{
		{ID: 1, Estado: models.EstadoPendiente},
		{ID: 2, Estado: models.EstadoPendiente},
		{ID: 3, Estado: models.EstadoCompletada},
	}}
	metricsUC := application.NewGetMetricsUseCase(mockRepo)

	m, err := metricsUC.GetMetrics(context.Background())
	if err != nil {
		t.Fatalf("no se esperaba error: %v", err)
	}
	if m.Total != 3 || m.Pendientes != 2 || m.Completadas != 1 {
		t.Fatalf("metricas incorrectas: %+v", m)
	}
}

func TestClearCompletedTasksUseCase_EliminaSoloCompletadas(t *testing.T) {
	mockRepo := &MockTaskRepository{tasks: []models.Task{
		{ID: 1, Titulo: "A", Estado: models.EstadoPendiente},
		{ID: 2, Titulo: "B", Estado: models.EstadoCompletada},
		{ID: 3, Titulo: "C", Estado: models.EstadoCompletada},
	}}
	clearUC := application.NewClearCompletedTasksUseCase(mockRepo)

	afectadas, err := clearUC.ClearCompleted(context.Background())
	if err != nil {
		t.Fatalf("no se esperaba error: %v", err)
	}
	if afectadas != 2 {
		t.Fatalf("se esperaban 2 tareas eliminadas, se obtuvo %d", afectadas)
	}
	if len(mockRepo.tasks) != 1 || mockRepo.tasks[0].ID != 1 {
		t.Fatalf("debia quedar solo la tarea 1 pendiente, se obtuvo %+v", mockRepo.tasks)
	}
}

func TestUpdateTaskStatusUseCase_ModificaEstado(t *testing.T) {
	mockRepo := &MockTaskRepository{tasks: []models.Task{
		{ID: 1, Titulo: "A", Estado: models.EstadoPendiente},
	}}
	updateStatusUC := application.NewUpdateTaskStatusUseCase(mockRepo)

	task, err := updateStatusUC.UpdateStatus(context.Background(), 1, models.EstadoEnProgreso)
	if err != nil {
		t.Fatalf("no se esperaba error: %v", err)
	}
	if task.Estado != models.EstadoEnProgreso {
		t.Fatalf("se esperaba estado En Progreso, se obtuvo %s", task.Estado)
	}
}

func TestUpdateTaskStatusUseCase_RechazaEstadoInvalido(t *testing.T) {
	mockRepo := &MockTaskRepository{tasks: []models.Task{
		{ID: 1, Titulo: "A", Estado: models.EstadoPendiente},
	}}
	updateStatusUC := application.NewUpdateTaskStatusUseCase(mockRepo)

	_, err := updateStatusUC.UpdateStatus(context.Background(), 1, "EstadoInexistente")
	if !errors.Is(err, application.ErrEstadoInvalido) {
		t.Fatalf("se esperaba ErrEstadoInvalido, se obtuvo: %v", err)
	}
}
