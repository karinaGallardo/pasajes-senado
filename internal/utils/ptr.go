package utils

// Ptr retorna un puntero al valor proporcionado.
func Ptr[T any](v T) *T {
	return &v
}
