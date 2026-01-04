package utils

import (
	"strings"

	gonanoid "github.com/matoous/go-nanoid/v2"
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

func GenerateCode(length int) (string, error) {
	const alphabet = "ABCDEFGHJKLMNPQRSTUVWXYZ23456789"
	return gonanoid.Generate(alphabet, length)
}
