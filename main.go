package main

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib" // registra el driver "pgx" para database/sql

	db "servidor/db/sqlc"
	"servidor/pkg/handlers"
)

// ensureAndLoadEnv verifica la existencia del archivo .env. Si no existe, lo genera
// automáticamente a partir de .env.example (si este existe). Luego, lee y carga
// las variables definidas en .env en el entorno del proceso.
func ensureAndLoadEnv() {
	if _, err := os.Stat(".env"); os.IsNotExist(err) {
		if exampleData, err := os.ReadFile(".env.example"); err == nil {
			if err := os.WriteFile(".env", exampleData, 0644); err == nil {
				log.Println("Archivo .env creado automáticamente a partir de .env.example")
			}
		}
	}

	data, err := os.ReadFile(".env")
	if err != nil {
		return
	}

	for _, line := range strings.Split(string(data), "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		parts := strings.SplitN(line, "=", 2)
		if len(parts) == 2 {
			key := strings.TrimSpace(parts[0])
			val := strings.TrimSpace(parts[1])
			val = strings.Trim(val, `"'`)
			if os.Getenv(key) == "" {
				_ = os.Setenv(key, val)
			}
		}
	}
}

// loggingMiddleware registra cada petición HTTP entrante con método, ruta y dirección remota.
func loggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		log.Printf("[%s] %s desde %s", r.Method, r.URL.Path, r.RemoteAddr)
		next.ServeHTTP(w, r)
		log.Printf("[%s] %s completado en %v", r.Method, r.URL.Path, time.Since(start))
	})
}

// getDSN obtiene la cadena de conexión a PostgreSQL priorizando DATABASE_URL
// o componiéndola a partir de variables individuales (DB_HOST, DB_PORT, etc).
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
	// Inicialización automática de variables de entorno desde .env / .env.example
	ensureAndLoadEnv()

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

	// 2. Instancia del repositorio generado por sqlc y handlers HTTP
	queries := db.New(conn)
	h := handlers.New(queries)

	// 3. Registro de rutas
	mux := http.NewServeMux()

	// Endpoint de healthcheck
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status":"ok"}`))
	})

	// Handlers principales de la aplicación SSR con Templ (patrón PRG)
	mux.HandleFunc("/", h.IndexHandler)
	mux.HandleFunc("/tasks", h.CreateTaskHandler)

	// 4. Configuración del puerto e inicio del servidor HTTP
	port := os.Getenv("PORT")
	if port == "" {
		port = ":8080"
	} else if port[0] != ':' {
		port = ":" + port
	}

	serverHandler := loggingMiddleware(mux)

	fmt.Println("==================================================")
	fmt.Println("   Mi Primera Aplicación Web      ")
	fmt.Println("   Dominio: App de Gestión de Tareas (To-Do)     ")
	fmt.Printf("   Escuchando en: http://localhost%s\n", port)
	fmt.Println("   Presione Ctrl+C para detener el servidor       ")
	fmt.Println("==================================================")

	// 5. Inicia el servidor HTTP
	log.Printf("Servidor SSR iniciado en http://localhost%s/\n", port)
	if err := http.ListenAndServe(port, serverHandler); err != nil {
		log.Fatalf("Error al iniciar el servidor: %v", err)
	}
}
