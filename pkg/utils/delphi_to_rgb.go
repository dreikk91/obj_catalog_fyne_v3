package utils

import (
	"image/color"
)

// ColorPair описує пару кольорів (текст + фон рядка)
type ColorPair struct {
	Text color.NRGBA
	Row  color.NRGBA
}

// Базові (стандартні) карти кольорів для світлої теми
func defaultLightColorMapping() map[int]ColorPair {
	return map[int]ColorPair{
		1: { // Alarm
			Text: color.NRGBA{R: 255, G: 255, B: 128, A: 255}, // rgb(255,255,128)
			Row:  color.NRGBA{R: 183, G: 0, B: 0, A: 255},    // rgb(183,0,0)
		},
		2: { // Tech alarm
			Text: color.NRGBA{R: 0, G: 0, B: 0, A: 255},      // rgb(0,0,0)
			Row:  color.NRGBA{R: 250, G: 182, B: 69, A: 255}, // rgb(250,182,69)
		},
		5: { // Restore
			Text: color.NRGBA{R: 0, G: 0, B: 0, A: 255},        // rgb(0,0,0)
			Row:  color.NRGBA{R: 160, G: 160, B: 164, A: 255}, // rgb(160,160,164)
		},
		6: { // Info
			Text: color.NRGBA{R: 0, G: 0, B: 0, A: 255},        // rgb(0,0,0)
			Row:  color.NRGBA{R: 255, G: 255, B: 255, A: 255}, // rgb(255,255,255)
		},
		7: { // PartArmedOn
			Text: color.NRGBA{R: 0, G: 102, B: 51, A: 255},    // rgb(0,102,51)
			Row:  color.NRGBA{R: 255, G: 255, B: 255, A: 255}, // rgb(255,255,255)
		},
		8: { // PartArmedOn
			Text: color.NRGBA{R: 0, G: 102, B: 51, A: 255},    // rgb(0,102,51)
			Row:  color.NRGBA{R: 255, G: 255, B: 255, A: 255}, // rgb(255,255,255)
		},
		9: { // Restore
			Text: color.NRGBA{R: 0, G: 0, B: 0, A: 255},        // rgb(0,0,0)
			Row:  color.NRGBA{R: 160, G: 160, B: 164, A: 255}, // rgb(160,160,164)
		},
		10: { // ArmedOn
			Text: color.NRGBA{R: 255, G: 255, B: 255, A: 255}, // rgb(255,255,255)
			Row:  color.NRGBA{R: 99, G: 156, B: 111, A: 255},  // rgb(99,156,111)
		},
		12: { // ConnFailed
			Text: color.NRGBA{R: 0, G: 0, B: 0, A: 255},        // rgb(0,0,0)
			Row:  color.NRGBA{R: 255, G: 255, B: 128, A: 255}, // rgb(255,255,128)
		},
		13: { // Restore
			Text: color.NRGBA{R: 0, G: 0, B: 0, A: 255},        // rgb(0,0,0)
			Row:  color.NRGBA{R: 160, G: 160, B: 164, A: 255}, // rgb(160,160,164)
		},
		14: { // PartArmedOff
			Text: color.NRGBA{R: 255, G: 255, B: 255, A: 255}, // rgb(255,255,255)
			Row:  color.NRGBA{R: 128, G: 128, B: 0, A: 255},   // rgb(128,128,0)
		},
		17: { // Restore
			Text: color.NRGBA{R: 0, G: 0, B: 0, A: 255},        // rgb(0,0,0)
			Row:  color.NRGBA{R: 160, G: 160, B: 164, A: 255}, // rgb(160,160,164)
		},
		18: { // PartArmedOff
			Text: color.NRGBA{R: 255, G: 255, B: 255, A: 255}, // rgb(255,255,255)
			Row:  color.NRGBA{R: 128, G: 128, B: 0, A: 255},   // rgb(128,128,0)
		},
	}
}

// Базові (стандартні) карти кольорів для темної теми
func defaultDarkColorMapping() map[int]ColorPair {
	return map[int]ColorPair{
		1: { // Alarm
			Text: color.NRGBA{R: 255, G: 200, B: 200, A: 255}, // rgb(255,200,200)
			Row:  color.NRGBA{R: 90, G: 20, B: 20, A: 255},    // rgb(90,20,20)
		},
		2: { // Tech alarm
			Text: color.NRGBA{R: 255, G: 220, B: 170, A: 255}, // rgb(255,220,170)
			Row:  color.NRGBA{R: 80, G: 60, B: 30, A: 255},    // rgb(80,60,30)
		},
		5: { // Restore
			Text: color.NRGBA{R: 210, G: 210, B: 215, A: 255}, // rgb(210,210,215)
			Row:  color.NRGBA{R: 45, G: 45, B: 50, A: 255},    // rgb(45,45,50)
		},
		6: { // Info
			Text: color.NRGBA{R: 220, G: 220, B: 220, A: 255}, // rgb(220,220,220)
			Row:  color.NRGBA{R: 30, G: 30, B: 30, A: 255},    // rgb(30,30,30)
		},
		7: { // PartArmedOn
			Text: color.NRGBA{R: 170, G: 255, B: 210, A: 255}, // rgb(170,255,210)
			Row:  color.NRGBA{R: 20, G: 70, B: 45, A: 255},    // rgb(20,70,45)
		},
		8: { // PartArmedOn
			Text: color.NRGBA{R: 170, G: 255, B: 210, A: 255}, // rgb(170,255,210)
			Row:  color.NRGBA{R: 20, G: 70, B: 45, A: 255},    // rgb(20,70,45)
		},
		9: { // Restore
			Text: color.NRGBA{R: 210, G: 210, B: 215, A: 255}, // rgb(210,210,215)
			Row:  color.NRGBA{R: 45, G: 45, B: 50, A: 255},    // rgb(45,45,50)
		},
		10: { // ArmedOn
			Text: color.NRGBA{R: 200, G: 255, B: 225, A: 255}, // rgb(200,255,225)
			Row:  color.NRGBA{R: 25, G: 85, B: 60, A: 255},    // rgb(25,85,60)
		},
		12: { // ConnFailed
			Text: color.NRGBA{R: 255, G: 245, B: 180, A: 255}, // rgb(255,245,180)
			Row:  color.NRGBA{R: 85, G: 75, B: 30, A: 255},    // rgb(85,75,30)
		},
		13: { // Restore
			Text: color.NRGBA{R: 210, G: 210, B: 215, A: 255}, // rgb(210,210,215)
			Row:  color.NRGBA{R: 45, G: 45, B: 50, A: 255},    // rgb(45,45,50)
		},
		14: { // PartArmedOff
			Text: color.NRGBA{R: 255, G: 240, B: 180, A: 255}, // rgb(255,240,180)
			Row:  color.NRGBA{R: 90, G: 85, B: 30, A: 255},    // rgb(90,85,30)
		},
		17: { // Restore
			Text: color.NRGBA{R: 210, G: 210, B: 215, A: 255}, // rgb(210,210,215)
			Row:  color.NRGBA{R: 45, G: 45, B: 50, A: 255},    // rgb(45,45,50)
		},
		18: { // PartArmedOff
			Text: color.NRGBA{R: 255, G: 240, B: 180, A: 255}, // rgb(255,240,180)
			Row:  color.NRGBA{R: 90, G: 85, B: 30, A: 255},    // rgb(90,85,30)
		},
	}
}

// Поточні карти кольорів, які можна змінювати під час роботи додатку
var (
	lightColorMapping = defaultLightColorMapping()
	darkColorMapping  = defaultDarkColorMapping()
)

// SelectColorNRGBA повертає поточні (можливо змінені користувачем) кольори для світлої теми
func SelectColorNRGBA(colorValue int) (text, row color.NRGBA) {
	if c, ok := lightColorMapping[colorValue]; ok {
		return c.Text, c.Row
	}

	// default (Delphi 0,0 → чорний текст на білому)
	return color.NRGBA{R: 0, G: 0, B: 0, A: 255},   // rgb(0,0,0)
		color.NRGBA{R: 255, G: 255, B: 255, A: 255} // rgb(255, 255, 255)
}

// SelectColorNRGBADark повертає поточні (можливо змінені користувачем) кольори для темної теми
func SelectColorNRGBADark(colorValue int) (text, row color.NRGBA) {
	if c, ok := darkColorMapping[colorValue]; ok {
		return c.Text, c.Row
	}

	// default (dark)
	return color.NRGBA{R: 220, G: 220, B: 220, A: 255}, // rgb(220,220,220)
		color.NRGBA{R: 30, G: 30, B: 30, A: 255}        // rgb(30,30,30)
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
			if def, ok := defaultDarkColorMapping()[code]; ok {
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
		if def, ok := defaultLightColorMapping()[code]; ok {
			c.Text = def.Text
		} else {
			c.Text = color.NRGBA{R: 0, G: 0, B: 0, A: 255}
		}
	}
	lightColorMapping[code] = c
}

// ResetEventColorsToDefault скидає всі кольори подій до стандартних
// для поточної (isDark=true/false) теми.
func ResetEventColorsToDefault(isDark bool) {
	if isDark {
		darkColorMapping = defaultDarkColorMapping()
	} else {
		lightColorMapping = defaultLightColorMapping()
	}
}

