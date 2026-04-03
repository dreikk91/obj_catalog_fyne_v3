package simoperator

import "strings"

type Operator string

const (
	Unknown  Operator = ""
	Vodafone Operator = "vodafone"
	Kyivstar Operator = "kyivstar"
)

func Detect(raw string) Operator {
	switch {
	case IsVodafone(raw):
		return Vodafone
	case IsKyivstar(raw):
		return Kyivstar
	default:
		return Unknown
	}
}

func IsVodafone(raw string) bool {
	digits := digitsOnly(raw)
	switch {
	case len(digits) >= 5 && strings.HasPrefix(digits, "38050"):
		return true
	case len(digits) >= 5 && strings.HasPrefix(digits, "38066"):
		return true
	case len(digits) >= 5 && strings.HasPrefix(digits, "38075"):
		return true
	case len(digits) >= 5 && strings.HasPrefix(digits, "38095"):
		return true
	case len(digits) >= 5 && strings.HasPrefix(digits, "38099"):
		return true
	case len(digits) >= 3 && strings.HasPrefix(digits, "050"):
		return true
	case len(digits) >= 3 && strings.HasPrefix(digits, "066"):
		return true
	case len(digits) >= 3 && strings.HasPrefix(digits, "075"):
		return true
	case len(digits) >= 3 && strings.HasPrefix(digits, "095"):
		return true
	case len(digits) >= 3 && strings.HasPrefix(digits, "099"):
		return true
	default:
		return false
	}
}

func IsKyivstar(raw string) bool {
	digits := digitsOnly(raw)
	switch {
	case len(digits) >= 5 && strings.HasPrefix(digits, "38067"):
		return true
	case len(digits) >= 5 && strings.HasPrefix(digits, "38068"):
		return true
	case len(digits) >= 5 && strings.HasPrefix(digits, "38096"):
		return true
	case len(digits) >= 5 && strings.HasPrefix(digits, "38097"):
		return true
	case len(digits) >= 5 && strings.HasPrefix(digits, "38098"):
		return true
	case len(digits) >= 5 && strings.HasPrefix(digits, "38077"):
		return true
	case len(digits) >= 3 && strings.HasPrefix(digits, "067"):
		return true
	case len(digits) >= 3 && strings.HasPrefix(digits, "068"):
		return true
	case len(digits) >= 3 && strings.HasPrefix(digits, "096"):
		return true
	case len(digits) >= 3 && strings.HasPrefix(digits, "097"):
		return true
	case len(digits) >= 3 && strings.HasPrefix(digits, "098"):
		return true
	case len(digits) >= 3 && strings.HasPrefix(digits, "077"):
		return true
	default:
		return false
	}
}

func Label(operator Operator) string {
	switch operator {
	case Vodafone:
		return "Vodafone"
	case Kyivstar:
		return "Kyivstar"
	default:
		return "SIM API"
	}
}

func digitsOnly(raw string) string {
	var b strings.Builder
	b.Grow(len(raw))
	for _, r := range raw {
		if r >= '0' && r <= '9' {
			b.WriteRune(r)
		}
	}
	return b.String()
}
