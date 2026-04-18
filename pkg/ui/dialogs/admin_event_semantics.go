package dialogs

type adminEventSemanticInfo struct {
	TypeLabel   string
	Family      string
	PaletteCode int
}

type adminEventOverrideScenario struct {
	Label string
	SC1   int64
	Match map[int64]struct{}
}

type adminEventPaletteGroup struct {
	Label       string
	Codes       []int
	PreviewCode int
}

var adminEventSemanticBySC1 = map[int64]adminEventSemanticInfo{
	1:  {TypeLabel: "тривога", Family: "alarm", PaletteCode: 1},
	21: {TypeLabel: "паніка", Family: "alarm", PaletteCode: 1},
	22: {TypeLabel: "проникнення", Family: "alarm", PaletteCode: 1},
	23: {TypeLabel: "медична", Family: "alarm", PaletteCode: 1},
	24: {TypeLabel: "газ", Family: "alarm", PaletteCode: 1},
	25: {TypeLabel: "саботаж", Family: "alarm", PaletteCode: 1},
	2:  {TypeLabel: "тех. тривога", Family: "tech", PaletteCode: 2},
	3:  {TypeLabel: "тех. тривога", Family: "tech", PaletteCode: 2},
	4:  {TypeLabel: "живлення", Family: "tech", PaletteCode: 2},
	5:  {TypeLabel: "відновлення", Family: "restore", PaletteCode: 5},
	9:  {TypeLabel: "відновлення", Family: "restore", PaletteCode: 5},
	13: {TypeLabel: "відновлення", Family: "restore", PaletteCode: 5},
	17: {TypeLabel: "відновлення", Family: "restore", PaletteCode: 5},
	10: {TypeLabel: "під охороною", Family: "info", PaletteCode: 10},
	11: {TypeLabel: "знято з охорони", Family: "info", PaletteCode: 11},
	12: {TypeLabel: "нема зв'язку", Family: "info", PaletteCode: 12},
	14: {TypeLabel: "знято з охорони", Family: "info", PaletteCode: 14},
	16: {TypeLabel: "тестове", Family: "test", PaletteCode: 16},
	18: {TypeLabel: "знято з охорони", Family: "info", PaletteCode: 14},
	26: {TypeLabel: "живлення", Family: "tech", PaletteCode: 2},
	27: {TypeLabel: "живлення", Family: "tech", PaletteCode: 2},
	28: {TypeLabel: "на зв'язку", Family: "info", PaletteCode: 5},
	29: {TypeLabel: "офлайн", Family: "info", PaletteCode: 2},
	30: {TypeLabel: "система", Family: "info", PaletteCode: 6},
}

var adminEventOverrideScenarios = []adminEventOverrideScenario{
	{Label: "Тривога", SC1: 1, Match: setOfInt64(1, 21, 22, 23, 24, 25)},
	{Label: "Тривога техн.", SC1: 2, Match: setOfInt64(2, 3)},
	{Label: "Відновлення", SC1: 5, Match: setOfInt64(5, 9, 13, 17)},
	{Label: "Інформація", SC1: 6, Match: setOfInt64(6, 10, 11, 14, 18, 28, 29, 30)},
	{Label: "Подію заборонено", SC1: 12, Match: setOfInt64(12)},
	{Label: "Тестове повідомлення", SC1: 16, Match: setOfInt64(16)},
}

var adminEventPaletteGroups = []adminEventPaletteGroup{
	{Label: "Тривоги", Codes: []int{1, 21, 22, 23, 24, 25}, PreviewCode: 1},
	{Label: "Несправності / живлення / зв'язок", Codes: []int{2, 3, 4, 12, 26, 27, 29}, PreviewCode: 2},
	{Label: "Відновлення / на зв'язку", Codes: []int{5, 9, 13, 17, 28}, PreviewCode: 5},
	{Label: "Постановка під охорону", Codes: []int{7, 8, 10}, PreviewCode: 10},
	{Label: "Зняття з охорони", Codes: []int{11, 14, 18}, PreviewCode: 11},
	{Label: "Інформація / тест / сервіс", Codes: []int{6, 16, 30}, PreviewCode: 6},
}

func adminEventTypeLabel(sc1 *int64) string {
	if sc1 == nil {
		return "інформація"
	}
	if info, ok := adminEventSemanticBySC1[*sc1]; ok {
		return info.TypeLabel
	}
	return "інформація"
}

func adminEventMatchesFamily(sc1 *int64, family string) bool {
	if sc1 == nil {
		return family == "info"
	}
	if info, ok := adminEventSemanticBySC1[*sc1]; ok {
		return info.Family == family
	}
	return family == "info"
}

func adminEventPaletteCode(sc1 *int64) int {
	if sc1 == nil {
		return 6
	}
	if info, ok := adminEventSemanticBySC1[*sc1]; ok {
		return info.PaletteCode
	}
	return int(*sc1)
}

func adminEventOverrideLabelFromSC1(sc1 *int64) string {
	if sc1 == nil {
		return "Інформація"
	}
	for _, scenario := range adminEventOverrideScenarios {
		if _, ok := scenario.Match[*sc1]; ok {
			return scenario.Label
		}
	}
	return "Інформація"
}

func adminEventOverrideSC1FromLabel(label string) *int64 {
	for _, scenario := range adminEventOverrideScenarios {
		if scenario.Label == label {
			return i64(scenario.SC1)
		}
	}
	return i64(6)
}

func adminEventOverrideLabels() []string {
	labels := make([]string, 0, len(adminEventOverrideScenarios))
	for _, scenario := range adminEventOverrideScenarios {
		labels = append(labels, scenario.Label)
	}
	return labels
}

func adminEventColorOptions() []eventColorOption {
	options := make([]eventColorOption, 0, len(adminEventPaletteGroups))
	for _, group := range adminEventPaletteGroups {
		options = append(options, eventColorOption{
			Label:       group.Label,
			Codes:       append([]int(nil), group.Codes...),
			PreviewCode: group.PreviewCode,
		})
	}
	return options
}

func setOfInt64(values ...int64) map[int64]struct{} {
	result := make(map[int64]struct{}, len(values))
	for _, value := range values {
		result[value] = struct{}{}
	}
	return result
}
