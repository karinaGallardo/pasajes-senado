package utils

import (
	"fmt"
	"strings"
	"time"

	gonanoid "github.com/matoous/go-nanoid/v2"
	"golang.org/x/text/cases"
	"golang.org/x/text/language"
)

// CleanName normaliza el string aplicando Trim y Title mapping (es).
func CleanName(s string) string {
	s = strings.TrimSpace(s)
	if s == "" {
		return ""
	}
	caser := cases.Title(language.Spanish)
	return caser.String(strings.ToLower(s))
}

// CleanString retorna el string con TrimSpace aplicado.
func CleanString(s string) string {
	return strings.TrimSpace(s)
}

// GenerateCode genera un NanoID aleatorio basado en un alfabeto de exclusión (O, 0, I, 1).
func GenerateCode(length int) (string, error) {
	const alphabet = "ABCDEFGHJKLMNPQRSTUVWXYZ23456789"
	return gonanoid.Generate(alphabet, length)
}

// GenerateYearlyCode genera un código con prefijo y el año actual.
func GenerateYearlyCode(prefix string, length int) string {
	const alphabet = "0123456789ABCDEFGHIJKLMNOPQRSTUVWXYZ"
	code, _ := gonanoid.Generate(alphabet, length)
	return fmt.Sprintf("%s-%d-%s", prefix, time.Now().Year(), code)
}

// GeneratePrefixedCode genera un código con prefijo y un NanoID aleatorio.
func GeneratePrefixedCode(prefix string, length int) string {
	const alphabet = "0123456789ABCDEFGHIJKLMNOPQRSTUVWXYZ"
	code, _ := gonanoid.Generate(alphabet, length)
	return prefix + code
}

// SplitRoute toma una ruta como "TJA - CBB - LPB" y devuelve ["TJA - CBB", "CBB - LPB"]
func SplitRoute(route string) []string {
	if route == "" {
		return []string{}
	}
	parts := strings.Split(route, " - ")
	if len(parts) < 2 {
		return []string{route}
	}
	var connections []string
	for i := 0; i < len(parts)-1; i++ {
		connections = append(connections, fmt.Sprintf("%s - %s", parts[i], parts[i+1]))
	}
	return connections
}
