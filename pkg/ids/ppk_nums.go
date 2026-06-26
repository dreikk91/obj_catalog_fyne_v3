package ids

import (
	"fmt"
	"strconv"
	"strings"
)

// ppk_num ranges on the ZMQ broker:
//
//	     1 – 99 999  : Bridge (МІСТ/Firebird) — device's own numeric number
//	100 000 – 199 999: Phoenix              — 100_000 + L-number (L00022 → 100022)
//
// CASL-native devices use small numbers below 1 000 and are not owned by the bridge.
const (
	BridgePPKNumMin = 1
	BridgePPKNumMax = 99_999

	PhoenixPPKNumOffset = 100_000
	PhoenixPPKNumMin    = 100_000
	PhoenixPPKNumMax    = 199_999
)

// IsBridgePPKNum reports whether num belongs to the Bridge (МІСТ/Firebird) range.
func IsBridgePPKNum(num int) bool {
	return num >= BridgePPKNumMin && num <= BridgePPKNumMax
}

// IsPhoenixPPKNum reports whether num belongs to the Phoenix range.
func IsPhoenixPPKNum(num int) bool {
	return num >= PhoenixPPKNumMin && num <= PhoenixPPKNumMax
}

// PhoenixPPKNum converts a Phoenix panel ID ("L00022") to a broker ppk_num (100022).
// Returns (0, false) for unrecognised or out-of-range values.
func PhoenixPPKNum(panelID string) (int, bool) {
	s := strings.TrimPrefix(strings.ToUpper(strings.TrimSpace(panelID)), "L")
	n, err := strconv.Atoi(s)
	if err != nil || n < 0 || PhoenixPPKNumOffset+n > PhoenixPPKNumMax {
		return 0, false
	}
	return PhoenixPPKNumOffset + n, true
}

// PhoenixPanelID converts a broker ppk_num (100022) back to a Phoenix panel ID ("L00022").
func PhoenixPanelID(ppkNum int) (string, bool) {
	if !IsPhoenixPPKNum(ppkNum) {
		return "", false
	}
	return fmt.Sprintf("L%05d", ppkNum-PhoenixPPKNumOffset), true
}

// BridgePPKNum parses a Bridge object number string ("10001") and validates it
// against the bridge range. Returns (0, false) if the value is out of range or
// not a valid integer.
func BridgePPKNum(objectNumber string) (int, bool) {
	n, err := strconv.Atoi(strings.TrimSpace(objectNumber))
	if err != nil || n < BridgePPKNumMin || n > BridgePPKNumMax {
		return 0, false
	}
	return n, true
}

// BridgeObjectNumber converts a bridge ppk_num back to its string object number.
func BridgeObjectNumber(ppkNum int) (string, bool) {
	if !IsBridgePPKNum(ppkNum) {
		return "", false
	}
	return strconv.Itoa(ppkNum), true
}
