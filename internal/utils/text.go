package utils

import (
	"strings"

	"golang.org/x/text/cases"
	"golang.org/x/text/language"
)

func CleanName(s string) string {
	s = strings.TrimSpace(s)
	if s == "" {
		return ""
	}
	caser := cases.Title(language.Spanish)
	return caser.String(strings.ToLower(s))
}

func CleanString(s string) string {
	return strings.TrimSpace(s)
}
