package utils

import (
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
