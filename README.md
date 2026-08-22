# Trabajo de Cursada: Mi Primera Aplicación Web
## App de Gestión de Tareas (To-Do List) — Grupo 22

Esta primer instancia de entrega de tp es una aplicación web básica desarrollada en **Go (Golang)** que levanta un servidor HTTP nativo para presentar la definición del dominio de una aplicación de gestión de tareas (*To-Do List*).

---

## 1. Elección del Dominio
- **Dominio elegido:** Lista de Tareas (*To-Do List*).
- **Entidad principal:** `Task` (Tarea).
- **Atributos definidos:**
  - `ID` (`int`): Identificador numérico único de la tarea.
  - `Titulo` (`string`): Nombre o título breve de la tarea.
  - `Descripcion` (`string`): Detalle explicativo de la tarea.
  - `Estado` (`string`): Estado actual (*"Pendiente"*, *"En Progreso"*, *"Completada"*).
  - `Prioridad` (`string`): Importancia (*"Baja"*, *"Media"*, *"Alta"*).
  - `CreadoEn` (`time.Time`): Fecha y hora de creación.
  - `Vencimiento` (`time.Time`): Fecha y hora en la que vence la tarea.

---

## 2. Estructura del Proyecto

```text
├── go.mod        # Definición del módulo de Go
├── index.html    # Página web con la presentación y definición del dominio de datos
├── main.go       # Servidor web HTTP en Go
└── README.md     # Documentación del TP e instrucciones de ejecución
```

---

## 3. Requisitos y Ejecución

### Requisitos Previos

Para ejecutar este proyecto necesitas tener instalado:
* **[Go](https://go.dev/dl/)** (versión 1.20 o superior recomendada).

Puedes verificar tu instalación ejecutando en la terminal:
```bash
go version
```

---

### Cómo ejecutar el servidor

1. **Abrir la terminal** en el directorio raíz del proyecto:
   ```bash
   cd "ruta_del_proyecto"
   ```

2. **Ejecución directa del servidor con Go**:
   ```bash
   go run main.go
   ```

   *(Opcional) Si prefieres compilar el binario primero:*
   ```bash
   # Compilación
   go build -o servidor main.go

   # Ejecución en Windows (PowerShell / CMD) para nada recomendable
   .\servidor.exe

   # Ejecución en Linux / macOS
   ./servidor
   ```

3. **Acceder desde el navegador**:
   Abre tu navegador web e ingresa a:
   [http://localhost:8080](http://localhost:8080)

4. **Detener el servidor**:
   Para finalizar la ejecución del servidor, presiona `Ctrl + C` en la terminal.

---

## Características de esta primer implementación

* **Control de rutas**: Responde exclusivamente en la ruta raíz (`/`), retornando código de estado `404 Not Found` para cualquier ruta inexistente.
* **Validación de método HTTP**: Permite únicamente solicitudes mediante el método `GET`, respondiendo con código `405 Method Not Allowed` ante otros métodos.
* **Cabeceras HTTP**: Configura la cabecera `Content-Type: text/html; charset=utf-8`.