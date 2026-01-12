package utils

import "fmt"

// GetString castea un valor any a string de forma segura. Retorna "" si es nil.
func GetString(val any) string {
	if val == nil {
		return ""
	}
	if str, ok := val.(string); ok {
		return str
	}
	return fmt.Sprintf("%v", val)
}
