package dialogs

type messageCategoryOption struct {
	Label string
	SC1   *int64
}

func i64(v int64) *int64 {
	return &v
}

func messageCategoryOptions() []messageCategoryOption {
	return []messageCategoryOption{
		{Label: "Тривога", SC1: i64(1)},
		{Label: "Технічна тривога", SC1: i64(2)},
		{Label: "Відновлення", SC1: i64(5)},
		{Label: "Інформація", SC1: i64(6)},
		{Label: "Під охороною", SC1: i64(10)},
		{Label: "Знято з охорони", SC1: i64(11)},
		{Label: "Немає зв'язку", SC1: i64(12)},
		{Label: "Тестове", SC1: i64(16)},
		{Label: "Інше / без категорії", SC1: nil},
	}
}

func messageTypeLabel(sc1 *int64) string {
	if sc1 == nil {
		return "інформація"
	}
	switch *sc1 {
	case 1:
		return "тривога"
	case 2, 3:
		return "тех. тривога"
	case 5, 9, 13, 17:
		return "відновлення"
	case 10:
		return "під охороною"
	case 11, 14, 18:
		return "знято з охорони"
	case 12:
		return "нема зв'язку"
	case 16:
		return "тестове"
	default:
		return "інформація"
	}
}

func categoryLabelFromSC1(sc1 *int64) string {
	if sc1 == nil {
		return "Інше / без категорії"
	}
	for _, c := range messageCategoryOptions() {
		if c.SC1 != nil && *c.SC1 == *sc1 {
			return c.Label
		}
	}
	return "Інше / без категорії"
}

func categorySC1FromLabel(label string) *int64 {
	for _, c := range messageCategoryOptions() {
		if c.Label == label {
			if c.SC1 == nil {
				return nil
			}
			v := *c.SC1
			return &v
		}
	}
	return nil
}

func sc1MatchesFamily(sc1 *int64, family string) bool {
	v := int64(0)
	if sc1 != nil {
		v = *sc1
	}

	switch family {
	case "alarm":
		return v == 1
	case "tech":
		return v == 2 || v == 3
	case "restore":
		return v == 5 || v == 9 || v == 13 || v == 17
	case "test":
		return v == 16
	case "info":
		if sc1 == nil {
			return true
		}
		return v == 6 || v == 10 || v == 11 || v == 12 || v == 14 || v == 18
	default:
		return true
	}
}
