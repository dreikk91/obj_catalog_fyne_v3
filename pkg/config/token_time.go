package config

import (
	"strings"
	"time"
)

func parseTokenExpiry(raw string) time.Time {
	value := strings.TrimSpace(raw)
	if value == "" {
		return time.Time{}
	}

	parsed, err := time.Parse(time.RFC3339, value)
	if err != nil {
		return time.Time{}
	}
	return parsed
}

func tokenUsableAt(accessToken string, tokenExpiry string, now time.Time) bool {
	if strings.TrimSpace(accessToken) == "" {
		return false
	}

	expiry := parseTokenExpiry(tokenExpiry)
	if expiry.IsZero() {
		return true
	}
	return expiry.After(now)
}
