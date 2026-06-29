package utils

import (
	"image/color"
	"maps"
)

// ColorPair описує пару кольорів (текст + фон рядка)
type ColorPair struct {
	Text color.NRGBA
	Row  color.NRGBA
}

func cloneColorMapping(src map[int]ColorPair) map[int]ColorPair {
	return maps.Clone(src)
}

// Базові (стандартні) карти кольорів для світлої теми.
//
// Палітра організована за 5 семантичними рівнями:
//
//	🔴 Критичний  — пожежа, проникнення, паніка, мед.тривога, газ
//	🟠 Тривога    — тампер, несправність, техн.тривоги
//	🟡 Попередження — втрата зв'язку, АКБ, живлення
//	🟢 Норма      — постановка, зняття, тест, онлайн, часткова охорона
//	⚪ Інфо       — відновлення, системні, сервісні
//
// Принципи: пастельний фон + насичений текст → контраст WCAG AA.
func defaultLightColorMapping() map[int]ColorPair {
	// --- Критичний рівень: ніжний червоний фон, насичений червоний текст ---
	critical := ColorPair{
		Text: color.NRGBA{R: 198, G: 40, B: 40, A: 255},   // #C62828
		Row:  color.NRGBA{R: 253, G: 236, B: 234, A: 255}, // #FDECEA
	}

	// --- Тривога: теплий помаранчевий фон, яскравий оранж текст ---
	alarm := ColorPair{
		Text: color.NRGBA{R: 230, G: 81, B: 0, A: 255},    // #E65100
		Row:  color.NRGBA{R: 255, G: 243, B: 224, A: 255}, // #FFF3E0
	}

	// --- Попередження: м'який жовтий фон, темний жовтий текст ---
	warning := ColorPair{
		Text: color.NRGBA{R: 245, G: 127, B: 23, A: 255},  // #F57F17
		Row:  color.NRGBA{R: 255, G: 253, B: 231, A: 255}, // #FFFDE7
	}

	// --- Норма: світлий зелений фон, зелений текст ---
	normal := ColorPair{
		Text: color.NRGBA{R: 46, G: 125, B: 50, A: 255},   // #2E7D32
		Row:  color.NRGBA{R: 232, G: 245, B: 233, A: 255}, // #E8F5E9
	}

	// --- Інфо: нейтральний (білий) фон, темно-сірий текст ---
	info := ColorPair{
		Text: color.NRGBA{R: 66, G: 66, B: 66, A: 255},    // #424242
		Row:  color.NRGBA{R: 255, G: 255, B: 255, A: 255}, // #FFFFFF
	}

	return map[int]ColorPair{
		// === Критичний рівень ===
		1:  critical, // Alarm (пожежна тривога)
		21: critical, // Panic (тривожна кнопка)
		22: critical, // Burglary (проникнення)
		23: critical, // Medical (медична тривога)
		24: critical, // Gas (газова тривога)

		// === Тривога ===
		2:  alarm, // Tech alarm (технічна тривога)
		25: alarm, // Tamper (саботаж)
		3:  alarm, // PowerFail (втрата 220В — критичне для обладнання)

		// === Попередження ===
		4:  warning, // BatteryLow (низький заряд АКБ)
		12: warning, // ConnFailed (втрата зв'язку)
		26: warning, // PowerFail extended (проблеми з живленням)
		27: warning, // BatteryLow extended (стан АКБ)
		29: warning, // Offline (прилад не на зв'язку)

		// === Норма ===
		7:  normal, // PartArmedOn (часткова постановка)
		8:  normal, // PartArmedOn (часткова постановка)
		10: normal, // ArmedOn (під охороною)
		11: normal, // Disarmed (зняття з охорони)
		14: normal, // PartArmedOff (часткове зняття)
		16: normal, // Test (тестовий сигнал)
		18: normal, // PartArmedOff (часткове зняття)
		28: normal, // Online (прилад на зв'язку)

		// === Інфо ===
		5:  info, // Restore (відновлення)
		6:  info, // Info (інформаційна подія)
		9:  info, // Restore (відновлення)
		13: info, // Restore (відновлення)
		17: info, // Restore (відновлення)
		30: info, // System/Service (системна/сервісна подія)
	}
}

// Базові (стандартні) карти кольорів для темної теми.
// Структура аналогічна світлій темі — 5 семантичних рівнів.
func defaultDarkColorMapping() map[int]ColorPair {
	// --- Критичний рівень: темно-червоний фон, світлий червонуватий текст ---
	critical := ColorPair{
		Text: color.NRGBA{R: 255, G: 138, B: 128, A: 255}, // #FF8A80
		Row:  color.NRGBA{R: 78, G: 21, B: 21, A: 255},    // #4E1515
	}

	// --- Тривога: темно-помаранчевий фон, світлий оранж текст ---
	alarm := ColorPair{
		Text: color.NRGBA{R: 255, G: 171, B: 145, A: 255}, // #FFAB91
		Row:  color.NRGBA{R: 78, G: 44, B: 16, A: 255},    // #4E2C10
	}

	// --- Попередження: темно-жовтий фон, світлий жовтий текст ---
	warning := ColorPair{
		Text: color.NRGBA{R: 255, G: 213, B: 79, A: 255}, // #FFD54F
		Row:  color.NRGBA{R: 78, G: 68, B: 16, A: 255},   // #4E4410
	}

	// --- Норма: темно-зелений фон, світлий зелений текст ---
	normal := ColorPair{
		Text: color.NRGBA{R: 165, G: 214, B: 167, A: 255}, // #A5D6A7
		Row:  color.NRGBA{R: 27, G: 58, B: 27, A: 255},    // #1B3A1B
	}

	// --- Інфо: нейтральний темний фон, світло-сірий текст ---
	info := ColorPair{
		Text: color.NRGBA{R: 189, G: 189, B: 189, A: 255}, // #BDBDBD
		Row:  color.NRGBA{R: 44, G: 44, B: 44, A: 255},    // #2C2C2C
	}

	return map[int]ColorPair{
		// === Критичний рівень ===
		1:  critical, // Alarm (пожежна тривога)
		21: critical, // Panic (тривожна кнопка)
		22: critical, // Burglary (проникнення)
		23: critical, // Medical (медична тривога)
		24: critical, // Gas (газова тривога)

		// === Тривога ===
		2:  alarm, // Tech alarm (технічна тривога)
		25: alarm, // Tamper (саботаж)
		3:  alarm, // PowerFail (втрата 220В)

		// === Попередження ===
		4:  warning, // BatteryLow (низький заряд АКБ)
		12: warning, // ConnFailed (втрата зв'язку)
		26: warning, // PowerFail extended
		27: warning, // BatteryLow extended
		29: warning, // Offline (прилад не на зв'язку)

		// === Норма ===
		7:  normal, // PartArmedOn
		8:  normal, // PartArmedOn
		10: normal, // ArmedOn (під охороною)
		11: normal, // Disarmed (зняття з охорони)
		14: normal, // PartArmedOff
		16: normal, // Test (тестовий сигнал)
		18: normal, // PartArmedOff
		28: normal, // Online (прилад на зв'язку)

		// === Інфо ===
		5:  info, // Restore (відновлення)
		6:  info, // Info (інформаційна подія)
		9:  info, // Restore
		13: info, // Restore
		17: info, // Restore
		30: info, // System/Service
	}
}

// Поточні карти кольорів, які можна змінювати під час роботи додатку
var (
	defaultLightPalette = defaultLightColorMapping()
	defaultDarkPalette  = defaultDarkColorMapping()
	lightColorMapping   = cloneColorMapping(defaultLightPalette)
	darkColorMapping    = cloneColorMapping(defaultDarkPalette)
)

// SelectColorNRGBA повертає поточні (можливо змінені користувачем) кольори для світлої теми
func SelectColorNRGBA(colorValue int) (text, row color.NRGBA) {
	if c, ok := lightColorMapping[colorValue]; ok {
		return c.Text, c.Row
	}

	// default (Delphi 0,0 → чорний текст на білому)
	return color.NRGBA{R: 0, G: 0, B: 0, A: 255}, // rgb(0,0,0)
		color.NRGBA{R: 255, G: 255, B: 255, A: 255} // rgb(255, 255, 255)
}

// SelectColorNRGBADark повертає поточні (можливо змінені користувачем) кольори для темної теми
func SelectColorNRGBADark(colorValue int) (text, row color.NRGBA) {
	if c, ok := darkColorMapping[colorValue]; ok {
		return c.Text, c.Row
	}

	// default (dark)
	return color.NRGBA{R: 220, G: 220, B: 220, A: 255}, // rgb(220,220,220)
		color.NRGBA{R: 30, G: 30, B: 30, A: 255} // rgb(30,30,30)
}

// SelectObjectColorNRGBA повертає стандартні кольори списку об'єктів для світлої теми.
// Ця палітра не змінюється через налаштування кольорів подій.
func SelectObjectColorNRGBA(colorValue int) (text, row color.NRGBA) {
	if c, ok := defaultLightPalette[colorValue]; ok {
		return c.Text, c.Row
	}

	return color.NRGBA{R: 0, G: 0, B: 0, A: 255},
		color.NRGBA{R: 255, G: 255, B: 255, A: 255}
}

// SelectObjectColorNRGBADark повертає стандартні кольори списку об'єктів для темної теми.
// Ця палітра не змінюється через налаштування кольорів подій.
func SelectObjectColorNRGBADark(colorValue int) (text, row color.NRGBA) {
	if c, ok := defaultDarkPalette[colorValue]; ok {
		return c.Text, c.Row
	}

	return color.NRGBA{R: 220, G: 220, B: 220, A: 255},
		color.NRGBA{R: 30, G: 30, B: 30, A: 255}
}

// GetEventRowColor повертає поточний колір фону рядка для коду події
func GetEventRowColor(code int, isDark bool) color.NRGBA {
	if isDark {
		if c, ok := darkColorMapping[code]; ok {
			return c.Row
		}
		_, row := SelectColorNRGBADark(code)
		return row
	}

	if c, ok := lightColorMapping[code]; ok {
		return c.Row
	}
	_, row := SelectColorNRGBA(code)
	return row
}

// GetEventTextColor повертає поточний колір тексту для коду події.
func GetEventTextColor(code int, isDark bool) color.NRGBA {
	if isDark {
		if c, ok := darkColorMapping[code]; ok {
			return c.Text
		}
		text, _ := SelectColorNRGBADark(code)
		return text
	}

	if c, ok := lightColorMapping[code]; ok {
		return c.Text
	}
	text, _ := SelectColorNRGBA(code)
	return text
}

// SetEventRowColor змінює тільки колір фону рядка для вказаного коду події.
// Текст залишається таким, як у поточному мапінгу, щоб зберегти читабельність.
func SetEventRowColor(code int, isDark bool, row color.NRGBA) {
	if isDark {
		c, ok := darkColorMapping[code]
		if !ok {
			c = ColorPair{}
		}
		c.Row = row
		// якщо Text порожній, беремо стандартне значення
		if c.Text == (color.NRGBA{}) {
			if def, ok := defaultDarkPalette[code]; ok {
				c.Text = def.Text
			} else {
				c.Text = color.NRGBA{R: 255, G: 255, B: 255, A: 255}
			}
		}
		darkColorMapping[code] = c
		return
	}

	c, ok := lightColorMapping[code]
	if !ok {
		c = ColorPair{}
	}
	c.Row = row
	if c.Text == (color.NRGBA{}) {
		if def, ok := defaultLightPalette[code]; ok {
			c.Text = def.Text
		} else {
			c.Text = color.NRGBA{R: 0, G: 0, B: 0, A: 255}
		}
	}
	lightColorMapping[code] = c
}

// SetEventTextColor змінює тільки колір тексту для вказаного коду події.
// Фон зберігається з поточного мапінгу або береться зі стандартної палітри.
func SetEventTextColor(code int, isDark bool, text color.NRGBA) {
	if isDark {
		c, ok := darkColorMapping[code]
		if !ok {
			c = ColorPair{}
		}
		c.Text = text
		if c.Row == (color.NRGBA{}) {
			if def, ok := defaultDarkPalette[code]; ok {
				c.Row = def.Row
			} else {
				c.Row = color.NRGBA{R: 30, G: 30, B: 30, A: 255}
			}
		}
		darkColorMapping[code] = c
		return
	}

	c, ok := lightColorMapping[code]
	if !ok {
		c = ColorPair{}
	}
	c.Text = text
	if c.Row == (color.NRGBA{}) {
		if def, ok := defaultLightPalette[code]; ok {
			c.Row = def.Row
		} else {
			c.Row = color.NRGBA{R: 255, G: 255, B: 255, A: 255}
		}
	}
	lightColorMapping[code] = c
}

// ResetEventColorsToDefault скидає всі кольори подій до стандартних
// для поточної (isDark=true/false) теми.
func ResetEventColorsToDefault(isDark bool) {
	if isDark {
		darkColorMapping = cloneColorMapping(defaultDarkPalette)
	} else {
		lightColorMapping = cloneColorMapping(defaultLightPalette)
	}
}
