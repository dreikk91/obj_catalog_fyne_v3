package data

import (
	"context"
	"strconv"
	"strings"
	"time"
)

func (p *CASLCloudProvider) buildCASLLineInfoIndex(ctx context.Context, lines []caslDeviceLine) map[int]caslEventLineInfo {
	if len(lines) == 0 {
		return nil
	}

	index := make(map[int]caslEventLineInfo, len(lines))
	for _, line := range lines {
		number := int(line.Number.Int64())
		if number <= 0 {
			number = int(line.ID.Int64())
		}
		if number <= 0 {
			continue
		}

		lineType := strings.TrimSpace(line.LineType.String())
		index[number] = caslEventLineInfo{
			Description:   strings.TrimSpace(firstCASLValue(line.Description.String(), line.Name.String())),
			LineType:      lineType,
			LineTypeLabel: strings.TrimSpace(p.resolveCASLZoneTypeLabel(ctx, lineType)),
			AdapterType:   strings.TrimSpace(line.AdapterType.String()),
			AdapterNumber: int(line.AdapterNumber.Int64()),
		}
	}
	if len(index) == 0 {
		return nil
	}
	return index
}

func shouldLoadCASLEventUsers(rows []CASLObjectEvent) bool {
	for _, row := range rows {
		if strings.TrimSpace(row.HozUserID) != "" {
			return true
		}
	}
	return false
}

func buildCASLPPKEventDetails(
	row CASLObjectEvent,
	translator map[string]string,
	dictMap map[string]string,
	deviceType string,
	lineInfos map[int]caslEventLineInfo,
	users map[string]caslUser,
) string {
	number := int(row.Number)
	base := decodeCASLEventDescription(translator, dictMap, row.Code, row.ContactID, number, deviceType)
	messageKey := resolveCASLEventMessageKey(translator, row.Code, row.ContactID, deviceType)
	lineInfo, hasLine := findCASLEventLineInfo(lineInfos, number, messageKey)

	if messageKey != "" && hasLine {
		if adjusted := adjustCASLEventMessageKeyForLineType(messageKey, lineInfo.LineType); adjusted != messageKey {
			if text := decodeCASLEventDescription(translator, dictMap, adjusted, "", number, deviceType); strings.TrimSpace(text) != "" {
				base = text
				messageKey = adjusted
			}
		}
	}

	showLine := isCASLLineMessageKey(messageKey) || isCASLContactIDZoneEvent(row.ContactID)
	parts := make([]string, 0, 6)
	if text := strings.TrimSpace(base); text != "" {
		parts = append(parts, text)
	}
	if showLine && hasLine && strings.TrimSpace(lineInfo.Description) != "" {
		parts = append(parts, "Опис: "+strings.TrimSpace(lineInfo.Description))
	}
	if contactID := strings.TrimSpace(row.ContactID); contactID != "" {
		parts = append(parts, "("+contactID+")")
	}
	if userName := caslEventUserName(users, row.HozUserID); userName != "" {
		parts = append(parts, "Користувач: "+userName)
	}
	if showLine && hasLine && strings.TrimSpace(lineInfo.AdapterType) != "" {
		parts = append(parts, "Адаптер: "+strings.TrimSpace(lineInfo.AdapterType))
	}
	if showLine && hasLine && strings.TrimSpace(lineInfo.LineTypeLabel) != "" {
		parts = append(parts, "Тип: "+strings.TrimSpace(lineInfo.LineTypeLabel))
	}
	return strings.Join(parts, ", ")
}

func isCASLPPKMessageSource(sourceType string) bool {
	switch strings.ToLower(strings.TrimSpace(sourceType)) {
	case "", "ppk_event", "ppk_in", "ppk_service", "alarm":
		return true
	default:
		return false
	}
}

func resolveCASLEventMessageKey(translator map[string]string, code string, contactID string, deviceType string) string {
	code = strings.TrimSpace(code)
	contactID = strings.TrimSpace(contactID)

	if decoded, ok := decodeCASLProtocolCode(code, deviceType); ok {
		return strings.TrimSpace(decoded.MessageKey)
	}
	if key := symbolicCASLEventKey(code, translator); key != "" {
		return key
	}
	if key := symbolicCASLEventKey(contactID, translator); key != "" {
		return key
	}
	return ""
}

func resolveCASLEventClassificationKey(
	translator map[string]string,
	code string,
	contactID string,
	deviceType string,
	number int,
	lineInfos map[int]caslEventLineInfo,
) string {
	messageKey := resolveCASLEventMessageKey(translator, code, contactID, deviceType)
	if lineInfo, ok := findCASLEventLineInfo(lineInfos, number, messageKey); ok {
		if adjusted := adjustCASLEventMessageKeyForLineType(messageKey, lineInfo.LineType); adjusted != "" {
			messageKey = adjusted
		}
	}
	if messageKey != "" {
		return strings.TrimSpace(messageKey)
	}
	if code = strings.TrimSpace(code); code != "" {
		return code
	}
	return strings.TrimSpace(contactID)
}

func symbolicCASLEventKey(candidate string, translator map[string]string) string {
	candidate = strings.TrimSpace(candidate)
	if candidate == "" {
		return ""
	}
	if !isCASLNumericToken(candidate) {
		return candidate
	}
	if translated := strings.TrimSpace(resolveCASLTemplate(translator, candidate)); translated != "" && !hasCyrillicChars(translated) {
		return translated
	}
	return ""
}

func isCASLNumericToken(value string) bool {
	value = strings.TrimSpace(value)
	if value == "" {
		return false
	}
	for _, ch := range value {
		if ch < '0' || ch > '9' {
			return false
		}
	}
	return true
}

func adjustCASLEventMessageKeyForLineType(messageKey string, lineType string) string {
	messageKey = strings.ToUpper(strings.TrimSpace(messageKey))
	lineType = strings.ToUpper(strings.TrimSpace(lineType))
	switch lineType {
	case "ALM_BTN":
		switch messageKey {
		case "LINE_BRK":
			return "ATTACK"
		case "OO_LINE_BRK":
			return "OO_ATTACK"
		}
	case "ZONE_ALARM_ON_KZ":
		switch messageKey {
		case "LINE_BAD", "LINE_KZ":
			return "ATTACK"
		case "OO_LINE_BAD", "OO_LINE_KZ":
			return "OO_ATTACK"
		}
	}
	return messageKey
}

func findCASLEventLineInfo(lineInfos map[int]caslEventLineInfo, number int, messageKey string) (caslEventLineInfo, bool) {
	if len(lineInfos) == 0 || number <= 0 {
		return caslEventLineInfo{}, false
	}
	if info, ok := lineInfos[number]; ok {
		return info, true
	}
	if strings.Contains(strings.ToUpper(strings.TrimSpace(messageKey)), "AD_") {
		for _, info := range lineInfos {
			if info.AdapterNumber == number {
				return info, true
			}
		}
	}
	return caslEventLineInfo{}, false
}

func isCASLLineMessageKey(messageKey string) bool {
	key := strings.ToUpper(strings.TrimSpace(messageKey))
	switch {
	case key == "":
		return false
	case strings.Contains(key, "GROUP"),
		strings.Contains(key, "USER_ACCESS"),
		strings.Contains(key, "ID_HOZ"),
		strings.Contains(key, "PRIMUS"):
		return false
	case strings.Contains(key, "LINE"),
		strings.Contains(key, "ZONE"),
		strings.Contains(key, "ATTACK"),
		strings.Contains(key, "ALM_BTN"),
		strings.Contains(key, "AD_"):
		return true
	default:
		return false
	}
}

func isCASLContactIDZoneEvent(contactID string) bool {
	digits := caslDigitsOnly(contactID)
	if digits == "" {
		return false
	}
	value, err := strconv.Atoi(digits)
	if err != nil {
		return false
	}
	return value >= 100 && value <= 200
}

func caslDigitsOnly(raw string) string {
	var builder strings.Builder
	for _, ch := range raw {
		if ch >= '0' && ch <= '9' {
			builder.WriteRune(ch)
		}
	}
	return builder.String()
}

func caslEventUserName(users map[string]caslUser, userID string) string {
	if len(users) == 0 {
		return ""
	}
	userID = strings.TrimSpace(userID)
	if userID == "" {
		return ""
	}
	user, ok := users[userID]
	if !ok {
		return ""
	}
	return strings.TrimSpace(user.FullName())
}

func formatCASLActionCause(dictionary map[string]string, cause string) string {
	cause = strings.TrimSpace(cause)
	if cause == "" {
		return ""
	}
	if translated := strings.TrimSpace(resolveCASLTemplate(dictionary, cause)); translated != "" {
		return translated
	}
	return cause
}

func formatCASLActionUnblockTime(unblockUnix int64) string {
	if unblockUnix <= 0 {
		return ""
	}
	unblockTime := time.Unix(unblockUnix, 0).Local()
	maxBlockTime := time.Now().Add(24 * time.Hour)
	if unblockTime.After(maxBlockTime) {
		return "безстроково"
	}
	if sameCASLDate(unblockTime, time.Now()) {
		return "о " + unblockTime.Format("15:04:05")
	}
	return unblockTime.Format("02.01.2006 о 15:04:05")
}

func sameCASLDate(left time.Time, right time.Time) bool {
	ly, lm, ld := left.Date()
	ry, rm, rd := right.Date()
	return ly == ry && lm == rm && ld == rd
}
