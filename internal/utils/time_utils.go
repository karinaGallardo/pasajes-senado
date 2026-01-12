package utils

import (
	"time"
)

// CalcularFechaLimiteDescargo retorna la fecha 8 días hábiles posterior a la de retorno.
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

// TranslateMonth retorna el mapping ES del month name.
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

// GetMonthNames retorna un slice con los nombres de meses indexados en 1.
func GetMonthNames() []string {
	return []string{"", "Enero", "Febrero", "Marzo", "Abril", "Mayo", "Junio", "Julio", "Agosto", "Septiembre", "Octubre", "Noviembre", "Diciembre"}
}

// WeekRange define un intervalo de tiempo semanal.
type WeekRange struct {
	Inicio time.Time
	Fin    time.Time
}

// GetWeeksInMonth retorna los rangos Lunes-Domingo del mes especificado.
func GetWeeksInMonth(year int, month time.Month) []WeekRange {
	t := time.Date(year, month, 1, 0, 0, 0, 0, time.Local)
	lastDay := t.AddDate(0, 1, -1)

	var weeks []WeekRange
	for d := t; !d.After(lastDay); d = d.AddDate(0, 0, 1) {
		if d.Weekday() == time.Monday {
			monday := d
			sunday := d.AddDate(0, 0, 6)
			weeks = append(weeks, WeekRange{Inicio: monday, Fin: sunday})
		}
	}
	return weeks
}

// GetWeekDays retorna una lista de mapas con información de días entre dos fechas.
func GetWeekDays(desde, hasta *time.Time) []map[string]string {
	var weekDays []map[string]string
	dayNames := map[string]string{
		"Monday":    "Lun",
		"Tuesday":   "Mar",
		"Wednesday": "Mie",
		"Thursday":  "Jue",
		"Friday":    "Vie",
		"Saturday":  "Sab",
		"Sunday":    "Dom",
	}

	if desde != nil && hasta != nil {
		for d := *desde; !d.After(*hasta); d = d.AddDate(0, 0, 1) {
			esName := dayNames[d.Weekday().String()]
			weekDays = append(weekDays, map[string]string{
				"date":   d.Format("2006-01-02"),
				"name":   esName,
				"dayNum": d.Format("02"),
			})
		}
	}
	return weekDays
}
