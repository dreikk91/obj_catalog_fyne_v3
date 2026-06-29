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
	ColorFire = color.NRGBA{R: 198, G: 40, B: 40, A: 255} // #C62828

	// ColorFault - жовтий для несправності
	ColorFault = color.NRGBA{R: 255, G: 143, B: 0, A: 255} // #FF8F00

	// ColorNormal - зелений для норми
	ColorNormal = color.NRGBA{R: 61, G: 156, B: 59, A: 255} // #3D9C3B

	// ColorInfo - синій для інформаційних повідомлень
	ColorInfo = color.NRGBA{R: 69, G: 133, B: 188, A: 255} // #4585BC

	// ColorSelection - колір виділеного рядка
	ColorSelection = ColorInfo

	// Семантичні аліаси для UI-компонентів
	ColorDanger       = ColorFire                                  // Деструктивні дії (видалення тощо)
	ColorSuccess      = ColorNormal                                // Успіх / норма
	ColorWarning      = ColorFault                                 // Попередження / несправність
	ColorSectionTitle = color.NRGBA{R: 96, G: 125, B: 139, A: 255} // #607D8B
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
		return color.NRGBA{R: 43, G: 49, B: 69, A: 255} // #2B3145
	case theme.ColorNameButton:
		return color.NRGBA{R: 37, G: 43, B: 62, A: 255} // #252B3E
	case theme.ColorNameDisabledButton:
		return color.NRGBA{R: 61, G: 68, B: 89, A: 255} // #3D4459
	case theme.ColorNameForeground:
		return color.NRGBA{R: 225, G: 225, B: 225, A: 255} // #E1E1E1
	case theme.ColorNamePlaceHolder:
		return color.NRGBA{R: 178, G: 192, B: 206, A: 255} // #B2C0CE
	case theme.ColorNamePrimary:
		return ColorInfo // Синій акцент
	case theme.ColorNameHover:
		return color.NRGBA{R: 61, G: 68, B: 89, A: 255}
	case theme.ColorNameInputBackground:
		return color.NRGBA{R: 37, G: 43, B: 62, A: 255}
	case theme.ColorNameScrollBar:
		return color.NRGBA{R: 91, G: 99, B: 123, A: 255} // #5B637B
	case theme.ColorNameSeparator:
		return color.NRGBA{R: 61, G: 68, B: 89, A: 255}
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
	case theme.SizeNameLineSpacing:
		return 2 // Міжрядковий інтервал (менше = компактніше)
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
		return color.NRGBA{R: 248, G: 249, B: 250, A: 255} // #F8F9FA
	case theme.ColorNameButton:
		return color.NRGBA{R: 255, G: 255, B: 255, A: 255}
	case theme.ColorNameDisabledButton:
		return color.NRGBA{R: 226, G: 230, B: 234, A: 255} // #E2E6EA
	case theme.ColorNameForeground:
		return color.NRGBA{R: 33, G: 37, B: 41, A: 255} // #212529
	case theme.ColorNamePlaceHolder:
		return color.NRGBA{R: 96, G: 125, B: 139, A: 255} // #607D8B
	case theme.ColorNamePrimary:
		return ColorInfo // Синій акцент
	case theme.ColorNameHover:
		return color.NRGBA{R: 239, G: 243, B: 247, A: 255} // #EFF3F7
	case theme.ColorNameInputBackground:
		return color.NRGBA{R: 255, G: 255, B: 255, A: 255}
	case theme.ColorNameScrollBar:
		return color.NRGBA{R: 178, G: 192, B: 206, A: 255} // #B2C0CE
	case theme.ColorNameSeparator:
		return color.NRGBA{R: 177, G: 191, B: 205, A: 255} // #B1BFCD
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
