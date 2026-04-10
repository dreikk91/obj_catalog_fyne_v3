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
		if unicode.Is(unicode.Cyrillic, r) {
			return true
		}
	}
	return false
}

// LeadingDigits повертає початкову послідовність цифр, ігноруючи початкові пробіли.
func LeadingDigits(value string) string {
	if value == "" {
		return ""
	}

	var b strings.Builder
	for _, r := range value {
		if unicode.IsDigit(r) {
			b.WriteRune(r)
			continue
		}
		if b.Len() > 0 {
			break
		}
		if !unicode.IsSpace(r) {
			return ""
		}
	}
	return b.String()
}

// StripCountSuffix обрізає службовий суфікс виду " (N)" у UI-фільтрах.
func StripCountSuffix(value string) string {
	value = strings.TrimSpace(value)
	if idx := strings.Index(value, " ("); idx >= 0 {
		return strings.TrimSpace(value[:idx])
	}
	return value
}

// TrimmedNonEmptyStrings повертає копію зрізу без порожніх після TrimSpace значень.
func TrimmedNonEmptyStrings(values []string) []string {
	result := make([]string, 0, len(values))
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" {
			continue
		}
		result = append(result, value)
	}
	return result
}

// JoinTrimmedNonEmpty об'єднує непорожні після TrimSpace частини через пробіл.
func JoinTrimmedNonEmpty(values ...string) string {
	return strings.Join(TrimmedNonEmptyStrings(values), " ")
}
