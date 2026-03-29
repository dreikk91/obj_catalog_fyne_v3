package protocol

import (
	"obj_catalog_fyne_v3/pkg/models"
	"strconv"
	"strings"
)

var ContactIDFallbackTemplates = map[string]string{
	"E110":  "Пожежна тривога",
	"R110":  "Відновлення після пожежної тривоги",
	"E120":  "Тривожна кнопка",
	"R120":  "Відновлення після тривожної кнопки",
	"E130":  "Тривога проникнення",
	"R130":  "Відновлення після тривоги проникнення",
	"E301":  "Втрата живлення 220В",
	"R301":  "Відновлення живлення 220В",
	"E302":  "Низький заряд АКБ",
	"R302":  "Відновлення АКБ",
	"E390":  "Не прийшло опитування за вказаний час",
	"R390":  "Відновлення опитування",
	"R401":  "Зняття групи № {number}",
	"R402":  "Взяття групи № {number}",
	"E627":  "Старт процесу оновлення чи застосування нових налаштувань",
	"R627":  "Старт процесу оновлення чи застосування нових налаштувань",
	"E628":  "Завершення процесу оновлення чи застосування нових налаштувань",
	"R628":  "Завершення процесу оновлення чи застосування нових налаштувань",
	"61184": "Відповідь на опитування - норма шлейфа № {number}",
}

var MessageKeyFallbackTemplates = map[string]string{
	"GROUP_ON":        "Постановка групи {number}",
	"OO_GROUP_ON":     "Постановка групи {number}",
	"GROUP_OFF":       "Зняття групи № {number}",
	"OO_GROUP_OFF":    "Зняття групи № {number}",
	"LINE_BRK":        "Обрив шлейфа № {number}",
	"OO_LINE_BRK":     "Обрив шлейфа № {number}",
	"LINE_NORM":       "Норма шлейфа № {number}",
	"OO_LINE_NORM":    "Норма шлейфа № {number}",
	"LINE_KZ":         "Коротке замикання шлейфа № {number}",
	"OO_LINE_KZ":      "Коротке замикання шлейфа № {number}",
	"LINE_BAD":        "Несправність шлейфа № {number}",
	"OO_LINE_BAD":     "Несправність шлейфа № {number}",
	"ZONE_ALM":        "Тривога в зоні № {number}",
	"ZONE_NORM":       "Норма в зоні № {number}",
	"ALM_INNER_ZONE":  "Тривога внутрішньої зони № {number}",
	"NORM_INNER_ZONE": "Норма внутрішньої зони № {number}",
	"NORM_IO":         "Норма IO № {number}",
	"NO_220":          "Втрата живлення 220В",
	"OK_220":          "Відновлення живлення 220В",
	"PPK_NO_CONN":     "Немає зв'язку з ППК",
	"PPK_CONN_OK":     "Зв'язок з ППК відновлено",
	"ACC_BAD":         "Низький заряд АКБ",
	"ACC_OK":          "АКБ в нормі",
	"DOOR_OP":         "Відкриття корпусу/дверей",
	"DOOR_CL":         "Закриття корпусу/дверей",
	"CHECK_CONN":      "Перевірка зв'язку",
	"ENABLED":         "Прилад увімкнено",
	"DISABLED":        "Прилад вимкнено",
	"FULL_REBOOT":     "Повне перезавантаження ППК",
	"ID_HOZ":          "Ідентифікація користувача {number}",
	"PRIMUS":          "Ідентифікація користувача {number}",
	"UPD_START":       "Старт процесу оновлення чи застосування нових налаштувань",
	"UPD_END":         "Завершення процесу оновлення чи застосування нових налаштувань",
}

func DecodeEventDescription(translator map[string]string, dictionary map[string]string, code string, contactID string, number int, deviceType string) string {
	code = strings.TrimSpace(code)
	contactID = strings.TrimSpace(contactID)
	resolvedNumber := number

	template := resolveTemplate(translator, code)
	if template != "" && !hasCyrillicChars(template) {
		if dictText := resolveTemplate(dictionary, template); dictText != "" {
			template = dictText
		} else if fb := resolveTemplate(MessageKeyFallbackTemplates, template); fb != "" {
			template = fb
		}
	}
	if template == "" {
		template = resolveTemplate(dictionary, code)
	}
	if fb := resolveTemplate(MessageKeyFallbackTemplates, code); fb != "" {
		template = fb
	}
	if template == "" {
		template = resolveTemplate(ContactIDFallbackTemplates, code)
	}
	if template != "" {
		return applyNumberTemplate(template, resolvedNumber)
	}

	decoder := GetDecoder(deviceType)
	b1, b2, ok := parseCodeBytes(code)
	if ok {
		if decoded, ok := decoder.Decode(b1, b2); ok {
			if resolvedNumber <= 0 && decoded.HasNumber {
				resolvedNumber = decoded.Number
			}
			template = resolveTemplate(translator, decoded.MessageKey)
			if template != "" && !hasCyrillicChars(template) {
				if dictText := resolveTemplate(dictionary, template); dictText != "" {
					template = dictText
				} else if fb := resolveTemplate(MessageKeyFallbackTemplates, template); fb != "" {
					template = fb
				}
			}
			if template == "" {
				template = resolveTemplate(dictionary, decoded.MessageKey)
			}
			if fb := resolveTemplate(MessageKeyFallbackTemplates, decoded.MessageKey); fb != "" {
				template = fb
			}
			if template == "" {
				template = resolveTemplate(ContactIDFallbackTemplates, decoded.MessageKey)
			}
			if template == "" {
				template = decoded.MessageKey
			}
			return finalizeDecodedTemplate(template, resolvedNumber, decoded.MessageKey)
		}
	}

	template = resolveTemplate(translator, contactID)
	if template == "" {
		template = resolveTemplate(dictionary, contactID)
	}
	if template == "" {
		template = resolveTemplate(ContactIDFallbackTemplates, contactID)
	}
	if template == "" {
		template = fallbackContactIDTemplate(contactID)
	}
	if template == "" {
		return ""
	}
	return applyNumberTemplate(template, resolvedNumber)
}

func resolveTemplate(source map[string]string, key string) string {
	if len(source) == 0 {
		return ""
	}
	key = strings.TrimSpace(key)
	if key == "" {
		return ""
	}
	upper := strings.ToUpper(key)
	for _, candidate := range []string{key, upper, strings.ToLower(key)} {
		if value, ok := source[candidate]; ok && value != "" {
			return value
		}
	}
	return ""
}

func applyNumberTemplate(template string, number int) string {
	out := template
	sNum := strconv.Itoa(number)
	out = strings.ReplaceAll(out, "{number}", sNum)
	out = strings.ReplaceAll(out, "{zone}", sNum)
	out = strings.ReplaceAll(out, "{line}", sNum)
	out = strings.ReplaceAll(out, "%number%", sNum)
	return out
}

func hasCyrillicChars(text string) bool {
	for _, r := range text {
		if (r >= 'А' && r <= 'я') || r == 'Ї' || r == 'ї' || r == 'Є' || r == 'є' || r == 'І' || r == 'і' || r == 'Ґ' || r == 'ґ' {
			return true
		}
	}
	return false
}

func parseCodeBytes(code string) (byte, byte, bool) {
	val, err := strconv.ParseInt(code, 10, 64)
	if err != nil {
		return 0, 0, false
	}
	return byte((val >> 8) & 0xFF), byte(val & 0xFF), true
}

func finalizeDecodedTemplate(template string, number int, messageKey string) string {
	out := applyNumberTemplate(template, number)
	if number > 0 && needsNumberSuffix(messageKey) && !strings.Contains(template, "{") {
		out += " № " + strconv.Itoa(number)
	}
	return out
}

func needsNumberSuffix(key string) bool {
	upper := strings.ToUpper(key)
	return strings.Contains(upper, "GROUP") || strings.Contains(upper, "ZONE") || strings.Contains(upper, "LINE")
}

func fallbackContactIDTemplate(contactID string) string {
	if len(contactID) < 4 {
		return ""
	}
	if contactID[0] == 'R' {
		return "Відновлення ContactID " + contactID
	}
	if contactID[0] == 'E' {
		return "Тривога ContactID " + contactID
	}
	return ""
}

func ClassifyEventType(code string) models.EventType {
	value := strings.ToUpper(strings.TrimSpace(code))
	valueLower := strings.ToLower(strings.TrimSpace(code))

	switch {
	case strings.Contains(value, "FIRE"), strings.Contains(value, "SMOKE"), strings.Contains(value, "HEAT"), strings.Contains(valueLower, "пожеж"), strings.Contains(valueLower, "дим"), strings.Contains(valueLower, "тепл"):
		return models.EventFire
	case strings.Contains(value, "BURGLARY"), strings.Contains(value, "INTRUSION"), strings.Contains(value, "BRUTFORS"), strings.Contains(value, "ZONE_ALM"), strings.Contains(valueLower, "проник"), strings.Contains(valueLower, "злом"):
		return models.EventBurglary
	case strings.Contains(value, "PANIC"), strings.Contains(value, "COERCION"), strings.Contains(value, "ATTACK"), strings.Contains(valueLower, "панік"), strings.Contains(valueLower, "напад"):
		return models.EventPanic
	case strings.Contains(value, "MEDICAL"), strings.Contains(valueLower, "медич"):
		return models.EventMedical
	case strings.Contains(value, "GAS"), strings.Contains(valueLower, "газ"):
		return models.EventGas
	case strings.Contains(value, "SABOTAGE"), strings.Contains(value, "TAMPER"), strings.Contains(valueLower, "саботаж"), strings.Contains(valueLower, "тампер"):
		return models.EventTamper
	case strings.Contains(value, "ARMED"), strings.Contains(valueLower, "взят"), strings.Contains(value, "R402"):
		return models.EventArm
	case strings.Contains(value, "DISARM"), strings.Contains(valueLower, "знят"), strings.Contains(value, "R401"):
		return models.EventDisarm
	case strings.Contains(value, "NO_220"), strings.Contains(value, "POWER_FAIL"), strings.Contains(valueLower, "220") && strings.Contains(valueLower, "пропаж"):
		return models.EventPowerFail
	case strings.Contains(value, "OK_220"), strings.Contains(value, "RESTORE"), strings.Contains(value, "RECOVER"), strings.Contains(valueLower, "віднов"), strings.Contains(valueLower, "норма"):
		return models.EventRestore
	case strings.Contains(value, "OFFLINE"), strings.Contains(value, "CONNECTION_LOST"), strings.Contains(valueLower, "нема зв"):
		return models.EventOffline
	case strings.Contains(value, "BATTERY") && strings.Contains(value, "LOW"), strings.Contains(valueLower, "акб") && strings.Contains(valueLower, "розряд"):
		return models.EventBatteryLow
	case strings.Contains(value, "TEST"), strings.Contains(value, "PING"), strings.Contains(value, "POLL"):
		return models.EventTest
	case strings.Contains(value, "UPD_START"), strings.Contains(value, "FIRMWARE"), strings.Contains(value, "ID_HOZ"), strings.Contains(value, "USER_ACCESS"):
		return models.SystemEvent
	}
	return models.EventFault
}

func ClassifyEventTypeWithContext(code, contactID, sourceType, details string) models.EventType {
	normalizedType := strings.TrimSpace(sourceType)
	if normalizedType != "" && !strings.EqualFold(normalizedType, "user_action") && !strings.EqualFold(normalizedType, "mob_user_action") {
		if mapped := mapTapeEventType(normalizedType); mapped != models.EventFault || strings.EqualFold(normalizedType, "fault") {
			return mapped
		}
	}

	if et := ClassifyEventType(code); et != models.EventFault {
		return et
	}
	if et := ClassifyEventType(contactID); et != models.EventFault {
		return et
	}
	if et := ClassifyEventType(details); et != models.EventFault {
		return et
	}
	return models.EventFault
}

func mapTapeEventType(raw string) models.EventType {
	value := strings.ToLower(strings.TrimSpace(raw))
	switch value {
	case "fire": return models.EventFire
	case "burglary": return models.EventBurglary
	case "panic": return models.EventPanic
	case "medical": return models.EventMedical
	case "gas": return models.EventGas
	case "tamper": return models.EventTamper
	case "fault": return models.EventFault
	case "restore": return models.EventRestore
	case "arm": return models.EventArm
	case "disarm": return models.EventDisarm
	case "test", "poll": return models.EventTest
	case "power_fail": return models.EventPowerFail
	case "power_ok": return models.EventPowerOK
	case "batt_low": return models.EventBatteryLow
	case "offline": return models.EventOffline
	case "online": return models.EventOnline
	case "system", "user_action", "ppk_action", "ppk_service", "system_event", "system_action", "m3_in", "mob_user_action":
		return models.SystemEvent
	}
	return ClassifyEventType(value)
}

func MapEventSC1(eventType models.EventType) int {
	switch eventType {
	case models.EventFire: return 1
	case models.EventBurglary: return 22
	case models.EventPanic: return 21
	case models.EventMedical: return 23
	case models.EventGas: return 24
	case models.EventTamper: return 25
	case models.EventRestore, models.EventPowerOK: return 5
	case models.EventArm: return 10
	case models.EventDisarm: return 14
	case models.EventOffline: return 12
	case models.EventTest, models.SystemEvent: return 6
	default: return 2
	}
}
