package utils

import (
	"image/color"
)

// ChangeItemColorNRGBA визначає кольори елемента на основі стану об'єкта
func ChangeItemColorNRGBA(
	alarmstate, guardstate, techAlarmState, isConstate int64,
	isDark bool,
) (textColor, rowColor color.NRGBA) {

	// Вибір палітри залежно від теми
	selectEventColor := SelectColorNRGBA
	if isDark {
		selectEventColor = SelectColorNRGBADark
	}

	// Дефолтні кольори (на випадок, якщо не підійде жоден case)
	if isDark {
		textColor = color.NRGBA{R: 123, G: 123, B: 143, A: 255}  // rgb(123,123,143)
		rowColor = color.NRGBA{R: 55, G: 80, B: 38, A: 255}      // rgb(55,80,38)
	} else {
		textColor = color.NRGBA{R: 123, G: 23, B: 143, A: 255}   // rgb(123,23,143)
		rowColor = color.NRGBA{R: 55, G: 240, B: 38, A: 255}     // rgb(55,240,38)
	}

	switch {
	// 1. Під охороною, є зв'язок, немає тривоги, немає тех.тривоги
	case isConstate == 1 && alarmstate == 0 && guardstate >= 1 && guardstate <= 3 && techAlarmState == 0:
		textColor, rowColor = selectEventColor(10) // ArmedOn

	// 2. Є зв'язок, тривога, під охороною, немає тех.тривоги (дублюється в Python)
	case isConstate == 1 && alarmstate == 1 && guardstate == 1 && techAlarmState >= 0:
		textColor, rowColor = selectEventColor(1) // Alarm

	// 3. Є зв'язок, немає тривоги, знято з охорони, немає тех.тривоги
	case isConstate == 1 && alarmstate == 0 && guardstate == 0 && techAlarmState == 0:
		if isDark {
			textColor = color.NRGBA{R: 230, G: 230, B: 250, A: 255}  // rgb(230,230,250)
			rowColor = color.NRGBA{R: 100, G: 15, B: 120, A: 255}    // rgb(100,15,120)
		} else {
			textColor = color.NRGBA{R: 255, G: 255, B: 255, A: 255}  // rgb(255,255,255)
			rowColor = color.NRGBA{R: 170, G: 14, B: 201, A: 255}    // rgb(170,14,201)
		}

	// 4. Немає зв'язку, немає тривоги, знято з охорони, немає тех.тривоги
	case isConstate == 0 && alarmstate == 0 && guardstate == 0 && techAlarmState == 0:
		if isDark {
			textColor = color.NRGBA{R: 230, G: 230, B: 250, A: 255}  // rgb(230,230,250)
			rowColor = color.NRGBA{R: 100, G: 15, B: 120, A: 255}    // rgb(100,15,120)
		} else {
			textColor = color.NRGBA{R: 255, G: 255, B: 255, A: 255}  // rgb(255,255,255)
			rowColor = color.NRGBA{R: 170, G: 14, B: 201, A: 255}    // rgb(170,14,201)
		}

	// 5. Є зв'язок, немає тривоги, guardstate=2, немає тех.тривоги
	case isConstate == 1 && alarmstate == 0 && guardstate == 2 && techAlarmState == 0:
		textColor, rowColor = selectEventColor(10) // ArmedOn

	// 6. Немає зв'язку, немає тривоги, під охороною, немає тех.тривоги
	case isConstate == 0 && alarmstate == 0 && guardstate >= 1 && techAlarmState == 0:
		if isDark {
			textColor = color.NRGBA{R: 255, G: 250, B: 180, A: 255}  // rgb(255,250,180)
			rowColor = color.NRGBA{R: 90, G: 90, B: 20, A: 255}      // rgb(90,90,20)
		} else {
			textColor = color.NRGBA{R: 0, G: 0, B: 0, A: 255}        // rgb(0,0,0)
			rowColor = color.NRGBA{R: 225, G: 235, B: 35, A: 255}    // rgb(225,235,35)
		}

	// 7. Немає зв'язку, тривога, під охороною, немає тех.тривоги
	case isConstate == 0 && alarmstate == 1 && guardstate == 1 && techAlarmState == 0:
		textColor, rowColor = selectEventColor(1) // Alarm

	// 8. Немає зв'язку, тривога, знято з охорони, немає тех.тривоги
	case isConstate == 0 && alarmstate == 1 && guardstate == 0 && techAlarmState == 0:
		if isDark {
			textColor = color.NRGBA{R: 200, G: 150, B: 210, A: 255}  // rgb(200,150,210)
			rowColor = color.NRGBA{R: 100, G: 15, B: 120, A: 255}    // rgb(100,15,120)
		} else {
			textColor = color.NRGBA{R: 74, G: 10, B: 87, A: 255}     // rgb(74,10,87)
			rowColor = color.NRGBA{R: 170, G: 14, B: 201, A: 255}    // rgb(170,14,201)
		}

	// 9. Є зв'язок, тривога, guardstate=2, немає тех.тривоги
	case isConstate == 1 && alarmstate == 1 && guardstate == 2 && techAlarmState == 0:
		textColor, rowColor = selectEventColor(10) // ArmedOn
		// Перевизначаємо колір тексту
		if isDark {
			textColor = color.NRGBA{R: 150, G: 130, B: 170, A: 255}  // rgb(150,130,170)
		} else {
			textColor = color.NRGBA{R: 28, G: 10, B: 87, A: 255}     // rgb(28,10,87)
		}

	// 10. Є зв'язок, немає тривоги, під охороною, є тех.тривога
	case isConstate >= 0 && alarmstate == 0 && guardstate >= 1 && techAlarmState == 1:
		textColor, rowColor = selectEventColor(2) // Tech alarm
	}

	return textColor, rowColor
}