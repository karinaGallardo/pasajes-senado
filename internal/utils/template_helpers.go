package utils

import (
	"fmt"
	"html/template"
	"time"
)

func TemplateFuncs() template.FuncMap {
	return template.FuncMap{
		"add": Add,
		"sum": Sum,
		"sub": Sub,
		"mul": Mul,
		"inc": Inc,
		"dt":  FormatDate,
		"df":  FormatDateRange,
		"dtl": FormatDateTimeES,
	}
}

func Add(a, b float64) float64 { return a + b }
func Sum(a, b int) int         { return a + b }
func Sub(a, b int) int         { return a - b }
func Mul(a, b int) int         { return a * b }
func Inc(i int) int            { return i + 1 }

func FormatDate(t *time.Time) string {
	if t == nil {
		return "-"
	}
	return t.Format("02/01/2006")
}

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
		if contains(str, en) {
			return replace(str, en, es)
		}
	}
	return str
}

func contains(s, substr string) bool {
	return (len(s) >= len(substr)) && (s[0:len(substr)] == substr || contains(s[1:], substr))
}

func replace(s, old, new string) string {
	res := ""
	for i := 0; i < len(s); i++ {
		if i+len(old) <= len(s) && s[i:i+len(old)] == old {
			res += new
			i += len(old) - 1
		} else {
			res += string(s[i])
		}
	}
	return res
}

func FormatDateRange(ini, fin *time.Time) string {
	if ini == nil || fin == nil {
		return "-"
	}
	return fmt.Sprintf("%s - %s", ini.Format("02/01"), fin.Format("02/01"))
}
