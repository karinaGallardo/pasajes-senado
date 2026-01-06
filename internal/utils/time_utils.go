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

func TranslateMonth(month time.Month) string {
	mesES := map[string]string{
		"January":   "ENERO",
		"February":  "FEBRERO",
		"March":     "MARZO",
		"April":     "ABRIL",
		"May":       "MAYO",
		"June":      "JUNIO",
		"July":      "JULIO",
		"August":    "AGOSTO",
		"September": "SEPTIEMBRE",
		"October":   "OCTUBRE",
		"November":  "NOVIEMBRE",
		"December":  "DICIEMBRE",
	}
	return mesES[month.String()]
}

func GetMonthNames() []string {
	return []string{"", "Enero", "Febrero", "Marzo", "Abril", "Mayo", "Junio", "Julio", "Agosto", "Septiembre", "Octubre", "Noviembre", "Diciembre"}
}
