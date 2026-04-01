package data

import (
	"context"
	"sort"
	"strings"
	"time"

	"github.com/rs/zerolog/log"
)

func (p *CASLCloudProvider) resolveCASLDeviceTypeLabel(ctx context.Context, rawType string) string {
	rawType = strings.TrimSpace(rawType)
	if rawType == "" {
		return "—"
	}

	if value := p.lookupCASLDeviceTypeInDictionary(ctx, rawType); value != "" {
		return value
	}

	return decodeCASLDeviceType(rawType)
}

func (p *CASLCloudProvider) lookupCASLDeviceTypeInDictionary(ctx context.Context, rawType string) string {
	dict, ok := p.cachedDictionarySnapshot(ctx)
	if !ok || len(dict) == 0 {
		return ""
	}
	

	deviceTypes := extractCASLDeviceTypesMap(dict)
	if len(deviceTypes) == 0 {
		return ""
	}

	for _, key := range []string{rawType, strings.ToUpper(rawType), strings.ToLower(rawType)} {
		if text := strings.TrimSpace(deviceTypes[key]); text != "" {
			return text
		}
	}

	return ""
}

func (p *CASLCloudProvider) cachedDictionarySnapshot(ctx context.Context) (map[string]any, bool) {
	p.mu.RLock()
	cacheValid := len(p.cachedDictionary) > 0 && time.Since(p.cachedDictionaryAt) <= caslDictionaryTTL
	if cacheValid {
		snapshot := copyStringAnyMap(p.cachedDictionary)
		p.mu.RUnlock()
		return snapshot, true
	}
	p.mu.RUnlock()

	_ = p.loadDictionaryMap(ctx)

	p.mu.RLock()
	defer p.mu.RUnlock()
	if len(p.cachedDictionary) == 0 {
		return nil, false
	}
	return copyStringAnyMap(p.cachedDictionary), true
}

func extractCASLDeviceTypesMap(value any) map[string]string {
	root, ok := value.(map[string]any)
	if !ok || len(root) == 0 {
		return nil
	}

	// Старий формат: user_device_types — це map[string]string
	if raw, exists := root["user_device_types"]; exists {
		if mapped := flattenStringMap(raw); len(mapped) > 0 {
			return mapped
		}
	}

	if nestedRaw, exists := root["dictionary"]; exists {
		if nested, ok := nestedRaw.(map[string]any); ok {
			if raw, exists := nested["user_device_types"]; exists {
				if mapped := flattenStringMap(raw); len(mapped) > 0 {
					return mapped
				}
			}
		}
	}

	// Новий формат: user_device_types — це []string (ключі),
	// переклади лежать у translate.uk (або dictionary.translate.uk)
	ukMap := extractCASLDictionaryLanguageMap(root, "uk")
	if len(ukMap) == 0 {
		return nil
	}

	// Якщо є явний список ключів — повертаємо лише ті, що є в ньому
	if raw, exists := root["user_device_types"]; exists {
		if arr, ok := raw.([]any); ok && len(arr) > 0 {
			result := make(map[string]string, len(arr))
			for _, item := range arr {
				key := strings.TrimSpace(asString(item))
				if key == "" {
					continue
				}
				for _, candidate := range []string{key, strings.ToUpper(key), strings.ToLower(key)} {
					if v, found := ukMap[candidate]; found && v != "" {
						result[key] = v
						break
					}
				}
			}
			if len(result) > 0 {
				return result
			}
		}
	}

	// Немає явного списку — повертаємо весь ukMap як є
	return ukMap
}


func (p *CASLCloudProvider) loadDictionaryMap(ctx context.Context) map[string]string {
	p.mu.RLock()
	if len(p.cachedDictionary) > 0 && time.Since(p.cachedDictionaryAt) <= caslDictionaryTTL {
		cached := flattenLocalizedDictionaryMap(p.cachedDictionary)
		p.mu.RUnlock()
		return cached
	}
	p.mu.RUnlock()

	dict, err := p.ReadDictionary(ctx)
	if err != nil || len(dict) == 0 {
		if err != nil {
			log.Debug().Err(err).Msg("CASL: не вдалося прочитати dictionary для розшифровки подій")
		}
		return nil
	}

	p.mu.Lock()
	p.cachedDictionary = copyStringAnyMap(dict)
	p.cachedDictionaryAt = time.Now()
	p.mu.Unlock()

	go p.preloadTranslatorsFromDict(context.Background(), dict)

	return flattenLocalizedDictionaryMap(dict)
}

// preloadTranslatorsFromDict читає список user_device_types зі словника
// та послідовно завантажує транслятор (get_msg_translator_by_device_type) для кожного типу.
// Викликається один раз після кешування словника.
func (p *CASLCloudProvider) preloadTranslatorsFromDict(ctx context.Context, dict map[string]any) {
	deviceTypes := extractCASLUserDeviceTypes(dict)
	if len(deviceTypes) == 0 {
		return
	}
	log.Debug().
		Strs("device_types", deviceTypes).
		Msg("CASL: попереднє завантаження трансляторів для типів пристроїв")

	for _, dt := range deviceTypes {
		select {
		case <-ctx.Done():
			return
		default:
		}
		p.loadTranslatorMap(ctx, dt)
	}
}

// extractCASLUserDeviceTypes витягує всі типи приладів зі словника.
// Читає два поля:
//   - user_device_types: ["AX_PRO","Ajax Pro","MAKS_PRO","SATEL",...]
//   - devices: [{"type":"TYPE_DEVICE_Ajax",...}, {"type":"TYPE_DEVICE_Dunay_4_3",...}, ...]
//
// Результат — об'єднаний дедупльований список.
func extractCASLUserDeviceTypes(dict map[string]any) []string {
	if len(dict) == 0 {
		return nil
	}

	seen := make(map[string]struct{})
	types := make([]string, 0, 32)

	addType := func(s string) {
		s = strings.TrimSpace(s)
		if s == "" {
			return
		}
		if _, exists := seen[s]; exists {
			return
		}
		seen[s] = struct{}{}
		types = append(types, s)
	}

	if raw, ok := dict["user_device_types"]; ok {
		if arr, ok := raw.([]any); ok {
			for _, item := range arr {
				addType(asString(item))
			}
		}
	}

	if raw, ok := dict["devices"]; ok {
		if arr, ok := raw.([]any); ok {
			for _, item := range arr {
				if obj, ok := item.(map[string]any); ok {
					addType(asString(obj["type"]))
				}
			}
		}
	}

	return types
}

func flattenLocalizedDictionaryMap(dict map[string]any) map[string]string {
	base := flattenStringMap(dict)
	uk := extractCASLDictionaryLanguageMap(dict, "uk")
	for key, value := range uk {
		if strings.TrimSpace(key) == "" || strings.TrimSpace(value) == "" {
			continue
		}
		base[key] = value
	}
	return base
}

func extractCASLDictionaryLanguageMap(dict map[string]any, lang string) map[string]string {
	lang = strings.ToLower(strings.TrimSpace(lang))
	if len(dict) == 0 || lang == "" {
		return nil
	}

	langCandidates := []string{lang, strings.ToUpper(lang), "uk-UA", "uk_ua", "ua", "UA"}
	resolveLangMap := func(node any) map[string]string {
		root, ok := node.(map[string]any)
		if !ok || len(root) == 0 {
			return nil
		}
		for _, key := range langCandidates {
			if nested, exists := root[key]; exists {
				flat := flattenStringMap(nested)
				if len(flat) > 0 {
					return flat
				}
			}
		}
		return nil
	}

	if nested, ok := dict["translate"]; ok {
		if out := resolveLangMap(nested); len(out) > 0 {
			return out
		}
	}
	if nestedRaw, ok := dict["dictionary"]; ok {
		if nested, okNested := nestedRaw.(map[string]any); okNested {
			if tr, exists := nested["translate"]; exists {
				if out := resolveLangMap(tr); len(out) > 0 {
					return out
				}
			}
		}
	}
	if out := resolveLangMap(dict); len(out) > 0 {
		return out
	}
	return nil
}

func (p *CASLCloudProvider) loadTranslatorMap(ctx context.Context, deviceType string) map[string]string {
	key := strings.TrimSpace(deviceType)
	if key == "" {
		return nil
	}

	p.mu.RLock()
	if !p.translatorDisabledUntil.IsZero() && time.Now().Before(p.translatorDisabledUntil) {
		p.mu.RUnlock()
		return nil
	}
	if cached, ok := p.cachedTranslators[key]; ok && time.Since(p.cachedTransAt[key]) <= caslTranslatorTTL {
		if len(cached) == 0 {
			p.mu.RUnlock()
			return nil
		}
		out := make(map[string]string, len(cached))
		for k, v := range cached {
			out[k] = v
		}
		p.mu.RUnlock()
		return out
	}
	p.mu.RUnlock()

	rawTranslator, err := p.GetMessageTranslatorByDeviceType(ctx, key)
	if err != nil {
		if isCASLWrongFormatErr(err) {

			rawAll, retryErr := p.GetMessageTranslatorByDeviceType(ctx, "")
			if retryErr == nil {
				flat := extractCASLTranslatorByType(rawAll, key)
				if len(flat) > 0 {
					p.mu.Lock()
					p.cachedTranslators[key] = flat
					p.cachedTransAt[key] = time.Now()
					p.mu.Unlock()

					out := make(map[string]string, len(flat))
					for k, v := range flat {
						out[k] = v
					}
					return out
				}
			}
		}

		p.mu.Lock()
		p.cachedTranslators[key] = map[string]string{}
		p.cachedTransAt[key] = time.Now()
		shouldLog := true
		if isCASLWrongFormatErr(err) {
			alreadyDisabled := !p.translatorDisabledUntil.IsZero() && time.Now().Before(p.translatorDisabledUntil)
			p.translatorDisabledUntil = time.Now().Add(caslTranslatorTTL)
			shouldLog = !alreadyDisabled
		}
		p.mu.Unlock()
		if shouldLog {
			log.Debug().Err(err).Str("deviceType", key).Msg("CASL: не вдалося отримати translator для типу пристрою")
		}
		return nil
	}

	flat := flattenCASLTranslatorMap(rawTranslator)
	if len(flat) == 0 {
		p.mu.Lock()
		p.cachedTranslators[key] = map[string]string{}
		p.cachedTransAt[key] = time.Now()
		p.mu.Unlock()
		return nil
	}

	p.mu.Lock()
	p.cachedTranslators[key] = flat
	p.cachedTransAt[key] = time.Now()
	p.mu.Unlock()

	out := make(map[string]string, len(flat))
	for k, v := range flat {
		out[k] = v
	}
	return out
}

func flattenStringMap(value any) map[string]string {
	result := make(map[string]string)

	var walk func(v any)
	walk = func(v any) {
		switch typed := v.(type) {
		case map[string]any:
			keys := make([]string, 0, len(typed))
			for k := range typed {
				keys = append(keys, k)
			}
			sort.Strings(keys)
			for _, k := range keys {
				nested := typed[k]
				switch n := nested.(type) {
				case string:
					text := strings.TrimSpace(n)
					if text != "" {
						key := strings.TrimSpace(k)
						if key != "" {
							result[key] = text
						}
					}
				default:
					walk(n)
				}
			}
		case []any:
			for _, nested := range typed {
				walk(nested)
			}
		}
	}

	walk(value)
	return result
}

func extractCASLTranslatorByType(raw any, deviceType string) map[string]string {
	flat := flattenCASLTranslatorMap(raw)
	if len(flat) == 0 {
		return nil
	}

	root, ok := raw.(map[string]any)
	if !ok {
		return flat
	}

	key := strings.TrimSpace(deviceType)
	if key == "" {
		return flat
	}

	candidates := []string{key, strings.ToUpper(key), strings.ToLower(key)}
	for _, candidate := range candidates {
		if nested, exists := root[candidate]; exists {
			if mapped := flattenCASLTranslatorMap(nested); len(mapped) > 0 {
				return mapped
			}
		}
	}

	return flat
}

func flattenCASLTranslatorMap(value any) map[string]string {
	result := make(map[string]string)

	setIfEmpty := func(key string, text string) {
		key = strings.TrimSpace(key)
		text = strings.TrimSpace(text)
		if key == "" || text == "" {
			return
		}
		if _, exists := result[key]; exists {
			return
		}
		result[key] = text
	}

	var walk func(v any)
	walk = func(v any) {
		switch typed := v.(type) {
		case map[string]any:
			if codes := extractCASLTranslatorCodes(typed); len(codes) > 0 {
				if text := extractCASLTranslatorText(typed); text != "" {
					for _, code := range codes {
						setIfEmpty(code, text)
					}
				}
			}

			for key, nested := range typed {
				candidate := strings.TrimSpace(key)
				if candidate == "" {
					continue
				}
				if looksLikeCASLTranslatorCode(candidate) {
					if text := extractCASLTranslatorText(nested); text != "" {
						setIfEmpty(candidate, text)
					}
				}
				walk(nested)
			}
		case []any:
			for _, nested := range typed {
				walk(nested)
			}
		}
	}

	walk(value)
	return result
}

func extractCASLTranslatorCode(entry map[string]any) string {
	codes := extractCASLTranslatorCodes(entry)
	if len(codes) > 0 {
		return codes[0]
	}
	return ""
}

func extractCASLTranslatorCodes(entry map[string]any) []string {
	base := ""
	for _, key := range []string{"contact_id", "contactId", "code", "event_code", "eventCode", "id", "key"} {
		if raw, ok := entry[key]; ok {
			code := strings.TrimSpace(asString(raw))
			if looksLikeCASLTranslatorCode(code) {
				base = code
				break
			}
		}
	}
	if base == "" {
		return nil
	}

	keys := []string{base}
	if isCASLTranslatorNumericCode(base) {
		if eventType := extractCASLTranslatorTypeEvent(entry); eventType != "" {
			keys = append([]string{eventType + base}, keys...)
		}
	}

	seen := make(map[string]struct{}, len(keys))
	out := make([]string, 0, len(keys))
	for _, key := range keys {
		key = strings.TrimSpace(key)
		if key == "" {
			continue
		}
		if _, exists := seen[key]; exists {
			continue
		}
		seen[key] = struct{}{}
		out = append(out, key)
	}
	return out
}

func extractCASLTranslatorTypeEvent(entry map[string]any) string {
	for _, key := range []string{"typeEvent", "type_event", "eventType", "event_type"} {
		raw, ok := entry[key]
		if !ok {
			continue
		}
		value := strings.ToUpper(strings.TrimSpace(asString(raw)))
		if value == "" {
			continue
		}
		if len(value) == 1 && (value == "E" || value == "R" || value == "P") {
			return value
		}
		if len(value) > 1 && (value[0] == 'E' || value[0] == 'R' || value[0] == 'P') {
			isNumericTail := true
			for _, ch := range value[1:] {
				if ch < '0' || ch > '9' {
					isNumericTail = false
					break
				}
			}
			if isNumericTail {
				return string(value[0])
			}
		}
	}
	return ""
}

func isCASLTranslatorNumericCode(value string) bool {
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

func extractCASLTranslatorText(value any) string {
	switch typed := value.(type) {
	case string:
		return strings.TrimSpace(typed)
	case map[string]any:
		priority := []string{
			"msg", "message", "text", "description", "title", "template",
			"name", "label", "value", "lang_uk", "uk", "lang_ru", "ru", "lang_en", "en",
		}
		for _, key := range priority {
			if raw, ok := typed[key]; ok {
				if text := extractCASLTranslatorText(raw); text != "" {
					return text
				}
			}
		}

		for _, key := range []string{"lang", "langs", "translations", "translate"} {
			if raw, ok := typed[key]; ok {
				if text := extractCASLTranslatorText(raw); text != "" {
					return text
				}
			}
		}

		if len(typed) == 1 {
			for _, nested := range typed {
				if text := extractCASLTranslatorText(nested); text != "" {
					return text
				}
			}
		}
	}
	return ""
}

func looksLikeCASLTranslatorCode(key string) bool {
	key = strings.TrimSpace(key)
	if key == "" {
		return false
	}

	upper := strings.ToUpper(key)
	if strings.HasPrefix(upper, "TYPE_DEVICE_") {
		return false
	}

	switch upper {
	case "MSG", "MESSAGE", "TEXT", "DESCRIPTION", "TITLE", "TEMPLATE", "NAME", "LABEL", "VALUE",
		"ISALARM", "ALARMCOLOR", "PRIORITY", "CODE", "EVENT_CODE", "CONTACT_ID", "LANG_UK", "LANG_RU", "LANG_EN",
		"UK", "RU", "EN", "STATUS", "DATA", "DICTIONARY", "TYPE":
		return false
	}

	if len(upper) >= 4 && (upper[0] == 'E' || upper[0] == 'R') {
		allDigits := true
		for _, ch := range upper[1:] {
			if ch < '0' || ch > '9' {
				allDigits = false
				break
			}
		}
		if allDigits {
			return true
		}
	}

	allDigits := len(upper) >= 3
	for _, ch := range upper {
		if ch < '0' || ch > '9' {
			allDigits = false
			break
		}
	}
	if allDigits {
		return true
	}

	if strings.ContainsRune(upper, '_') {
		hasLetter := false
		for _, ch := range upper {
			if ch >= 'A' && ch <= 'Z' {
				hasLetter = true
				break
			}
		}
		return hasLetter
	}

	return false
}

func decodeCASLEventDescription(translator map[string]string, dictionary map[string]string, code string, contactID string, number int, deviceType ...string) string {
	code = strings.TrimSpace(code)
	contactID = strings.TrimSpace(contactID)
	resolvedNumber := number

	template := resolveCASLTemplate(translator, code)
	if template != "" && !hasCyrillicChars(template) {

		if dictText := resolveCASLTemplate(dictionary, template); dictText != "" {
			template = dictText
		} else if fb := resolveCASLTemplate(caslMessageKeyFallbackTemplates, template); fb != "" {
			template = fb
		}
	}
	if template == "" {
		template = resolveCASLTemplate(dictionary, code)
	}
	fallbackTemplate := resolveCASLTemplate(caslMessageKeyFallbackTemplates, code)
	if fallbackTemplate != "" {

		template = fallbackTemplate
	}
	if template == "" {
		template = resolveCASLTemplate(caslContactIDFallbackTemplates, code)
	}
	if template != "" {
		return applyCASLNumberTemplate(template, resolvedNumber)
	}

	rawDeviceType := ""
	if len(deviceType) > 0 {
		rawDeviceType = strings.TrimSpace(deviceType[0])
	}
	if decoded, ok := decodeCASLProtocolCode(code, rawDeviceType); ok {
		if resolvedNumber <= 0 && decoded.HasNumber {
			resolvedNumber = decoded.Number
		}
		template = resolveCASLTemplate(translator, decoded.MessageKey)

		if template != "" && !hasCyrillicChars(template) {
			if dictText := resolveCASLTemplate(dictionary, template); dictText != "" {
				template = dictText
			} else if fb := resolveCASLTemplate(caslMessageKeyFallbackTemplates, template); fb != "" {
				template = fb
			}
		}
		if template == "" {
			template = resolveCASLTemplate(dictionary, decoded.MessageKey)
		}
		fallbackTemplate = resolveCASLTemplate(caslMessageKeyFallbackTemplates, decoded.MessageKey)
		if fallbackTemplate != "" {
			template = fallbackTemplate
		}
		if template == "" {
			template = resolveCASLTemplate(caslContactIDFallbackTemplates, decoded.MessageKey)
		}
		if template == "" {
			template = strings.TrimSpace(decoded.MessageKey)
		}
		if template != "" {
			return finalizeCASLDecodedTemplate(template, resolvedNumber, decoded.MessageKey)
		}
	}

	template = resolveCASLTemplate(translator, contactID)
	if template == "" {
		template = resolveCASLTemplate(dictionary, contactID)
	}
	if template == "" {
		template = resolveCASLTemplate(caslContactIDFallbackTemplates, contactID)
	}
	if template == "" {
		template = fallbackCASLContactIDTemplate(contactID)
	}
	if template == "" {
		return ""
	}
	return applyCASLNumberTemplate(template, resolvedNumber)
}

func buildCASLUserActionDetails(row CASLObjectEvent) string {
	action := strings.ToUpper(strings.TrimSpace(row.Action))
	if action == "" {
		action = strings.ToUpper(strings.TrimSpace(row.Code))
	}
	if action == "" {
		return ""
	}

	objectLabel := strings.TrimSpace(row.ObjName)
	if objectLabel == "" {
		if objID := strings.TrimSpace(row.ObjID); objID != "" {
			objectLabel = "Об'єкт #" + objID
		}
	}

	switch action {
	case "GRD_OBJ_NOTIF":
		parts := []string{"Нова тривога"}
		if objectLabel != "" {
			parts = append(parts, objectLabel)
		}
		return strings.Join(parts, " | ")
	case "GRD_OBJ_PICK":
		base := "Тривогу взято в роботу оператором"
		who := strings.TrimSpace(row.UserFIO)
		if who == "" {
			who = strings.TrimSpace(row.UserID)
		}
		if who != "" {
			return base + ": " + who
		}
		return base
	case "GRD_OBJ_ASS_MGR":
		base := "На тривогу призначено ГМР"
		mgrID := strings.TrimSpace(row.MgrID)
		if mgrID != "" {
			return base + " #" + mgrID
		}
		return base
	case "GRD_OBJ_MGR_CANCEL":
		base := "Тривогу скасовано оператором"
		mgrID := strings.TrimSpace(row.MgrID)
		if mgrID != "" {
			return base + " (ГМР #" + mgrID + ")"
		}
		return base
	case "GRD_OBJ_FINISH":
		base := "Обробку тривоги завершено оператором"
		who := strings.TrimSpace(row.UserFIO)
		if who == "" {
			who = strings.TrimSpace(row.UserID)
		}
		if who != "" {
			return base + ": " + who
		}
		return base
	case "DEVICE_BLOCK":
		return "Пристрій заблоковано"
	case "DEVICE_UNBLOCK":
		return "Пристрій розблоковано"
	default:
		return ""
	}
}

func buildCASLLineNameIndex(lines []caslDeviceLine) map[int]string {
	if len(lines) == 0 {
		return nil
	}

	index := make(map[int]string, len(lines))
	for _, line := range lines {
		name := strings.TrimSpace(line.Name.String())
		if name == "" {
			continue
		}
		if number := int(line.ID.Int64()); number > 0 {
			index[number] = name
			continue
		}
		if number := int(line.Number.Int64()); number > 0 {
			index[number] = name
		}
	}
	return index
}
