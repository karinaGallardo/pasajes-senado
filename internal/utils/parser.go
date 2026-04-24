package utils

import (
	"fmt"
	"strconv"
	"strings"
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
	// Normalizar a mayúsculas para facilitar el parseo de AM/PM
	val = strings.ToUpper(val)
	// Formater T (usado en inputs datetime-local o modales específicos)
	if t, err := time.ParseInLocation("2006-01-02T15:04", val, time.Local); err == nil {
		return &t, nil
	}
	// Formato espacio (default de Flatpickr/AirDatepicker 24h)
	if t, err := time.ParseInLocation("2006-01-02 15:04", val, time.Local); err == nil {
		return &t, nil
	}
	// Formato 12h (AM/PM) con espacio
	if t, err := time.ParseInLocation("2006-01-02 03:04 PM", val, time.Local); err == nil {
		return &t, nil
	}
	// Formato 12h (AM/PM) con T
	if t, err := time.ParseInLocation("2006-01-02T03:04 PM", val, time.Local); err == nil {
		return &t, nil
	}
	// === NUEVOS FORMATOS SOPORTADOS (DD/MM/YYYY) ===
	// Formato latino estándar (solo fecha)
	if t, err := time.ParseInLocation("02/01/2006", val, time.Local); err == nil {
		return &t, nil
	}
	// Formato latino con hora 24h
	if t, err := time.ParseInLocation("02/01/2006 15:04", val, time.Local); err == nil {
		return &t, nil
	}
	// Formato ISO estándar (solo fecha con guiones)
	if t, err := time.ParseInLocation("2006-01-02", val, time.Local); err == nil {
		return &t, nil
	}
	// Formato latino con guiones
	if t, err := time.ParseInLocation("02-01-2006", val, time.Local); err == nil {
		return &t, nil
	}
	if t, err := time.ParseInLocation("02-01-2006 15:04", val, time.Local); err == nil {
		return &t, nil
	}
	return nil, fmt.Errorf("formato de fecha y hora inválido: %s", val)
}

// ParseDateAndTime combina un string de fecha y otro de hora en un solo time.Time
func ParseDateAndTime(dateStr, timeStr string) (*time.Time, error) {
	if dateStr == "" {
		return nil, nil
	}
	if timeStr == "" {
		timeStr = "00:00"
	}
	// Normalizar hora (ej: "8:00" -> "08:00")
	parts := strings.Split(timeStr, ":")
	if len(parts) == 2 && len(parts[0]) == 1 {
		timeStr = "0" + timeStr
	}

	combined := dateStr + " " + timeStr
	return ParseDateTime(combined)
}

func StrToInt(s string, def int) int {
	if s == "" {
		return def
	}
	val, err := strconv.Atoi(s)
	if err != nil {
		return def
	}
	return val
}
