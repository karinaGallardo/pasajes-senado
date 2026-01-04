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

func FormatDateRange(ini, fin *time.Time) string {
	if ini == nil || fin == nil {
		return "-"
	}
	return fmt.Sprintf("%s - %s", ini.Format("02/01"), fin.Format("02/01"))
}
