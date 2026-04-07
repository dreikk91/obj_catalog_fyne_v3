package simoperator

import (
	"strings"

	"obj_catalog_fyne_v3/pkg/utils"
)

type Operator string

const (
	Unknown  Operator = ""
	Vodafone Operator = "vodafone"
	Kyivstar Operator = "kyivstar"
	Lifecell Operator = "lifecell"
)

func Detect(raw string) Operator {
	switch {
	case IsVodafone(raw):
		return Vodafone
	case IsKyivstar(raw):
		return Kyivstar
	case IsLifecell(raw):
		return Lifecell
	default:
		return Unknown
	}
}

func IsVodafone(raw string) bool {
	digits := utils.DigitsOnly(raw)
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
	digits := utils.DigitsOnly(raw)
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

func IsLifecell(raw string) bool {
	digits := utils.DigitsOnly(raw)
	switch {
	case len(digits) >= 5 && strings.HasPrefix(digits, "38063"):
		return true
	case len(digits) >= 5 && strings.HasPrefix(digits, "38073"):
		return true
	case len(digits) >= 5 && strings.HasPrefix(digits, "38093"):
		return true
	case len(digits) >= 3 && strings.HasPrefix(digits, "063"):
		return true
	case len(digits) >= 3 && strings.HasPrefix(digits, "073"):
		return true
	case len(digits) >= 3 && strings.HasPrefix(digits, "093"):
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
	case Lifecell:
		return "lifecell"
	default:
		return "SIM API"
	}
}
