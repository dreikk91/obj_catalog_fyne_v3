// Package theme містить кастомні теми для додатку АРМ.
// Підтримує темну та світлу теми з кольоровим кодуванням статусів.
package theme

import (
	"image/color"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/theme"
)

// Кольори статусів - однакові для обох тем
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

// ================== ТЕМНА ТЕМА ==================

// DarkTheme - темна тема для нічної роботи диспетчерів
type DarkTheme struct {
	fontSize float32
}

// NewDarkTheme створює нову темну тему з вказаним розміром шрифту
func NewDarkTheme(fontSize float32) fyne.Theme {
	return &DarkTheme{fontSize: fontSize}
}

// Color повертає колір для заданого елемента
func (t *DarkTheme) Color(name fyne.ThemeColorName, variant fyne.ThemeVariant) color.Color {
	switch name {
	case theme.ColorNameBackground:
		return color.NRGBA{R: 28, G: 28, B: 30, A: 255} // #1C1C1E
	case theme.ColorNameButton:
		return color.NRGBA{R: 44, G: 44, B: 46, A: 255} // #2C2C2E
	case theme.ColorNameDisabledButton:
		return color.NRGBA{R: 58, G: 58, B: 60, A: 255}
	case theme.ColorNameForeground:
		return color.NRGBA{R: 255, G: 255, B: 255, A: 255} // Білий текст
	case theme.ColorNamePlaceHolder:
		return color.NRGBA{R: 142, G: 142, B: 147, A: 255}
	case theme.ColorNamePrimary:
		return ColorInfo // Синій акцент
	case theme.ColorNameHover:
		return color.NRGBA{R: 58, G: 58, B: 60, A: 255}
	case theme.ColorNameInputBackground:
		return color.NRGBA{R: 44, G: 44, B: 46, A: 255}
	case theme.ColorNameScrollBar:
		return color.NRGBA{R: 99, G: 99, B: 102, A: 255}
	case theme.ColorNameSeparator:
		return color.NRGBA{R: 58, G: 58, B: 60, A: 255}
	default:
		return theme.DefaultTheme().Color(name, theme.VariantDark)
	}
}

// Font повертає шрифт для стилю тексту
func (t *DarkTheme) Font(style fyne.TextStyle) fyne.Resource {
	return theme.DefaultTheme().Font(style)
}

// Icon повертає іконку за назвою
func (t *DarkTheme) Icon(name fyne.ThemeIconName) fyne.Resource {
	return theme.DefaultTheme().Icon(name)
}

// Size повертає розмір для елемента
func (t *DarkTheme) Size(name fyne.ThemeSizeName) float32 {
	switch name {
	case theme.SizeNameText:
		return t.fontSize
	case theme.SizeNamePadding:
		return 4 // Менші відступи для компактності
	case theme.SizeNameInnerPadding:
		return 6
	default:
		return theme.DefaultTheme().Size(name)
	}
}

// ================== СВІТЛА ТЕМА ==================

// LightTheme - світла тема для денної роботи
type LightTheme struct {
	fontSize float32
}

// NewLightTheme створює нову світлу тему з вказаним розміром шрифту
func NewLightTheme(fontSize float32) fyne.Theme {
	return &LightTheme{fontSize: fontSize}
}

// Color повертає колір для заданого елемента
func (t *LightTheme) Color(name fyne.ThemeColorName, variant fyne.ThemeVariant) color.Color {
	switch name {
	case theme.ColorNameBackground:
		return color.NRGBA{R: 242, G: 242, B: 247, A: 255} // #F2F2F7
	case theme.ColorNameButton:
		return color.NRGBA{R: 255, G: 255, B: 255, A: 255}
	case theme.ColorNameDisabledButton:
		return color.NRGBA{R: 229, G: 229, B: 234, A: 255}
	case theme.ColorNameForeground:
		return color.NRGBA{R: 0, G: 0, B: 0, A: 255} // Чорний текст
	case theme.ColorNamePlaceHolder:
		return color.NRGBA{R: 142, G: 142, B: 147, A: 255}
	case theme.ColorNamePrimary:
		return ColorInfo // Синій акцент
	case theme.ColorNameHover:
		return color.NRGBA{R: 229, G: 229, B: 234, A: 255}
	case theme.ColorNameInputBackground:
		return color.NRGBA{R: 255, G: 255, B: 255, A: 255}
	case theme.ColorNameScrollBar:
		return color.NRGBA{R: 174, G: 174, B: 178, A: 255}
	case theme.ColorNameSeparator:
		return color.NRGBA{R: 199, G: 199, B: 204, A: 255}
	default:
		return theme.DefaultTheme().Color(name, theme.VariantLight)
	}
}

// Font повертає шрифт для стилю тексту
func (t *LightTheme) Font(style fyne.TextStyle) fyne.Resource {
	return theme.DefaultTheme().Font(style)
}

// Icon повертає іконку за назвою
func (t *LightTheme) Icon(name fyne.ThemeIconName) fyne.Resource {
	return theme.DefaultTheme().Icon(name)
}

// Size повертає розмір для елемента
func (t *LightTheme) Size(name fyne.ThemeSizeName) float32 {
	switch name {
	case theme.SizeNameText:
		return t.fontSize
	case theme.SizeNamePadding:
		return 4 // Менші відступи для компактності
	case theme.SizeNameInnerPadding:
		return 6
	default:
		return theme.DefaultTheme().Size(name)
	}
}
