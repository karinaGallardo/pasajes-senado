package utils

// ToMap convierte un slice de strings en un mapa para búsqueda rápida O(1)
func ToMap(s []string) map[string]bool {
	m := make(map[string]bool)
	for _, v := range s {
		m[v] = true
	}
	return m
}

// GetIdx obtiene un elemento de un slice de forma segura por su índice.
// Si el índice está fuera de rango, retorna el valor cero del tipo.
func GetIdx[T any](s []T, i int) T {
	var zero T
	if i < 0 || i >= len(s) {
		return zero
	}
	return s[i]
}

// GetIdxOrDefault obtiene un elemento de un slice de forma segura por su índice.
// Si el índice está fuera de rango, retorna el valor d (default).
func GetIdxOrDefault[T any](s []T, i int, d T) T {
	if i < 0 || i >= len(s) {
		return d
	}
	return s[i]
}

// ParseDatePtr parsea una fecha y retorna un puntero a time.Time.
// Si la fecha es inválida o vacía, retorna nil.
// Ptr retorna un puntero a cualquier valor (Ya está en ptr.go, pero lo mantengo referencialmente si se borrara de allá)
// func Ptr[T any](v T) *T { return &v }
