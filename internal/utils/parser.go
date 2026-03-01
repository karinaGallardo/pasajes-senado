package utils

import (
	"fmt"
	"strconv"
	"time"
)

// ParseFloat convierte un string a float64. Retorna 0 si falla.
func ParseFloat(s string) float64 {
	val, _ := strconv.ParseFloat(s, 64)
	return val
}

// ParseDate convierte un string a time.Time usando el formato proporcionado.
func ParseDate(layout, value string) time.Time {
	t, _ := time.Parse(layout, value)
	return t
}

// ParseDatePtr convierte un string a *time.Time. Retorna nil si el string es vacío o falla.
func ParseDatePtr(layout, value string) *time.Time {
	if value == "" {
		return nil
	}
	t, err := time.Parse(layout, value)
	if err != nil {
		return nil
	}
	return &t
}

// ParseDateTime intenta parsear un string de fecha y hora en formato T (ISO) o espacio (estándar),
// común en selectores de calendario y navegadores.
func ParseDateTime(val string) (*time.Time, error) {
	if val == "" {
		return nil, nil
	}
	// Formato T (usado en inputs datetime-local o modales específicos)
	if t, err := time.Parse("2006-01-02T15:04", val); err == nil {
		return &t, nil
	}
	// Formato espacio (default de Flatpickr)
	if t, err := time.Parse("2006-01-02 15:04", val); err == nil {
		return &t, nil
	}
	return nil, fmt.Errorf("formato de fecha y hora inválido: %s", val)
}
