package models

import (
	"strings"
	"testing"
)

// TestValidateTask verifica exhaustivamente las reglas de validación del dominio
// utilizando la metodología de Table-Driven Tests enseñada en la clase.
func TestValidateTask(t *testing.T) {
	casos := []struct {
		nombre        string
		task          Task
		esperaError   bool
		errorContiene string
	}{
		{
			nombre: "Tarea válida con todos los campos",
			task: Task{
				Titulo:    "Estudiar para el parcial",
				Estado:    EstadoPendiente,
				Prioridad: PrioridadAlta,
			},
			esperaError: false,
		},
		{
			nombre: "Título vacío debe fallar",
			task: Task{
				Titulo:    "",
				Estado:    EstadoPendiente,
				Prioridad: PrioridadMedia,
			},
			esperaError:   true,
			errorContiene: "el titulo es obligatorio",
		},
		{
			nombre: "Título que excede 255 caracteres debe fallar",
			task: Task{
				Titulo:    strings.Repeat("a", 256),
				Estado:    EstadoPendiente,
				Prioridad: PrioridadBaja,
			},
			esperaError:   true,
			errorContiene: "no puede superar los 255 caracteres",
		},
		{
			nombre: "Estado inválido debe fallar",
			task: Task{
				Titulo:    "Tarea con estado erróneo",
				Estado:    "Cancelada",
				Prioridad: PrioridadMedia,
			},
			esperaError:   true,
			errorContiene: "el estado debe ser Pendiente, En Progreso o Completada",
		},
		{
			nombre: "Prioridad inválida debe fallar",
			task: Task{
				Titulo:    "Tarea con prioridad errónea",
				Estado:    EstadoPendiente,
				Prioridad: "Urgente",
			},
			esperaError:   true,
			errorContiene: "la prioridad debe ser Baja, Media o Alta",
		},
	}

	for _, tc := range casos {
		t.Run(tc.nombre, func(t *testing.T) {
			err := ValidateTask(tc.task)
			if tc.esperaError {
				if err == nil {
					t.Fatalf("se esperaba un error pero la validación fue exitosa")
				}
				if !strings.Contains(err.Error(), tc.errorContiene) {
					t.Errorf("error obtenido %q no contiene el texto esperado %q", err.Error(), tc.errorContiene)
				}
			} else {
				if err != nil {
					t.Fatalf("no se esperaba error, obtenido: %v", err)
				}
			}
		})
	}
}

// TestApplyDefaults verifica que los espacios en blanco se recorten y que
// los campos opcionales adopten los valores por defecto del dominio.
func TestApplyDefaults(t *testing.T) {
	entrada := Task{
		Titulo:      "   Comprar insumos   ",
		Descripcion: "   Detalle del pedido   ",
		Estado:      "",
		Prioridad:   "",
	}

	resultado := ApplyDefaults(entrada)

	if resultado.Titulo != "Comprar insumos" {
		t.Errorf("título esperado 'Comprar insumos', obtenido %q", resultado.Titulo)
	}
	if resultado.Descripcion != "Detalle del pedido" {
		t.Errorf("descripción esperada 'Detalle del pedido', obtenido %q", resultado.Descripcion)
	}
	if resultado.Estado != EstadoPendiente {
		t.Errorf("estado por defecto esperado %q, obtenido %q", EstadoPendiente, resultado.Estado)
	}
	if resultado.Prioridad != PrioridadMedia {
		t.Errorf("prioridad por defecto esperada %q, obtenido %q", PrioridadMedia, resultado.Prioridad)
	}
}
