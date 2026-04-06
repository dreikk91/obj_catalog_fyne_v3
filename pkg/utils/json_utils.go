package utils

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"
)

// ParseAnyInt намагається перетворити будь-який тип (json.Number, float64, string, int) на ціле число.
// Зручно для парсингу нетипізованого/динамічного JSON.
func ParseAnyInt(value any) int {
	switch v := value.(type) {
	case nil:
		return 0
	case int:
		return v
	case int64:
		return int(v)
	case int32:
		return int(v)
	case float64:
		return int(v)
	case float32:
		return int(v)
	case json.Number:
		if i, err := v.Int64(); err == nil {
			return int(i)
		}
		if f, err := v.Float64(); err == nil {
			return int(f)
		}
		return 0
	case string:
		text := strings.TrimSpace(v)
		if text == "" {
			return 0
		}
		if i, err := strconv.Atoi(text); err == nil {
			return i
		}
		if f, err := strconv.ParseFloat(text, 64); err == nil {
			return int(f)
		}
		return 0
	default:
		text := strings.TrimSpace(fmt.Sprintf("%v", value))
		if text == "" {
			return 0
		}
		if i, err := strconv.Atoi(text); err == nil {
			return i
		}
		if f, err := strconv.ParseFloat(text, 64); err == nil {
			return int(f)
		}
		return 0
	}
}

// ParseAnyTime намагається перетворити будь-який тип (int64 epoch, string ISO, time.Time) на час.
func ParseAnyTime(value any) time.Time {
	parseEpoch := func(epoch int64) time.Time {
		if epoch == 0 {
			return time.Time{}
		}
		// Мілісекунди (ms) - дуже велике число
		if epoch > 1_000_000_000_000 || epoch < -1_000_000_000_000 {
			return time.UnixMilli(epoch).Local()
		}
		// Секунди (s)
		if epoch > 1_000_000_000 || epoch < -1_000_000_000 {
			return time.Unix(epoch, 0).Local()
		}
		return time.Time{}
	}

	switch v := value.(type) {
	case nil:
		return time.Time{}
	case time.Time:
		return v.Local()
	case int64:
		return parseEpoch(v)
	case int:
		return parseEpoch(int64(v))
	case float64:
		return parseEpoch(int64(v))
	case float32:
		return parseEpoch(int64(v))
	case json.Number:
		if i, err := v.Int64(); err == nil {
			return parseEpoch(i)
		}
		if f, err := v.Float64(); err == nil {
			return parseEpoch(int64(f))
		}
		return time.Time{}
	case string:
		text := strings.TrimSpace(v)
		if text == "" {
			return time.Time{}
		}
		// Спроба ISO 8601 / RFC 3339
		if t, err := time.Parse(time.RFC3339, text); err == nil {
			return t.Local()
		}
		// Спроба чистого числа (epoch)
		if i, err := strconv.ParseInt(text, 10, 64); err == nil {
			return parseEpoch(i)
		}
		return time.Time{}
	default:
		return time.Time{}
	}
}

// BoolFromAny намагається перетворити будь-який тип на булеве значення.
func BoolFromAny(value any) (bool, bool) {
	switch typed := value.(type) {
	case bool:
		return typed, true
	case int:
		return typed > 0, true
	case int64:
		return typed > 0, true
	case float64:
		return typed > 0, true
	case string:
		raw := strings.TrimSpace(strings.ToLower(typed))
		switch raw {
		case "1", "true", "on", "armed", "guard", "group_on", "взято":
			return true, true
		case "0", "false", "off", "disarmed", "not_guard", "group_off", "знято":
			return false, true
		default:
			return false, false
		}
	default:
		return false, false
	}
}

// AsString перетворює будь-який базовий тип на рядок.
func AsString(value any) string {
	switch v := value.(type) {
	case nil:
		return ""
	case string:
		return strings.TrimSpace(v)
	case fmt.Stringer:
		return strings.TrimSpace(v.String())
	case int:
		return strconv.Itoa(v)
	case int64:
		return strconv.FormatInt(v, 10)
	case float64:
		return strconv.FormatInt(int64(v), 10)
	default:
		return strings.TrimSpace(fmt.Sprintf("%v", value))
	}
}
