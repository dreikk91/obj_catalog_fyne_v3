package dialogs

func i64(v int64) *int64 {
	return &v
}

func messageTypeLabel(sc1 *int64) string {
	if sc1 == nil {
		return "інформація"
	}
	switch *sc1 {
	case 1:
		return "тривога"
	case 21:
		return "паніка"
	case 22:
		return "проникнення"
	case 23:
		return "медична"
	case 24:
		return "газ"
	case 25:
		return "саботаж"
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
	case 28:
		return "на зв'язку"
	case 29:
		return "офлайн"
	case 30:
		return "система"
	default:
		return "інформація"
	}
}

func sc1MatchesFamily(sc1 *int64, family string) bool {
	v := int64(0)
	if sc1 != nil {
		v = *sc1
	}

	switch family {
	case "alarm":
		return v == 1 || v == 21 || v == 22 || v == 23 || v == 24 || v == 25
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
		return v == 6 || v == 10 || v == 11 || v == 12 || v == 14 || v == 18 || v == 28 || v == 29 || v == 30
	default:
		return true
	}
}
