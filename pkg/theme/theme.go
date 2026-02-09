// Package theme містить кольори для статусів.
package theme

import (
	"image/color"
)

// Кольори статусів
var (
	// ColorFire - червоний для пожежі/тривоги
	ColorFire = color.NRGBA{R: 255, G: 59, B: 48, A: 255} // #FF3B30

	// ColorFault - жовтий для несправності
	ColorFault = color.NRGBA{R: 255, G: 204, B: 0, A: 255} // #FFCC00

	// ColorNormal - зелений для норми
	ColorNormal = color.NRGBA{R: 52, G: 199, B: 89, A: 255} // #34C759

	// ColorInfo - синій для інформаційних повідомлень
	ColorInfo = color.NRGBA{R: 0, G: 122, B: 255, A: 255} // #007AFF

	// ColorSelection - колір виділеного рядка
	ColorSelection = color.NRGBA{R: 0, G: 122, B: 255, A: 100} // Напівпрозорий синій
)
