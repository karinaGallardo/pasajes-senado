package utils

// Ptr retorna un puntero al valor proporcionado.
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

// Deref retorna el valor apuntado por el puntero, o el valor cero del tipo si el puntero es nil.
func Deref[T any](ptr *T) T {
	if ptr == nil {
		var zero T
		return zero
	}
	return *ptr
}
