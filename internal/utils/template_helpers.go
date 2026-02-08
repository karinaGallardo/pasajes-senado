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
		"add":         Add,
		"sum":         Sum,
		"sub":         Sub,
		"mul":         Mul,
		"inc":         Inc,
		"dt":          FormatDate,
		"df":          FormatDateRange,
		"dtl":         FormatDateTimeES,
		"deref":       DerefString,
		"safe":        UnsafeHTML,
		"json":        ToJSON,
		"currentYear": CurrentYear,
		"monthName":   GetMonthName,
		"weekRange":   FormatWeekRange,
	}
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

// FormatDateTimeES formatea fechas a formato texto en español.
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

	str := t.Format("02 Jan 2006, 15:04")
	meses := map[string]string{
		"Jan": "Ene", "Feb": "Feb", "Mar": "Mar", "Apr": "Abr",
		"May": "May", "Jun": "Jun", "Jul": "Jul", "Aug": "Ago",
		"Sep": "Sep", "Oct": "Oct", "Nov": "Nov", "Dec": "Dic",
	}
	for en, es := range meses {
		if strings.Contains(str, en) {
			return strings.ReplaceAll(str, en, es)
		}
	}
	return str
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
