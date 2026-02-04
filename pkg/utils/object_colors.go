package utils

import (
	"image/color"
)

// ChangeItemColorNRGBA визначає кольори елемента на основі стану об'єкта у форматі NRGBA
func ChangeItemColorNRGBA(alarmstate, guardstate, techAlarmState, isConstate int64, isDark bool) (color.NRGBA, color.NRGBA) {
	// Функція вибору кольору події
	selectEventColor := SelectColorNRGBA
	if isDark {
		selectEventColor = SelectColorNRGBADark
	}

	// Дефолтні кольори
	textColor, rowColor := selectEventColor(-1) // беремо дефолтні для цієї теми

	switch {
	// 1. Тривога під охороною (на зв'язку чи ні)
	case (isConstate == 1 || isConstate == 0) && alarmstate == 1 && guardstate == 1 && techAlarmState == 0:
		textColor, rowColor = selectEventColor(1)

	// 2. Охорона (на зв'язку)
	case isConstate == 1 && alarmstate == 0 && (guardstate == 1 || guardstate == 2) && techAlarmState == 0:
		textColor, rowColor = selectEventColor(10)

	// 3. Знято з охорони (на зв'язку або без)
	case (isConstate == 1 || isConstate == 0) && alarmstate == 0 && guardstate == 0 && techAlarmState == 0:
		if isDark {
			textColor, rowColor = selectEventColor(13) // Світла подія для фонового заповнення
		} else {
			textColor = color.NRGBA{R: 255, G: 255, B: 255, A: 255}
			rowColor = color.NRGBA{R: 170, G: 14, B: 201, A: 255}
		}

	// 4. Охорона (немає зв'язку)
	case isConstate == 0 && alarmstate == 0 && guardstate == 1 && techAlarmState == 0:
		if isDark {
			textColor, rowColor = selectEventColor(12)
		} else {
			textColor = color.NRGBA{R: 40, G: 40, B: 40, A: 255}
			rowColor = color.NRGBA{R: 255, G: 230, B: 140, A: 255}
		}

	// 5. Тривога без охорони (немає зв'язку)
	case isConstate == 0 && alarmstate == 1 && guardstate == 0 && techAlarmState == 0:
		textColor, rowColor = selectEventColor(1) // Alarm

	// 6. Спеціальний випадок: Охорона-2 з тривогою
	case isConstate == 1 && alarmstate == 1 && guardstate == 2 && techAlarmState == 0:
		textColor, rowColor = selectEventColor(10)
		if !isDark {
			textColor = color.NRGBA{R: 28, G: 10, B: 87, A: 255}
		}

	// 7. Технічна тривога
	case isConstate == 1 && alarmstate == 0 && guardstate == 1 && techAlarmState == 1:
		textColor, rowColor = selectEventColor(2)
	}

	return textColor, rowColor
}
