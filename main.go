package main

import (
	"fmt"
	"net/http"
	"time"
)

// Task representa la entidad principal del dominio (To-Do List).
// Se ubica fuera de main() para ser accesible en todo el paquete.
type Task struct {
	ID          int       `json:"id"`
	Titulo      string    `json:"titulo"`
	Descripcion string    `json:"descripcion"`
	Estado      string    `json:"estado"`    // "Pendiente", "En Progreso", "Completada"
	Prioridad   string    `json:"prioridad"` // "Baja", "Media", "Alta"
	CreadoEn    time.Time `json:"creado_en"`
	Vencimiento time.Time `json:"vencimiento"`
}

func main() {
	// Definimos un manejador para la ruta "/"
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		// 1. Manejo de rutas inexistente con error (404)
		if r.URL.Path != "/" {
			http.NotFound(w, r)
			return
		}
		// 2. Valida que la petición sea solo mediante GET
		if r.Method != http.MethodGet {
			http.Error(w, "Método no permitido", http.StatusMethodNotAllowed)
			return
		}
		// 3. Establece la cabecera Content-Type
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		// 4. Lee y escribe el archivo index en la respuesta
		http.ServeFile(w, r, "index.html")
	})

	port := ":8080"
	fmt.Println("==================================================")
	fmt.Println("   Mi Primera Aplicación Web       ")
	fmt.Println("   Dominio: App de Gestión de Tareas (To-Do)  ")
	fmt.Printf("   Escuchando en: http://localhost%s\n", port)
	fmt.Println("   Presione Ctrl+C para detener el servidor       ")
	fmt.Println("==================================================")

	// 5. Inicia el servidor HTTP
	err := http.ListenAndServe(port, nil)
	if err != nil {
		fmt.Printf("Error al iniciar el servidor: %s\n", err)
	}
}
