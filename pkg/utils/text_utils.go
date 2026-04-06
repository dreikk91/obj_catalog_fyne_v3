package utils

import (
	"strings"
	"unicode"
)

// DigitsOnly повертає рядок, що містить лише цифри з вхідного рядка.
func DigitsOnly(value string) string {
	if value == "" {
		return ""
	}
	var b strings.Builder
	b.Grow(len(value))
	for _, r := range value {
		if unicode.IsDigit(r) {
			b.WriteRune(r)
		}
	}
	return b.String()
}

// IsDigitsOnlyTerm перевіряє, чи складається рядок виключно з цифр.
func IsDigitsOnlyTerm(value string) bool {
	if value == "" {
		return false
	}
	for _, r := range value {
		if !unicode.IsDigit(r) {
			return false
		}
	}
	return true
}

// HasCyrillicChars перевіряє наявність кириличних символів у рядку.
func HasCyrillicChars(text string) bool {
	for _, r := range text {
		if (r >= 'А' && r <= 'я') || r == 'Ї' || r == 'ї' || r == 'Є' || r == 'є' || r == 'І' || r == 'і' || r == 'Ґ' || r == 'ґ' {
			return true
		}
	}
	return false
}
