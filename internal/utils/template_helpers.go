package utils

import (
	"encoding/json"
	"fmt"
	"html/template"
	"strings"
	"time"
)

// TemplateFuncs exporta el mapa de funciones para uso en plantillas HTML.
func TemplateFuncs() template.FuncMap {
	return template.FuncMap{
		"add":              Add,
		"sum":              Sum,
		"sub":              Sub,
		"mul":              Mul,
		"inc":              Inc,
		"fecha":            FormatDate,
		"fechaRango":       FormatDateRange,
		"fechaHora":        FormatDateTimeES,
		"deref":            DerefString,
		"safe":             UnsafeHTML,
		"json":             ToJSON,
		"currentYear":      CurrentYear,
		"nombreMes":        GetMonthName,
		"monthName":        GetMonthName,
		"rangoSemana":      FormatWeekRange,
		"nombreDiaCorto":   DayNameShort,
		"rangoSemanaCorto": FormatWeekRangeShort,
		"contains":         strings.Contains,
		"formatCurrency":   FormatCurrency,
	}
}

// FormatCurrency formatea un float64 a string con miles y 2 decimales.
func FormatCurrency(val interface{}) string {
	var f float64
	switch v := val.(type) {
	case float64:
		f = v
	case float32:
		f = float64(v)
	case int:
		f = float64(v)
	case int64:
		f = float64(v)
	default:
		return "0.00"
	}
	return fmt.Sprintf("%.2f", f)
}

func CurrentYear() int {
	return time.Now().Year()
}

// UnsafeHTML retorna el string como template.HTML para omitir el escape automático.
func UnsafeHTML(s string) template.HTML {
	return template.HTML(s)
}

// Helpers de aritmética.
func Add(a, b float64) float64 { return a + b }
func Sum(a, b int) int         { return a + b }
func Sub(a, b int) int         { return a - b }
func Mul(a, b int) int         { return a * b }
func Inc(i int) int            { return i + 1 }

// FormatDate formatea un *time.Time a "DD/MM/YYYY". Retorna "-" si es nil.
func FormatDate(t *time.Time) string {
	if t == nil {
		return "-"
	}
	return t.Format("02/01/2006")
}

// FormatDateTimeES formatea fechas a formato texto en español con AM/PM.
// Ejemplo: "dom, 08 feb 2026, 12:56 PM"
func FormatDateTimeES(val interface{}) string {
	var t time.Time
	switch v := val.(type) {
	case time.Time:
		t = v
	case *time.Time:
		if v == nil {
			return "-"
		}
		t = *v
	default:
		return "-"
	}

	day := DayNameShort(&t)
	month := strings.ToLower(GetMonthName(t.Month()))
	if len(month) > 3 {
		month = month[:3]
	}

	// Format: "dom, 08 feb 2026, 12:56 PM"
	// La hora en Go con PM/AM se hace con "03:04 PM"
	return fmt.Sprintf("%s, %s %s %d, %s",
		day,
		t.Format("02"),
		month,
		t.Year(),
		t.Format("03:04 PM"))
}

// DerefString desreferencia un string pointer de forma segura. Retorna "" si es nil.
func DerefString(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}

// FormatDateRange formatea dos punteros de tiempo como rango "DD/MM - DD/MM".
func FormatDateRange(ini, fin *time.Time) string {
	if ini == nil || fin == nil {
		return "-"
	}
	return fmt.Sprintf("%s - %s", ini.Format("02/01"), fin.Format("02/01"))
}

// ToJSON serializa un valor a string JSON.
func ToJSON(v any) string {
	b, _ := json.Marshal(v)
	return string(b)
}

// FormatWeekRange formatea un rango de fechas al estilo "(lun, 09 de feb al dom, 15 de feb)".
func FormatWeekRange(ini, fin *time.Time) string {
	if ini == nil || fin == nil {
		return ""
	}

	translateDay := func(d time.Weekday) string {
		switch d {
		case time.Monday:
			return "lun"
		case time.Tuesday:
			return "mar"
		case time.Wednesday:
			return "mie"
		case time.Thursday:
			return "jue"
		case time.Friday:
			return "vie"
		case time.Saturday:
			return "sab"
		case time.Sunday:
			return "dom"
		}
		return ""
	}

	translateMonth := func(m time.Month) string {
		switch m {
		case time.January:
			return "ene"
		case time.February:
			return "feb"
		case time.March:
			return "mar"
		case time.April:
			return "abr"
		case time.May:
			return "may"
		case time.June:
			return "jun"
		case time.July:
			return "jul"
		case time.August:
			return "ago"
		case time.September:
			return "sep"
		case time.October:
			return "oct"
		case time.November:
			return "nov"
		case time.December:
			return "dic"
		}
		return ""
	}

	return fmt.Sprintf("(%s, %d de %s al %s, %d de %s)",
		translateDay(ini.Weekday()), ini.Day(), translateMonth(ini.Month()),
		translateDay(fin.Weekday()), fin.Day(), translateMonth(fin.Month()))
}

// DayNameShort retorna el nombre corto del día (lun, mar, etc.)
func DayNameShort(t *time.Time) string {
	if t == nil {
		return ""
	}
	switch t.Weekday() {
	case time.Monday:
		return "lun"
	case time.Tuesday:
		return "mar"
	case time.Wednesday:
		return "mie"
	case time.Thursday:
		return "jue"
	case time.Friday:
		return "vie"
	case time.Saturday:
		return "sab"
	case time.Sunday:
		return "dom"
	}
	return ""
}

// FormatWeekRangeShort retorna un rango corto "( lun 3 - dom 9 )"
func FormatWeekRangeShort(ini, fin *time.Time) string {
	if ini == nil || fin == nil {
		return ""
	}
	return fmt.Sprintf("( %s %d - %s %d )",
		DayNameShort(ini), ini.Day(),
		DayNameShort(fin), fin.Day())
}

// FormatDateTimeLongES formatea una fecha a formato largo y legible.
// Ejemplo: "domingo, 08 de febrero de 2026 a las 12:56 PM"
func FormatDateTimeLongES(t time.Time) string {
	day := DayNameLong(&t)
	month := GetMonthName(t.Month())

	// Format: "domingo, 08 de febrero de 2026 a las 12:56 PM"
	return fmt.Sprintf("%s, %02d de %s de %d a las %s",
		day,
		t.Day(),
		strings.ToLower(month),
		t.Year(),
		t.Format("03:04 PM"))
}

// DayNameLong retorna el nombre completo del día en español.
func DayNameLong(t *time.Time) string {
	if t == nil {
		return ""
	}
	switch t.Weekday() {
	case time.Monday:
		return "lunes"
	case time.Tuesday:
		return "martes"
	case time.Wednesday:
		return "miércoles"
	case time.Thursday:
		return "jueves"
	case time.Friday:
		return "viernes"
	case time.Saturday:
		return "sábado"
	case time.Sunday:
		return "domingo"
	}
	return ""
}
