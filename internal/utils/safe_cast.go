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

// GetStringFromBson extrae un string de un mapa bson.M de forma segura. Retorna "" si falla.
func GetStringFromBson(m map[string]any, key string) string {
	if val, ok := m[key]; ok {
		if str, ok := val.(string); ok {
			return str
		}
	}
	return ""
}
