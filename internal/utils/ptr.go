package utils

// Ptr retorna un puntero al valor proporcionado.
//
//go:fix inline
func Ptr[T any](v T) *T {
	return &v
}

// NilIfEmpty retorna nil si el string está vacío, de lo contrario un puntero al string.
func NilIfEmpty(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}
