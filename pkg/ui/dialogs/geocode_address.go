package dialogs

import (
	"fmt"
	"regexp"
	"strings"
	"unicode"
)

func scoreGeocodeCandidate(target geocodeTarget, row geocodeCandidate) float64 {
	score := row.Importance * 10

	candidateCity := firstAddressValue(row.Address, "city", "town", "village", "hamlet", "municipality")
	candidateStreet := firstAddressValue(row.Address, "road", "pedestrian", "residential", "street")
	candidateHouse := firstAddressValue(row.Address, "house_number")

	if target.City != "" {
		score += similarityScore(target.City, candidateCity, 38, 18, -7)
	}
	if target.Street != "" {
		// Для вулиці перевіряємо також display_name, бо інколи road порожній.
		streetScore := similarityScore(target.Street, candidateStreet, 35, 16, -6)
		if streetScore < 0 {
			streetScore = similarityScore(target.Street, row.DisplayName, 22, 10, -3)
		}
		score += streetScore
	}
	if target.House != "" {
		score += houseMatchScore(target.House, candidateHouse, row.DisplayName)
	}

	if strings.EqualFold(strings.TrimSpace(row.Address["country_code"]), "ua") {
		score += 2
	}

	poiType := strings.ToLower(strings.TrimSpace(row.Type))
	poiClass := strings.ToLower(strings.TrimSpace(row.Class))
	if poiType == "house" || poiType == "building" || poiClass == "building" {
		score += 6
	}
	if poiClass == "boundary" {
		score -= 12
	}

	return score
}

func similarityScore(target string, candidate string, exact float64, partial float64, mismatch float64) float64 {
	t := normalizeGeoToken(target)
	c := normalizeGeoToken(candidate)
	if t == "" || c == "" {
		return 0
	}
	if t == c {
		return exact
	}
	if strings.Contains(c, t) || strings.Contains(t, c) {
		return partial
	}
	return mismatch
}

func houseMatchScore(targetHouse string, candidateHouse string, displayName string) float64 {
	t := normalizeHouseToken(targetHouse)
	if t == "" {
		return 0
	}
	c := normalizeHouseToken(candidateHouse)
	if c != "" {
		if c == t {
			return 36
		}
		if strings.HasPrefix(c, t) || strings.HasPrefix(t, c) {
			return 18
		}
		return -10
	}

	if strings.Contains(normalizeGeoToken(displayName), normalizeGeoToken(targetHouse)) {
		return 10
	}
	return -4
}

func normalizeGeoToken(v string) string {
	v = strings.ToLower(strings.TrimSpace(v))
	if v == "" {
		return ""
	}
	v = strings.NewReplacer("’", "'", "`", "'", "ʼ", "'", "ё", "е", "ї", "і").Replace(v)

	var b strings.Builder
	b.Grow(len(v))
	for _, r := range v {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			b.WriteRune(r)
		} else {
			b.WriteByte(' ')
		}
	}
	return strings.Join(strings.Fields(b.String()), " ")
}

func normalizeHouseToken(v string) string {
	v = strings.TrimSpace(v)
	if v == "" {
		return ""
	}
	re := regexp.MustCompile(`(?i)\d+[0-9\p{L}/-]*`)
	token := strings.ToLower(strings.TrimSpace(re.FindString(v)))
	token = strings.ReplaceAll(token, " ", "")
	token = strings.ReplaceAll(token, "/", "")
	token = strings.ReplaceAll(token, "-", "")
	return token
}

func firstAddressValue(address map[string]string, keys ...string) string {
	for _, key := range keys {
		if v := strings.TrimSpace(address[key]); v != "" {
			return v
		}
	}
	return ""
}

func buildGeocodeQueries(address string) []string {
	queries := make([]string, 0, 16)
	addQuery := func(v string) {
		v = normalizeAddressSpaces(v)
		if v == "" {
			return
		}
		for _, existing := range queries {
			if strings.EqualFold(existing, v) {
				return
			}
		}
		queries = append(queries, v)
	}

	raw := normalizeAddressSpaces(address)
	cleaned := normalizeAddressForGeocode(raw)
	addQuery(raw)
	addQuery(cleaned)

	expanded := expandAddressAbbreviations(raw)
	expandedClean := expandAddressAbbreviations(cleaned)
	addQuery(expanded)
	addQuery(expandedClean)
	addQuery(ensureCountrySuffix(raw))
	addQuery(ensureCountrySuffix(cleaned))
	addQuery(ensureCountrySuffix(expanded))
	addQuery(ensureCountrySuffix(expandedClean))

	city, street, house, ok := parseAddressComponents(cleaned)
	if ok {
		if house != "" {
			addQuery(fmt.Sprintf("вулиця %s %s, %s, Україна", street, house, city))
			addQuery(fmt.Sprintf("%s %s, %s, Україна", street, house, city))
			addQuery(fmt.Sprintf("%s, вулиця %s, %s, Україна", city, street, house))
		}
		addQuery(fmt.Sprintf("вулиця %s, %s, Україна", street, city))
		addQuery(fmt.Sprintf("%s, %s, Україна", street, city))
		addQuery(fmt.Sprintf("%s, вулиця %s, Україна", city, street))
	}

	if cityOnly, ok := parseCityOnly(cleaned); ok {
		addQuery(fmt.Sprintf("%s, Україна", cityOnly))
	} else if streetOnly, houseOnly, ok := parseStreetAndHouseOnly(cleaned); ok {
		const defaultCity = "Львів"
		if houseOnly != "" {
			addQuery(fmt.Sprintf("вулиця %s %s, %s, Україна", streetOnly, houseOnly, defaultCity))
			addQuery(fmt.Sprintf("%s %s, %s, Україна", streetOnly, houseOnly, defaultCity))
		}
		addQuery(fmt.Sprintf("вулиця %s, %s, Україна", streetOnly, defaultCity))
		addQuery(fmt.Sprintf("%s, %s, Україна", streetOnly, defaultCity))
	}

	return queries
}

func parseAddressComponents(address string) (string, string, string, bool) {
	raw := expandAddressAbbreviations(normalizeAddressSpaces(address))
	parts := strings.Split(raw, ",")
	clean := make([]string, 0, len(parts))
	for _, p := range parts {
		p = normalizeAddressSpaces(p)
		if p != "" {
			clean = append(clean, p)
		}
	}
	if len(clean) == 0 {
		return "", "", "", false
	}

	city := ""
	street := ""
	house := ""

	for _, p := range clean {
		if city == "" || street == "" {
			combinedCity, combinedStreet, combinedHouse, ok := splitCombinedLocalityStreetPart(p)
			if ok {
				if city == "" {
					city = combinedCity
				}
				if street == "" {
					street = combinedStreet
				}
				if house == "" && combinedHouse != "" {
					house = combinedHouse
				}
			}
		}

		if city == "" {
			if v, ok := extractCity(p); ok {
				city = v
				continue
			}
		}
		if street == "" {
			if v, ok := extractStreet(p); ok {
				street = v
			}
		}
		if house == "" {
			house = extractHouseNumber(p)
		}
	}

	if city == "" {
		for _, p := range clean {
			if extractHouseNumber(p) != "" {
				continue
			}
			if _, ok := extractStreet(p); ok {
				continue
			}
			if !isAdministrativePart(p) {
				city = normalizeAddressSpaces(p)
				break
			}
		}
	}
	if street == "" {
		for _, p := range clean {
			if extractHouseNumber(p) != "" {
				continue
			}
			if isAdministrativePart(p) {
				continue
			}
			street = normalizeStreetName(p)
			if street != "" {
				break
			}
		}
	}
	if house == "" {
		if h := extractHouseNumber(raw); h != "" {
			house = h
		}
	}

	if city == "" || street == "" {
		return "", "", "", false
	}
	return city, street, house, true
}

func parseCityOnly(address string) (string, bool) {
	raw := expandAddressAbbreviations(normalizeAddressSpaces(address))
	parts := strings.Split(raw, ",")
	for _, p := range parts {
		p = normalizeAddressSpaces(p)
		if p == "" {
			continue
		}
		if city, ok := extractCity(p); ok {
			return city, true
		}
	}
	return "", false
}

func parseStreetAndHouseOnly(address string) (string, string, bool) {
	raw := expandAddressAbbreviations(normalizeAddressSpaces(address))
	house := extractHouseNumber(raw)
	if house == "" {
		return "", "", false
	}
	street := normalizeStreetName(raw)
	if street == "" || isAdministrativePart(street) {
		return "", "", false
	}
	return street, house, true
}

func normalizeAddressSpaces(v string) string {
	v = strings.TrimSpace(v)
	if v == "" {
		return ""
	}
	return strings.Join(strings.Fields(v), " ")
}

func normalizeAddressForGeocode(v string) string {
	v = strings.TrimSpace(v)
	v = strings.Trim(v, "\"'`")
	v = normalizeAddressSpaces(v)
	if v == "" {
		return ""
	}

	v = strings.NewReplacer(
		"Львіська", "Львівська",
		"Червоноі", "Червоної",
		"Мечнікова", "Мечникова",
		"буд.", " ",
		"буд ", " ",
	).Replace(v)

	// Склеюємо випадки типу "Незалежност, і".
	letterComma := regexp.MustCompile(`([\p{L}]{2,})\s*,\s*([\p{L}])\b`)
	for i := 0; i < 3; i++ {
		nv := letterComma.ReplaceAllString(v, "$1$2")
		if nv == v {
			break
		}
		v = nv
	}

	// Вирізаємо дужки та службові примітки.
	v = regexp.MustCompile(`\([^)]*\)`).ReplaceAllString(v, " ")
	v = regexp.MustCompile(`\[[^\]]*\]`).ReplaceAllString(v, " ")
	v = regexp.MustCompile(`\{[^}]*\}`).ReplaceAllString(v, " ")

	// Прибираємо телефони та поштові індекси.
	v = regexp.MustCompile(`\b\d{2,4}[- ]\d{2}[- ]\d{2}(?:[- ]\d{2})?\b`).ReplaceAllString(v, " ")
	v = regexp.MustCompile(`\b\d{5}\b`).ReplaceAllString(v, " ")

	// Зайві службові хвости.
	if idx := indexOfAddressNoise(v); idx > 0 {
		v = v[:idx]
	}

	if idx := strings.Index(v, "+"); idx > 0 {
		v = v[:idx]
	}

	// Нормалізуємо розділювачі.
	v = strings.ReplaceAll(v, ";", ", ")
	v = strings.ReplaceAll(v, "|", ", ")
	v = regexp.MustCompile(`\s*,\s*`).ReplaceAllString(v, ", ")
	v = regexp.MustCompile(`\s*\.\s*`).ReplaceAllString(v, ". ")
	v = strings.Trim(v, " ,.-")
	return normalizeAddressSpaces(v)
}

func indexOfAddressNoise(v string) int {
	lower := strings.ToLower(v)
	keywords := []string{
		" ю/а ",
		" фактична",
		" завгосп",
		" централь",
		" охор",
		" режим роботи",
		" пожеж",
		" вхід ",
		" у дворі",
		" напроти ",
		" біля ",
		" на територ",
		" терітор",
		" корпус",
	}
	best := -1
	for _, kw := range keywords {
		idx := strings.Index(lower, kw)
		if idx >= 0 && (best < 0 || idx < best) {
			best = idx
		}
	}
	return best
}

func expandAddressAbbreviations(v string) string {
	v = normalizeAddressSpaces(v)
	abbrRules := []struct {
		pattern string
		repl    string
	}{
		{pattern: `(?i)(^|[\s,])м\.\s*`, repl: "${1}місто "},
		{pattern: `(?i)(^|[\s,])с\.\s*`, repl: "${1}село "},
		{pattern: `(?i)(^|[\s,])в\.\s*`, repl: "${1}вулиця "},
		{pattern: `(?i)(^|[\s,])вуп\.\s*`, repl: "${1}вулиця "},
		{pattern: `(?i)(^|[\s,])смт\.\s*`, repl: "${1}смт "},
		{pattern: `(?i)(^|[\s,])обл\.\s*`, repl: "${1}область "},
		{pattern: `(?i)(^|[\s,])вул\.\s*`, repl: "${1}вулиця "},
		{pattern: `(?i)(^|[\s,])пр\.\s*`, repl: "${1}проспект "},
		{pattern: `(?i)(^|[\s,])просп\.\s*`, repl: "${1}проспект "},
		{pattern: `(?i)(^|[\s,])пл\.\s*`, repl: "${1}площа "},
		{pattern: `(?i)(^|[\s,])бул\.\s*`, repl: "${1}бульвар "},
		{pattern: `(?i)(^|[\s,])пров\.\s*`, repl: "${1}провулок "},
	}
	for _, rule := range abbrRules {
		re := regexp.MustCompile(rule.pattern)
		v = re.ReplaceAllString(v, rule.repl)
	}

	repl := strings.NewReplacer(
		"м.", "місто ",
		"м ", "місто ",
		"с.", "село ",
		"с ", "село ",
		"в.", "вулиця ",
		"обл.", "область ",
		"обл ", "область ",
		"смт.", "смт ",
		"вул.", "вулиця ",
		"вул ", "вулиця ",
		"пр-т.", "проспект ",
		"пр-т", "проспект ",
		"просп.", "проспект ",
		"пл.", "площа ",
		"бул.", "бульвар ",
		"пров.", "провулок ",
	)
	v = repl.Replace(v)
	return normalizeAddressSpaces(v)
}

func ensureCountrySuffix(v string) string {
	v = normalizeAddressSpaces(v)
	if v == "" {
		return ""
	}
	lower := strings.ToLower(v)
	if strings.Contains(lower, "україн") || strings.Contains(lower, "ukraine") {
		return v
	}
	return v + ", Україна"
}

func extractCity(v string) (string, bool) {
	v = strings.NewReplacer(
		"смт.", "смт ",
		"м.", "місто ",
		"місто.", "місто ",
		"село.", "село ",
		"с.", "село ",
	).Replace(v)
	v = normalizeAddressSpaces(v)
	lower := strings.ToLower(v)
	switch {
	case strings.HasPrefix(lower, "місто "):
		return normalizeAddressSpaces(strings.TrimSpace(v[len("місто "):])), true
	case strings.HasPrefix(lower, "смт "):
		return normalizeAddressSpaces(strings.TrimSpace(v[len("смт "):])), true
	case strings.HasPrefix(lower, "селище "):
		return normalizeAddressSpaces(strings.TrimSpace(v[len("селище "):])), true
	case strings.HasPrefix(lower, "село "):
		return normalizeAddressSpaces(strings.TrimSpace(v[len("село "):])), true
	}

	// Підтримка рядків типу:
	// "Львівська область місто Львів" / "Львівська область село Зимна Вода".
	for _, marker := range []string{" місто ", " смт ", " селище ", " село "} {
		if idx := strings.Index(lower, marker); idx >= 0 {
			candidate := normalizeAddressSpaces(strings.TrimSpace(v[idx+len(marker):]))
			if candidate != "" {
				return candidate, true
			}
			before := normalizeAddressSpaces(strings.TrimSpace(v[:idx]))
			if before != "" && !isAdministrativePart(before) {
				return before, true
			}
		}
	}

	// Підтримка "Яворів м." / "Яворів місто".
	for _, marker := range []string{" місто", " м"} {
		if strings.HasSuffix(lower, marker) {
			before := normalizeAddressSpaces(strings.TrimSpace(v[:len(v)-len(marker)]))
			if before != "" && !isAdministrativePart(before) {
				return before, true
			}
		}
	}
	return "", false
}

func extractStreet(v string) (string, bool) {
	v = normalizeAddressSpaces(v)
	lower := strings.ToLower(v)
	prefixes := []string{
		"вулиця ",
		"проспект ",
		"бульвар ",
		"площа ",
		"провулок ",
		"шосе ",
		"узвіз ",
	}
	for _, pref := range prefixes {
		if strings.HasPrefix(lower, pref) {
			street := normalizeStreetName(strings.TrimSpace(v[len(pref):]))
			if street != "" {
				return street, true
			}
		}
	}

	// Підтримка "Маковея вулиця" / "Червоної Калини проспект".
	for _, pref := range prefixes {
		suffix := strings.TrimSpace(pref)
		if strings.HasSuffix(lower, " "+suffix) {
			street := normalizeStreetName(strings.TrimSpace(v[:len(v)-len(suffix)]))
			if street != "" {
				return street, true
			}
		}
	}
	return "", false
}

func splitCombinedLocalityStreetPart(part string) (string, string, string, bool) {
	part = normalizeAddressSpaces(part)
	if part == "" {
		return "", "", "", false
	}

	lower := strings.ToLower(part)
	streetPrefixes := []string{
		"вулиця ",
		"проспект ",
		"бульвар ",
		"площа ",
		"провулок ",
		"шосе ",
		"узвіз ",
	}

	streetIdx := -1
	for _, pref := range streetPrefixes {
		if strings.HasPrefix(lower, pref) {
			streetIdx = 0
			break
		}
		if idx := strings.Index(lower, " "+pref); idx >= 0 {
			idx++
			if streetIdx < 0 || idx < streetIdx {
				streetIdx = idx
			}
		}
	}
	if streetIdx <= 0 {
		return "", "", "", false
	}

	localityPart := normalizeAddressSpaces(part[:streetIdx])
	streetPart := normalizeAddressSpaces(part[streetIdx:])
	if localityPart == "" || streetPart == "" {
		return "", "", "", false
	}

	city, ok := extractCity(localityPart)
	if !ok {
		return "", "", "", false
	}

	street, ok := extractStreet(streetPart)
	if !ok {
		return "", "", "", false
	}
	house := extractHouseNumber(streetPart)
	return city, street, house, true
}

func normalizeStreetName(v string) string {
	v = normalizeAddressSpaces(v)
	if v == "" {
		return ""
	}
	// Якщо номер будинку написали разом зі вулицею, відкидаємо номер.
	house := extractHouseNumber(v)
	if house != "" {
		v = strings.TrimSpace(strings.Replace(v, house, "", 1))
	}
	return normalizeAddressSpaces(v)
}

func extractHouseNumber(v string) string {
	v = normalizeAddressSpaces(v)
	if v == "" {
		return ""
	}
	re := regexp.MustCompile(`(?i)\d+[0-9\p{L}/-]*`)
	return normalizeAddressSpaces(re.FindString(v))
}

func isAdministrativePart(v string) bool {
	l := strings.ToLower(normalizeAddressSpaces(v))
	if l == "" {
		return true
	}
	adminWords := []string{
		"район",
		"область",
		"громада",
		"україна",
		"украина",
		"ukraine",
	}
	for _, w := range adminWords {
		if strings.Contains(l, w) {
			return true
		}
	}
	return false
}

func collectDistrictHints(address map[string]string, displayName string) []string {
	hints := make([]string, 0, 8)
	addHint := func(v string) {
		v = strings.TrimSpace(v)
		if v == "" {
			return
		}
		for _, existing := range hints {
			if strings.EqualFold(existing, v) {
				return
			}
		}
		hints = append(hints, v)
	}

	keys := []string{
		"city_district",
		"district",
		"suburb",
		"borough",
		"county",
		"state_district",
		"municipality",
	}
	for _, key := range keys {
		addHint(address[key])
	}

	parts := strings.Split(displayName, ",")
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if strings.Contains(strings.ToLower(part), "район") {
			addHint(part)
		}
	}

	return hints
}
