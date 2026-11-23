package infrastructure

// Capa de infraestructura (Onion: borde exterior).
// Implementa los puertos del dominio, acá vive elacceso a PostgreSQL mediante el código generado por sqlc. Es el único lugar de la aplicación SSR que conoce los tipos de db/sqlc.

import (
	"context"
	"database/sql"
	"errors"
	db "servidor/db/sqlc"
	"servidor/pkg/models"
	"strings"
	"time"
)

// PostgresTaskRepository implementa models.TaskRepository usando sqlc
type PostgresTaskRepository struct {
	queries *db.Queries
}

// Verificación en tiempo de compilación: PostgresTaskRepository cumple el puerto del dominio
var _ models.TaskRepository = (*PostgresTaskRepository)(nil)

// NewPostgresTaskRepository crea el repositorio a partir de la conexión a la base.
func NewPostgresTaskRepository(conn *sql.DB) *PostgresTaskRepository {
	return &PostgresTaskRepository{queries: db.New(conn)}
}

// FindAll recupera todas las tareas de la base y las convierte a entidades del dominio
func (r *PostgresTaskRepository) FindAll(ctx context.Context) ([]models.Task, error) {
	rows, err := r.queries.ListTasks(ctx)
	if err != nil {
		return nil, err
	}
	tasks := make([]models.Task, 0, len(rows))
	for _, row := range rows {
		tasks = append(tasks, toDomain(row))
	}
	return tasks, nil
}

// Create guarda la tarea y devuelve la entidad con el ID y la fecha de creación asignados
func (r *PostgresTaskRepository) Create(ctx context.Context, task models.Task) (models.Task, error) {
	created, err := r.queries.CreateTask(ctx, db.CreateTaskParams{
		Titulo:      task.Titulo,
		Descripcion: task.Descripcion,
		Estado:      task.Estado,
		Prioridad:   task.Prioridad,
		Vencimiento: toNullTime(task.Vencimiento),
	})
	if err != nil {
		return models.Task{}, err
	}
	return toDomain(created), nil
}

// Delete borra la tarea con el ID indicado. Si la consulta no eliminó ninguna fila, la tarea no existía y se devuelve el error del dominio ErrTaskNotFound.
func (r *PostgresTaskRepository) Delete(ctx context.Context, id int32) error {
	rows, err := r.queries.DeleteTaskByID(ctx, id)
	if err != nil {
		return err
	}
	if rows == 0 {
		return models.ErrTaskNotFound
	}
	return nil
}

// MarkCompleted marca la tarea como completada con la consulta MarkTaskCompleted y devuelve la tarea actualizada 
// Si la tarea no existe, la consulta devuelve un error que se traduce al del dominio ErrTaskNotFound
func (r *PostgresTaskRepository) MarkCompleted(ctx context.Context, id int32) (models.Task, error) {
	updated, err := r.queries.MarkTaskCompleted(ctx, id)
	if errors.Is(err, sql.ErrNoRows) {
		return models.Task{}, models.ErrTaskNotFound
	}
	if err != nil {
		return models.Task{}, err
	}
	return toDomain(updated), nil
}

// UpdateStatus modifica el estado de la tarea mediante UpdateTaskStatus.
func (r *PostgresTaskRepository) UpdateStatus(ctx context.Context, id int32, status string) (models.Task, error) {
	updated, err := r.queries.UpdateTaskStatus(ctx, db.UpdateTaskStatusParams{
		ID:     id,
		Estado: status,
	})
	if errors.Is(err, sql.ErrNoRows) {
		return models.Task{}, models.ErrTaskNotFound
	}
	if err != nil {
		return models.Task{}, err
	}
	return toDomain(updated), nil
}

// Search busca tareas por coincidencia parcial en título o descripción (SearchTasks)
// Si el término está vacío, delega en FindAll.
func (r *PostgresTaskRepository) Search(ctx context.Context, query string) ([]models.Task, error) {
	query = strings.TrimSpace(query)
	if query == "" {
		return r.FindAll(ctx)
	}
	rows, err := r.queries.SearchTasks(ctx, query)
	if err != nil {
		return nil, err
	}
	tasks := make([]models.Task, 0, len(rows))
	for _, row := range rows {
		tasks = append(tasks, toDomain(row))
	}
	return tasks, nil
}

// GetMetrics consulta el resumen cuantitativo de la base con GetTaskMetrics.
func (r *PostgresTaskRepository) GetMetrics(ctx context.Context) (models.TaskMetrics, error) {
	row, err := r.queries.GetTaskMetrics(ctx)
	if err != nil {
		return models.TaskMetrics{}, err
	}
	return models.TaskMetrics{
		Total:       row.Total,
		Pendientes:  row.Pendientes,
		EnProgreso:  row.EnProgreso,
		Completadas: row.Completadas,
		Vencidas:    row.Vencidas,
	}, nil
}

// DeleteCompleted elimina en lote todas las tareas completadas (DeleteCompletedTasks).
func (r *PostgresTaskRepository) DeleteCompleted(ctx context.Context) (int64, error) {
	res, err := r.queries.DeleteCompletedTasks(ctx)
	if err != nil {
		return 0, err
	}
	return res.RowsAffected()
}

// toDomain convierte el struct generado por sqlc (db.Task) en la entidad del dominio.
// sql.NullTime pasa a *time.Time: nil sera "sin vencimiento"
func toDomain(t db.Task) models.Task {
	task := models.Task{
		ID:          t.ID,
		Titulo:      t.Titulo,
		Descripcion: t.Descripcion,
		Estado:      t.Estado,
		Prioridad:   t.Prioridad,
		CreadoEn:    t.CreadoEn,
	}
	if t.Vencimiento.Valid {
		v := t.Vencimiento.Time
		task.Vencimiento = &v
	}
	return task
}

// toNullTime es la conversión inversa: *time.Time (nil = sin fecha) a sql.NullTime.
func toNullTime(t *time.Time) sql.NullTime {
	if t == nil {
		return sql.NullTime{}
	}
	return sql.NullTime{Time: *t, Valid: true}
}
