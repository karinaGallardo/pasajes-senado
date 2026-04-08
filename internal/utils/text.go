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
	words := strings.Fields(s)
	for i, word := range words {
		lower := strings.ToLower(word)
		if isRoman(lower) {
			words[i] = strings.ToUpper(lower)
		} else {
			words[i] = caser.String(lower)
		}
	}
	return strings.Join(words, " ")
}

func isRoman(w string) bool {
	w = strings.ToUpper(w)
	// Números romanos comunes en cargos y niveles (I al XX)
	romanos := map[string]bool{
		"I": true, "II": true, "III": true, "IV": true, "V": true,
		"VI": true, "VII": true, "VIII": true, "IX": true, "X": true,
		"XI": true, "XII": true, "XIII": true, "XIV": true, "XV": true,
		"XVI": true, "XVII": true, "XVIII": true, "XIX": true, "XX": true,
	}
	return romanos[w]
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

// TruncateString recorta un string a la longitud máxima (si aplica).
func TruncateString(s string, max int) string {
	if len(s) <= max {
		return s
	}
	if max <= 3 {
		return s[:max]
	}
	return s[:max]
}

// UniqueStringsJoin elimina duplicados de un slice de strings y los une con un separador.
func UniqueStringsJoin(elements []string, sep string) string {
	if len(elements) == 0 {
		return ""
	}

	uniqueMap := make(map[string]bool)
	var uniqueElements []string

	for _, e := range elements {
		e = strings.TrimSpace(e)
		if e != "" && !uniqueMap[e] {
			uniqueMap[e] = true
			uniqueElements = append(uniqueElements, e)
		}
	}

	return strings.Join(uniqueElements, sep)
}
