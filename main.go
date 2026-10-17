package main

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib" // registra el driver "pgx" para database/sql

	db "servidor/db/sqlc"
	"servidor/pkg/handlers"
)

// loggingMiddleware registra cada petición HTTP entrante con método, ruta y dirección remota.
func loggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		log.Printf("[%s] %s desde %s", r.Method, r.URL.Path, r.RemoteAddr)
		next.ServeHTTP(w, r)
		log.Printf("[%s] %s completado en %v", r.Method, r.URL.Path, time.Since(start))
	})
}

// corsMiddleware habilita el consumo seguro de la API desde frontends en navegadores (para el TP4)
// respondiendo las cabeceras requeridas para fetch cruzado y peticiones OPTIONS preflight.
func corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, PATCH, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next.ServeHTTP(w, r)
	})
}

// getDSN obtiene la cadena de conexión a PostgreSQL priorizando DATABASE_URL
// o componiéndola a partir de variables individuales (DB_HOST, DB_PORT, etc) como vimos en las clases.
func getDSN() string {
	if dsn := os.Getenv("DATABASE_URL"); dsn != "" {
		return dsn
	}
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
	return fmt.Sprintf("postgres://%s:%s@%s:%s/%s?sslmode=disable", user, pass, host, port, dbname)
}

func main() {
	// 1. Conexión a la base de datos
	dsn := getDSN()
	conn, err := sql.Open("pgx", dsn)
	if err != nil {
		log.Fatalf("Error al abrir la conexión: %v", err)
	}
	defer conn.Close()

	// sql.Open no conecta inmediatamente: PingContext verifica que la BD responda
	for i := 1; i <= 10; i++ {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		err := conn.PingContext(ctx)
		cancel()
		if err == nil {break}
		time.Sleep(500 * time.Millisecond)
	}
	if err != nil {
		log.Fatalf("No se pudo conectar a la base de datos: %v", err)
	}
	

	// 2. Instancia del repositorio generado por sqlc y handlers CRUD
	h := handlers.New(db.New(conn))

	// 3. Registro de rutas
	mux := http.NewServeMux()
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status":"ok"}`))
	})
	// Endpoints especializados (prioridad por coincidencia más exacta en http.ServeMux)
	mux.HandleFunc("/tasks/metrics", h.MetricsHandler)
	mux.HandleFunc("/tasks/overdue", h.OverdueHandler)
	// Endpoints principales de colección y de recurso individual
	mux.HandleFunc("/tasks", h.TasksHandler)
	mux.HandleFunc("/tasks/", h.TaskHandler)

	// 4. Configuración del puerto e inicio del servidor HTTP
	port := os.Getenv("PORT")
	if port == "" {
		port = ":8080"
	} else if port[0] != ':' {
		port = ":" + port
	}

	serverHandler := loggingMiddleware(corsMiddleware(mux))
	
	fmt.Println("==================================================")
	fmt.Println("   Mi Primera Aplicación Web       ")
	fmt.Println("   Dominio: App de Gestión de Tareas (To-Do)  ")
	fmt.Printf("   Escuchando en: http://localhost%s\n", port)
	fmt.Println("   Presione Ctrl+C para detener el servidor       ")
	fmt.Println("==================================================")

	// 5. Inicia el servidor HTTP
	log.Printf("Servidor API REST iniciado en http://localhost%s/tasks\n", port)
	if err := http.ListenAndServe(port, serverHandler); err != nil {
		log.Fatalf("Error al iniciar el servidor: %v", err)
	}
}
