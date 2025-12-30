package utils

import (
	"time"
)

func CalcularFechaLimiteDescargo(fechaRetorno time.Time) time.Time {
	dias := 0
	fecha := fechaRetorno

	for dias < 8 {
		fecha = fecha.AddDate(0, 0, 1)
		weekday := fecha.Weekday()
		if weekday != time.Saturday && weekday != time.Sunday {
			dias++
		}
	}
	return fecha
}
