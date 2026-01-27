package utils

import (
	"fmt"
	"time"
)

// FormatDateShortES formatear una fecha en formato "mar. 03 feb. 2026"
func FormatDateShortES(t time.Time) string {
	dayNames := map[string]string{
		"Monday":    "lun.",
		"Tuesday":   "mar.",
		"Wednesday": "mié.",
		"Thursday":  "jue.",
		"Friday":    "vie.",
		"Saturday":  "sáb.",
		"Sunday":    "dom.",
	}

	monthNames := map[string]string{
		"January":   "ene.",
		"February":  "feb.",
		"March":     "mar.",
		"April":     "abr.",
		"May":       "may.",
		"June":      "jun.",
		"July":      "jul.",
		"August":    "ago.",
		"September": "sep.",
		"October":   "oct.",
		"November":  "nov.",
		"December":  "dic.",
	}

	day := dayNames[t.Weekday().String()]
	month := monthNames[t.Month().String()]
	dayNum := t.Format("02")
	year := t.Year()

	return fmt.Sprintf("%s %s %s %d", day, dayNum, month, year)
}
