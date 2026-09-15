package db

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"os"
	"testing"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib"
)

// getTestDB establece y verifica la conexión con la base de datos de pruebas.
func getTestDB(t *testing.T) *sql.DB {
	t.Helper()

	connStr := os.Getenv("DATABASE_URL")
	if connStr == "" {
		connStr = os.Getenv("TEST_DB_URL")
	}
	if connStr == "" {
		host := os.Getenv("DB_HOST")
		if host == "" {
			host = "localhost"
		}
		port := os.Getenv("DB_PORT")
		if port == "" {
			port = "5432"
		}
		user := os.Getenv("DB_USER")
		if user == "" {
			user = "postgres"
		}
		pass := os.Getenv("DB_PASSWORD")
		if pass == "" {
			pass = "postgres"
		}
		dbname := os.Getenv("DB_NAME")
		if dbname == "" {
			dbname = "todolist_test"
		}
		connStr = fmt.Sprintf("postgres://%s:%s@%s:%s/%s?sslmode=disable", user, pass, host, port, dbname)
	}

	db, err := sql.Open("pgx", connStr)
	if err != nil {
		t.Fatalf("error abriendo conexión a la base de datos: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := db.PingContext(ctx); err != nil {
		t.Fatalf("no se pudo conectar a la base de datos de prueba (%s): %v", connStr, err)
	}

	// Asegurar existencia de la tabla en caso de ejecuciones directas
	createTableSQL := `
	CREATE TABLE IF NOT EXISTS tasks (
		id SERIAL PRIMARY KEY,
		titulo VARCHAR(255) NOT NULL,
		descripcion TEXT NOT NULL DEFAULT '',
		estado VARCHAR(50) NOT NULL DEFAULT 'Pendiente',
		prioridad VARCHAR(50) NOT NULL DEFAULT 'Media',
		creado_en TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
		vencimiento TIMESTAMP WITH TIME ZONE
	);`
	if _, err := db.ExecContext(ctx, createTableSQL); err != nil {
		t.Fatalf("error asegurando tabla tasks: %v", err)
	}

	return db
}

// TestQueries_CRUD verifica el ciclo de vida CRUD completo usando el código generado por sqlc.
func TestQueries_CRUD(t *testing.T) {
	dbConn := getTestDB(t)
	defer dbConn.Close()

	queries := New(dbConn)
	ctx := context.Background()

	// Limpieza previa para garantizar aislamiento en las pruebas
	_, err := dbConn.ExecContext(ctx, "DELETE FROM tasks;")
	if err != nil {
		t.Fatalf("falló la limpieza previa de la tabla tasks: %v", err)
	}

	dueDate := time.Now().Add(48 * time.Hour).Truncate(time.Microsecond)
	var createdTask Task

	// 1. Create: Crear una nueva tarea y validar datos retornados
	t.Run("CreateTask", func(t *testing.T) {
		arg := CreateTaskParams{
			Titulo:      "Implementar persistencia TP2",
			Descripcion: "Configurar sqlc y PostgreSQL para la gestión de tareas",
			Estado:      "Pendiente",
			Prioridad:   "Alta",
			Vencimiento: sql.NullTime{Time: dueDate, Valid: true},
		}

		task, err := queries.CreateTask(ctx, arg)
		if err != nil {
			t.Fatalf("CreateTask falló inesperadamente: %v", err)
		}

		if task.ID <= 0 {
			t.Errorf("se esperaba un ID autoincremental positivo, obtenido: %d", task.ID)
		}
		if task.Titulo != arg.Titulo {
			t.Errorf("título esperado %q, obtenido %q", arg.Titulo, task.Titulo)
		}
		if task.Descripcion != arg.Descripcion {
			t.Errorf("descripción esperada %q, obtenido %q", arg.Descripcion, task.Descripcion)
		}
		if task.Estado != arg.Estado {
			t.Errorf("estado esperado %q, obtenido %q", arg.Estado, task.Estado)
		}
		if task.Prioridad != arg.Prioridad {
			t.Errorf("prioridad esperada %q, obtenido %q", arg.Prioridad, task.Prioridad)
		}
		if !task.Vencimiento.Valid || !task.Vencimiento.Time.Equal(dueDate) {
			t.Errorf("vencimiento esperado %v, obtenido %v", dueDate, task.Vencimiento.Time)
		}
		if task.CreadoEn.IsZero() {
			t.Errorf("creado_en no debe ser fecha cero")
		}

		createdTask = task
	})

	// 2. Read: Obtener por ID y verificar que coincida con lo creado
	t.Run("GetTaskByID", func(t *testing.T) {
		task, err := queries.GetTaskByID(ctx, createdTask.ID)
		if err != nil {
			t.Fatalf("GetTaskByID falló para ID %d: %v", createdTask.ID, err)
		}

		if task.ID != createdTask.ID {
			t.Errorf("ID esperado %d, obtenido %d", createdTask.ID, task.ID)
		}
		if task.Titulo != createdTask.Titulo {
			t.Errorf("título esperado %q, obtenido %q", createdTask.Titulo, task.Titulo)
		}
		if task.Estado != createdTask.Estado {
			t.Errorf("estado esperado %q, obtenido %q", createdTask.Estado, task.Estado)
		}
	})

	// 3. Update: Modificar datos y comprobar persistencia
	t.Run("UpdateTask", func(t *testing.T) {
		newDueDate := dueDate.Add(24 * time.Hour)
		updateArg := UpdateTaskParams{
			ID:          createdTask.ID,
			Titulo:      "Implementar persistencia TP2 (En Progreso)",
			Descripcion: "Avanzando en la generación de código y pruebas de integración",
			Estado:      "En Progreso",
			Prioridad:   "Media",
			Vencimiento: sql.NullTime{Time: newDueDate, Valid: true},
		}

		updatedTask, err := queries.UpdateTask(ctx, updateArg)
		if err != nil {
			t.Fatalf("UpdateTask falló: %v", err)
		}

		if updatedTask.ID != createdTask.ID {
			t.Errorf("ID esperado %d, obtenido %d", createdTask.ID, updatedTask.ID)
		}
		if updatedTask.Titulo != updateArg.Titulo {
			t.Errorf("título actualizado esperado %q, obtenido %q", updateArg.Titulo, updatedTask.Titulo)
		}
		if updatedTask.Estado != "En Progreso" {
			t.Errorf("estado esperado 'En Progreso', obtenido %q", updatedTask.Estado)
		}

		// Re-leer desde la base de datos para confirmar persistencia real
		freshTask, err := queries.GetTaskByID(ctx, createdTask.ID)
		if err != nil {
			t.Fatalf("error al re-consultar tarea actualizada: %v", err)
		}
		if freshTask.Titulo != updateArg.Titulo || freshTask.Estado != updateArg.Estado {
			t.Errorf("los datos persistidos no reflejan la actualización")
		}
	})

	// 4. List: Listar todas las tareas y validar presencia del registro
	t.Run("ListTasks", func(t *testing.T) {
		tasks, err := queries.ListTasks(ctx)
		if err != nil {
			t.Fatalf("ListTasks falló: %v", err)
		}

		if len(tasks) == 0 {
			t.Fatalf("se esperaba al menos 1 tarea en la lista, obtenidas 0")
		}

		found := false
		for _, item := range tasks {
			if item.ID == createdTask.ID {
				found = true
				if item.Estado != "En Progreso" {
					t.Errorf("se esperaba que la tarea listada tuviera estado 'En Progreso', obtenido %q", item.Estado)
				}
				break
			}
		}
		if !found {
			t.Errorf("la tarea con ID %d no fue encontrada en la lista retornada", createdTask.ID)
		}
	})

	// 5. Delete: Eliminar registro y validar que consultas posteriores retornen sql.ErrNoRows
	t.Run("DeleteTask", func(t *testing.T) {
		err := queries.DeleteTask(ctx, createdTask.ID)
		if err != nil {
			t.Fatalf("DeleteTask falló: %v", err)
		}

		_, err = queries.GetTaskByID(ctx, createdTask.ID)
		if err == nil {
			t.Fatalf("se esperaba error buscando tarea eliminada, pero no hubo error")
		}
		if !errors.Is(err, sql.ErrNoRows) {
			t.Errorf("se esperaba sql.ErrNoRows, obtenido: %v", err)
		}
	})
}

// TestQueries_Avanzadas prueba las consultas avanzadas y filtros del dominio
func TestQueries_Avanzadas(t *testing.T) {
	dbConn := getTestDB(t)
	defer dbConn.Close()

	queries := New(dbConn)
	ctx := context.Background()

	// Limpiar tabla tasks antes de inicializar el conjunto de datos de prueba
	_, err := dbConn.ExecContext(ctx, "DELETE FROM tasks;")
	if err != nil {
		t.Fatalf("falló la limpieza previa de la tabla tasks: %v", err)
	}

	now := time.Now()

	// 1. Ordenar por estado (pendiente) y fecha vencimiento
	t.Run("OrderByStatusAndDueDate", func(t *testing.T) {
		_, _ = dbConn.ExecContext(ctx, "DELETE FROM tasks;")

		t1, err := queries.CreateTask(ctx, CreateTaskParams{
			Titulo:      "Tarea pendiente lejana",
			Descripcion: "Vence en 10 días",
			Estado:      "Pendiente",
			Prioridad:   "Media",
			Vencimiento: sql.NullTime{Time: now.Add(10 * 24 * time.Hour), Valid: true},
		})
		if err != nil {
			t.Fatalf("error creando t1: %v", err)
		}

		t2, err := queries.CreateTask(ctx, CreateTaskParams{
			Titulo:      "Tarea pendiente cercana",
			Descripcion: "Vence en 2 días",
			Estado:      "Pendiente",
			Prioridad:   "Alta",
			Vencimiento: sql.NullTime{Time: now.Add(2 * 24 * time.Hour), Valid: true},
		})
		if err != nil {
			t.Fatalf("error creando t2: %v", err)
		}

		t3, err := queries.CreateTask(ctx, CreateTaskParams{
			Titulo:      "Tarea pendiente intermedia",
			Descripcion: "Vence en 5 días",
			Estado:      "Pendiente",
			Prioridad:   "Baja",
			Vencimiento: sql.NullTime{Time: now.Add(5 * 24 * time.Hour), Valid: true},
		})
		if err != nil {
			t.Fatalf("error creando t3: %v", err)
		}

		// Tarea ya completada para asegurar que el filtro la excluya
		_, err = queries.CreateTask(ctx, CreateTaskParams{
			Titulo:      "Tarea completada",
			Descripcion: "No debe figurar entre las pendientes",
			Estado:      "Completada",
			Prioridad:   "Alta",
			Vencimiento: sql.NullTime{Time: now.Add(1 * 24 * time.Hour), Valid: true},
		})
		if err != nil {
			t.Fatalf("error creando tarea completada: %v", err)
		}

		// Ejecutar consulta específica de pendientes ordenadas por fecha de vencimiento
		pendingTasks, err := queries.ListPendingTasksOrderByDueDate(ctx)
		if err != nil {
			t.Fatalf("ListPendingTasksOrderByDueDate falló: %v", err)
		}

		if len(pendingTasks) != 3 {
			t.Fatalf("se esperaban exactamente 3 tareas pendientes, obtenidas: %d", len(pendingTasks))
		}

		// Verificar que todas sean efectivamente 'Pendiente'
		for _, pt := range pendingTasks {
			if pt.Estado != "Pendiente" {
				t.Errorf("tarea %d tiene estado %q, se esperaba 'Pendiente'", pt.ID, pt.Estado)
			}
		}

		// Verificar orden cronológico ascendente con casos estructurados
		casosOrden := []struct {
			posicion   int
			idEsperado int32
			desc       string
		}{
			{0, t2.ID, "menor vencimiento (2 días)"},
			{1, t3.ID, "vencimiento intermedio (5 días)"},
			{2, t1.ID, "vencimiento más lejano (10 días)"},
		}
		for _, tc := range casosOrden {
			if pendingTasks[tc.posicion].ID != tc.idEsperado {
				t.Errorf("en posición %d (%s) se esperaba ID %d, obtenido %d",
					tc.posicion, tc.desc, tc.idEsperado, pendingTasks[tc.posicion].ID)
			}
		}
	})

	// 2. Filtrado por prioridad (Table-Driven Test)
	t.Run("FilterByPriority", func(t *testing.T) {
		_, _ = dbConn.ExecContext(ctx, "DELETE FROM tasks;")

		_, _ = queries.CreateTask(ctx, CreateTaskParams{Titulo: "Alta 1", Prioridad: "Alta", Estado: "Pendiente"})
		_, _ = queries.CreateTask(ctx, CreateTaskParams{Titulo: "Alta 2", Prioridad: "Alta", Estado: "En Progreso"})
		_, _ = queries.CreateTask(ctx, CreateTaskParams{Titulo: "Media 1", Prioridad: "Media", Estado: "Pendiente"})
		_, _ = queries.CreateTask(ctx, CreateTaskParams{Titulo: "Baja 1", Prioridad: "Baja", Estado: "Pendiente"})

		casos := []struct {
			nombre          string
			prioridadFiltro string
			esperadas       int
		}{
			{"Prioridad Alta", "Alta", 2},
			{"Prioridad Media", "Media", 1},
			{"Prioridad Baja", "Baja", 1},
			{"Prioridad Inexistente", "Inexistente", 0},
		}

		for _, tc := range casos {
			t.Run(tc.nombre, func(t *testing.T) {
				tasks, err := queries.ListTasksByPriority(ctx, tc.prioridadFiltro)
				if err != nil {
					t.Fatalf("ListTasksByPriority falló para %q: %v", tc.prioridadFiltro, err)
				}
				if len(tasks) != tc.esperadas {
					t.Errorf("ListTasksByPriority(%q) = %d tareas; se esperaban %d", tc.prioridadFiltro, len(tasks), tc.esperadas)
				}
				for _, task := range tasks {
					if task.Prioridad != tc.prioridadFiltro {
						t.Errorf("tarea ID %d tiene prioridad %q; se esperaba %q", task.ID, task.Prioridad, tc.prioridadFiltro)
					}
				}
			})
		}
	})

	// 3. Tareas completadas en los últimos x días (Table-Driven Test)
	t.Run("CompletedTasksInLastDays", func(t *testing.T) {
		_, _ = dbConn.ExecContext(ctx, "DELETE FROM tasks;")

		// Tarea 1: Completada reciente (creada hoy)
		_, err := queries.CreateTask(ctx, CreateTaskParams{
			Titulo:    "Completada hoy",
			Estado:    "Completada",
			Prioridad: "Media",
		})
		if err != nil {
			t.Fatalf("error creando tarea completada reciente: %v", err)
		}

		// Tarea 2: Pendiente reciente (no debe figurar porque no está completada)
		_, err = queries.CreateTask(ctx, CreateTaskParams{
			Titulo:    "Pendiente hoy",
			Estado:    "Pendiente",
			Prioridad: "Alta",
		})
		if err != nil {
			t.Fatalf("error creando tarea pendiente: %v", err)
		}

		// Tarea 3: Completada pero antigua (creada hace 10 días vía SQL directo)
		oldDate := now.AddDate(0, 0, -10)
		_, err = dbConn.ExecContext(ctx, `
			INSERT INTO tasks (titulo, descripcion, estado, prioridad, creado_en)
			VALUES ('Completada hace 10 días', '', 'Completada', 'Baja', $1);
		`, oldDate)
		if err != nil {
			t.Fatalf("error insertando tarea completada antigua: %v", err)
		}

		casos := []struct {
			nombre    string
			dias      int32
			esperadas int
		}{
			{"Ventana de 3 días (solo reciente)", 3, 1},
			{"Ventana de 15 días (reciente y antigua)", 15, 2},
		}

		for _, tc := range casos {
			t.Run(tc.nombre, func(t *testing.T) {
				tasks, err := queries.ListCompletedTasksInLastDays(ctx, tc.dias)
				if err != nil {
					t.Fatalf("ListCompletedTasksInLastDays falló para %d días: %v", tc.dias, err)
				}
				if len(tasks) != tc.esperadas {
					t.Errorf("ListCompletedTasksInLastDays(%d) = %d tareas; se esperaban %d", tc.dias, len(tasks), tc.esperadas)
				}
				for _, task := range tasks {
					if task.Estado != "Completada" {
						t.Errorf("tarea ID %d tiene estado %q; se esperaba 'Completada'", task.ID, task.Estado)
					}
					if task.TotalCompletadas != int32(tc.esperadas) {
						t.Errorf("campo total_completadas de ventana esperado %d; obtenido %d", tc.esperadas, task.TotalCompletadas)
					}
				}
			})
		}
	})

	// 4. Tareas que vencen en los siguientes x días (Table-Driven Test)
	t.Run("TasksDueInNextDays", func(t *testing.T) {
		_, _ = dbConn.ExecContext(ctx, "DELETE FROM tasks;")

		// Tarea A: Vence en 2 días (dentro del rango de 5 días)
		_, _ = queries.CreateTask(ctx, CreateTaskParams{
			Titulo:      "Vence pronto (2 días)",
			Estado:      "Pendiente",
			Prioridad:   "Alta",
			Vencimiento: sql.NullTime{Time: now.Add(2 * 24 * time.Hour), Valid: true},
		})

		// Tarea B: Vence en 4 días (dentro del rango de 5 días)
		_, _ = queries.CreateTask(ctx, CreateTaskParams{
			Titulo:      "Vence en 4 días",
			Estado:      "Pendiente",
			Prioridad:   "Media",
			Vencimiento: sql.NullTime{Time: now.Add(4 * 24 * time.Hour), Valid: true},
		})

		// Tarea C: Vence en 15 días (fuera del rango de 5 días)
		_, _ = queries.CreateTask(ctx, CreateTaskParams{
			Titulo:      "Vence lejos (15 días)",
			Estado:      "Pendiente",
			Prioridad:   "Baja",
			Vencimiento: sql.NullTime{Time: now.Add(15 * 24 * time.Hour), Valid: true},
		})

		// Tarea D: Venció ayer (en el pasado, no debe incluirse)
		_, _ = queries.CreateTask(ctx, CreateTaskParams{
			Titulo:      "Venció ayer",
			Estado:      "Pendiente",
			Prioridad:   "Media",
			Vencimiento: sql.NullTime{Time: now.Add(-24 * time.Hour), Valid: true},
		})

		// Tarea E: Sin vencimiento (NULL)
		_, _ = queries.CreateTask(ctx, CreateTaskParams{
			Titulo:      "Sin fecha de vencimiento",
			Estado:      "Pendiente",
			Prioridad:   "Baja",
			Vencimiento: sql.NullTime{Valid: false},
		})

		casos := []struct {
			nombre    string
			dias      int32
			esperadas int
		}{
			{"Ventana de 1 día (ninguna en rango)", 1, 0},
			{"Ventana de 3 días (incluye tarea de 2 días)", 3, 1},
			{"Ventana de 5 días (incluye tareas de 2 y 4 días)", 5, 2},
			{"Ventana de 20 días (incluye tareas de 2, 4 y 15 días)", 20, 3},
		}

		for _, tc := range casos {
			t.Run(tc.nombre, func(t *testing.T) {
				dueTasks, err := queries.ListTasksDueInNextDays(ctx, tc.dias)
				if err != nil {
					t.Fatalf("ListTasksDueInNextDays falló para %d días: %v", tc.dias, err)
				}
				if len(dueTasks) != tc.esperadas {
					t.Errorf("ListTasksDueInNextDays(%d) = %d tareas; se esperaban %d", tc.dias, len(dueTasks), tc.esperadas)
				}
			})
		}
	})

	// 5. Búsqueda global de palabra clave en título y descripción (Table-Driven Test)
	t.Run("SearchTasks", func(t *testing.T) {
		_, _ = dbConn.ExecContext(ctx, "DELETE FROM tasks;")

		// Tarea con palabra clave en el título
		t1, _ := queries.CreateTask(ctx, CreateTaskParams{
			Titulo:      "Reunión de Arquitectura del sistema",
			Descripcion: "Coordinar diseño general con el equipo",
			Estado:      "Pendiente",
			Prioridad:   "Alta",
		})

		// Tarea con palabra clave en la descripción
		t2, _ := queries.CreateTask(ctx, CreateTaskParams{
			Titulo:      "Módulo de datos",
			Descripcion: "Implementar la capa de ARQUITECTURA con sqlc",
			Estado:      "En Progreso",
			Prioridad:   "Media",
		})

		// Tarea sin la palabra clave
		_, _ = queries.CreateTask(ctx, CreateTaskParams{
			Titulo:      "Comprar café",
			Descripcion: "Insumos para la cocina",
			Estado:      "Pendiente",
			Prioridad:   "Baja",
		})

		casos := []struct {
			nombre    string
			query     string
			esperadas int
		}{
			{"Coincidencia en título y descripción", "arquitectura", 2},
			{"Insensible a mayúsculas", "ARQUITECTURA", 2},
			{"Coincidencia parcial en descripción", "sqlc", 1},
			{"Coincidencia parcial en título", "Reunión", 1},
			{"Término inexistente", "termino_inexistente_123", 0},
		}

		for _, tc := range casos {
			t.Run(tc.nombre, func(t *testing.T) {
				results, err := queries.SearchTasks(ctx, tc.query)
				if err != nil {
					t.Fatalf("SearchTasks falló para query %q: %v", tc.query, err)
				}
				if len(results) != tc.esperadas {
					t.Errorf("SearchTasks(%q) = %d resultados; se esperaban %d", tc.query, len(results), tc.esperadas)
				}
				if tc.query == "arquitectura" {
					foundT1, foundT2 := false, false
					for _, r := range results {
						if r.ID == t1.ID {
							foundT1 = true
						}
						if r.ID == t2.ID {
							foundT2 = true
						}
					}
					if !foundT1 || !foundT2 {
						t.Errorf("no se encontraron ambas tareas coincidentes (t1=%v, t2=%v)", foundT1, foundT2)
					}
				}
			})
		}
	})

	// 6. Métricas de tareas para dashboard (GetTaskMetrics)
	t.Run("GetTaskMetrics", func(t *testing.T) {
		_, _ = dbConn.ExecContext(ctx, "DELETE FROM tasks;")

		// 2 Pendientes (una a tiempo, una vencida)
		_, _ = queries.CreateTask(ctx, CreateTaskParams{
			Titulo: "Pendiente a tiempo", Estado: "Pendiente",
			Vencimiento: sql.NullTime{Time: now.Add(24 * time.Hour), Valid: true},
		})
		_, _ = queries.CreateTask(ctx, CreateTaskParams{
			Titulo: "Pendiente vencida", Estado: "Pendiente",
			Vencimiento: sql.NullTime{Time: now.Add(-24 * time.Hour), Valid: true},
		})

		// 1 En Progreso (a tiempo)
		_, _ = queries.CreateTask(ctx, CreateTaskParams{
			Titulo: "En progreso", Estado: "En Progreso",
			Vencimiento: sql.NullTime{Time: now.Add(48 * time.Hour), Valid: true},
		})

		// 2 Completadas (una con vencimiento en el pasado, pero al estar completada NO debe contar como vencida)
		_, _ = queries.CreateTask(ctx, CreateTaskParams{
			Titulo: "Completada 1", Estado: "Completada",
			Vencimiento: sql.NullTime{Time: now.Add(-48 * time.Hour), Valid: true},
		})
		_, _ = queries.CreateTask(ctx, CreateTaskParams{
			Titulo: "Completada 2", Estado: "Completada",
		})

		metrics, err := queries.GetTaskMetrics(ctx)
		if err != nil {
			t.Fatalf("GetTaskMetrics falló: %v", err)
		}

		if metrics.Total != 5 {
			t.Errorf("total esperado 5, obtenido: %d", metrics.Total)
		}
		if metrics.Pendientes != 2 {
			t.Errorf("pendientes esperadas 2, obtenidas: %d", metrics.Pendientes)
		}
		if metrics.EnProgreso != 1 {
			t.Errorf("en progreso esperadas 1, obtenidas: %d", metrics.EnProgreso)
		}
		if metrics.Completadas != 2 {
			t.Errorf("completadas esperadas 2, obtenidas: %d", metrics.Completadas)
		}
		if metrics.Vencidas != 1 {
			t.Errorf("vencidas esperadas 1, obtenidas: %d", metrics.Vencidas)
		}
	})

	// 7. Detección de tareas vencidas (ListOverdueTasks)
	t.Run("ListOverdueTasks", func(t *testing.T) {
		_, _ = dbConn.ExecContext(ctx, "DELETE FROM tasks;")

		// Tarea vencida pendiente (debe incluirse)
		vencida, _ := queries.CreateTask(ctx, CreateTaskParams{
			Titulo:      "Tarea vencida atrasada",
			Estado:      "Pendiente",
			Prioridad:   "Alta",
			Vencimiento: sql.NullTime{Time: now.Add(-48 * time.Hour), Valid: true},
		})

		// Tarea vencida pero completada (NO debe incluirse)
		_, _ = queries.CreateTask(ctx, CreateTaskParams{
			Titulo:      "Tarea vieja completada a tiempo",
			Estado:      "Completada",
			Prioridad:   "Media",
			Vencimiento: sql.NullTime{Time: now.Add(-24 * time.Hour), Valid: true},
		})

		// Tarea futura no vencida (NO debe incluirse)
		_, _ = queries.CreateTask(ctx, CreateTaskParams{
			Titulo:      "Tarea futura",
			Estado:      "Pendiente",
			Prioridad:   "Media",
			Vencimiento: sql.NullTime{Time: now.Add(24 * time.Hour), Valid: true},
		})

		overdue, err := queries.ListOverdueTasks(ctx)
		if err != nil {
			t.Fatalf("ListOverdueTasks falló: %v", err)
		}

		if len(overdue) != 1 {
			t.Fatalf("se esperaba exactamente 1 tarea vencida, obtenidas: %d", len(overdue))
		}
		if overdue[0].ID != vencida.ID {
			t.Errorf("ID de tarea vencida esperado %d, obtenido %d", vencida.ID, overdue[0].ID)
		}
	})

	// 8. Acción rápida: Marcar tarea como completada (MarkTaskCompleted)
	t.Run("MarkTaskCompleted", func(t *testing.T) {
		_, _ = dbConn.ExecContext(ctx, "DELETE FROM tasks;")

		task, err := queries.CreateTask(ctx, CreateTaskParams{
			Titulo:    "Tarea para completar",
			Estado:    "Pendiente",
			Prioridad: "Alta",
		})
		if err != nil {
			t.Fatalf("error creando tarea: %v", err)
		}

		updated, err := queries.MarkTaskCompleted(ctx, task.ID)
		if err != nil {
			t.Fatalf("MarkTaskCompleted falló: %v", err)
		}

		if updated.Estado != "Completada" {
			t.Errorf("estado esperado 'Completada', obtenido %q", updated.Estado)
		}

		// Re-leer para verificar persistencia real
		fresh, err := queries.GetTaskByID(ctx, task.ID)
		if err != nil {
			t.Fatalf("error al re-consultar tarea: %v", err)
		}
		if fresh.Estado != "Completada" {
			t.Errorf("estado persistido esperado 'Completada', obtenido %q", fresh.Estado)
		}
	})

	// 9. Limpieza en lote de tareas completadas (DeleteCompletedTasks)
	t.Run("DeleteCompletedTasks", func(t *testing.T) {
		_, _ = dbConn.ExecContext(ctx, "DELETE FROM tasks;")

		// 2 completadas y 1 pendiente
		_, _ = queries.CreateTask(ctx, CreateTaskParams{Titulo: "C1", Estado: "Completada"})
		_, _ = queries.CreateTask(ctx, CreateTaskParams{Titulo: "C2", Estado: "Completada"})
		p1, _ := queries.CreateTask(ctx, CreateTaskParams{Titulo: "P1", Estado: "Pendiente"})

		res, err := queries.DeleteCompletedTasks(ctx)
		if err != nil {
			t.Fatalf("DeleteCompletedTasks falló: %v", err)
		}

		rowsAffected, err := res.RowsAffected()
		if err != nil {
			t.Fatalf("error al leer filas afectadas: %v", err)
		}
		if rowsAffected != 2 {
			t.Errorf("se esperaban 2 filas afectadas por el borrado, obtenidas: %d", rowsAffected)
		}

		// Verificar que solo la tarea pendiente permanece
		remaining, err := queries.ListTasks(ctx)
		if err != nil {
			t.Fatalf("error listando tareas restantes: %v", err)
		}
		if len(remaining) != 1 {
			t.Fatalf("se esperaba 1 tarea restante, obtenidas: %d", len(remaining))
		}
		if remaining[0].ID != p1.ID {
			t.Errorf("la tarea restante debía ser P1 (ID %d), obtenida ID %d", p1.ID, remaining[0].ID)
		}
	})
}


