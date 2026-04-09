package dialogs

import (
	"fmt"
	"image/color"

	"obj_catalog_fyne_v3/pkg/utils"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/widget"
)

type eventColorOption struct {
	Label       string
	Codes       []int
	PreviewCode int
}

func eventColorOptions() []eventColorOption {
	return []eventColorOption{
		{Label: "Тривоги", Codes: []int{1, 21, 22, 23, 24, 25}, PreviewCode: 1},
		{Label: "Несправності / живлення / зв'язок", Codes: []int{2, 3, 4, 12, 26, 27, 29}, PreviewCode: 2},
		{Label: "Відновлення / на зв'язку", Codes: []int{5, 9, 13, 17, 28}, PreviewCode: 5},
		{Label: "Постановка під охорону", Codes: []int{7, 8, 10}, PreviewCode: 10},
		{Label: "Зняття з охорони", Codes: []int{11, 14, 18}, PreviewCode: 11},
		{Label: "Інформація / тест / сервіс", Codes: []int{6, 16, 30}, PreviewCode: 6},
	}
}

// ShowColorPaletteDialog відкриває діалог налаштування кольорів подій та об'єктів
// для поточної теми (isDark визначає, темна це чи світла тема).
// onChanged викликається після кожної зміни кольору, щоб оновити UI.
func ShowColorPaletteDialog(win fyne.Window, isDark bool, onChanged func()) {
	themeLabel := "Світла тема"
	if isDark {
		themeLabel = "Темна тема"
	}

	info := widget.NewLabel(
		"Зміни застосовуються лише до поточної теми: " + themeLabel + ". " +
			"Кольори налаштовуються для семантичних груп подій, а не для кожного SC1 окремо. " +
			"Стани об'єктів на кшталт зняття зі спостереження чи налагодження тут не змінюються.",
	)
	info.Wrapping = fyne.TextWrapWord

	list := container.NewVBox()
	options := eventColorOptions()
	for idx, item := range options {
		option := item
		list.Add(buildEventColorSettingsRow(win, option, isDark, onChanged))
		if idx < len(options)-1 {
			list.Add(widget.NewSeparator())
		}
	}

	resetBtn := widget.NewButton("Скинути кольори поточної теми до стандартних", func() {
		utils.ResetEventColorsToDefault(isDark)
		if onChanged != nil {
			onChanged()
		}
	})

	content := container.NewVBox(
		info,
		widget.NewSeparator(),
		list,
		widget.NewSeparator(),
		resetBtn,
	)

	scroll := container.NewVScroll(content)
	scroll.SetMinSize(fyne.NewSize(720, 480))

	d := dialog.NewCustom(
		"Кольори подій",
		"Закрити",
		scroll,
		win,
	)

	d.Resize(fyne.NewSize(820, 600))
	d.Show()
}

func buildEventColorSettingsRow(win fyne.Window, option eventColorOption, isDark bool, onChanged func()) fyne.CanvasObject {
	code := int64(option.PreviewCode)
	previewCell := newColoredTableCell()
	preview := container.NewGridWrap(fyne.NewSize(140, 30), previewCell)

	refreshPreview := func() {
		updateColoredMessageCell(previewCell, "Зразок", &code, false)
	}
	refreshPreview()

	textBtn := makeIconButton("Текст", iconEdit(), widget.LowImportance, func() {
		currentText := utils.GetEventTextColor(option.PreviewCode, isDark)
		showEventColorPicker(
			win,
			"Колір тексту: "+option.Label,
			"Оберіть колір тексту для цього типу події.",
			currentText,
			func(nrgba color.NRGBA) {
				applyEventTextColor(option.Codes, isDark, nrgba)
				refreshPreview()
				if onChanged != nil {
					onChanged()
				}
			},
		)
	})

	bgBtn := makeIconButton("Фон", iconColors(), widget.LowImportance, func() {
		currentRow := utils.GetEventRowColor(option.PreviewCode, isDark)
		showEventColorPicker(
			win,
			"Колір фону: "+option.Label,
			"Оберіть колір фону рядка для цього типу події.",
			currentRow,
			func(nrgba color.NRGBA) {
				applyEventRowColor(option.Codes, isDark, nrgba)
				refreshPreview()
				if onChanged != nil {
					onChanged()
				}
			},
		)
	})

	title := widget.NewLabelWithStyle(option.Label, fyne.TextAlignLeading, fyne.TextStyle{Bold: true})
	title.Wrapping = fyne.TextWrapOff

	codeLabel := widget.NewLabel(fmt.Sprintf("SC1: %s", formatEventCodes(option.Codes)))
	rightControls := container.NewHBox(preview, textBtn, bgBtn)

	return container.NewBorder(
		nil,
		nil,
		nil,
		rightControls,
		container.NewVBox(title, codeLabel),
	)
}

func applyEventTextColor(codes []int, isDark bool, text color.NRGBA) {
	for _, code := range codes {
		utils.SetEventTextColor(code, isDark, text)
	}
}

func applyEventRowColor(codes []int, isDark bool, row color.NRGBA) {
	for _, code := range codes {
		utils.SetEventRowColor(code, isDark, row)
	}
}

func formatEventCodes(codes []int) string {
	if len(codes) == 0 {
		return "—"
	}
	result := ""
	for idx, code := range codes {
		if idx > 0 {
			result += ", "
		}
		result += fmt.Sprintf("%d", code)
	}
	return result
}
