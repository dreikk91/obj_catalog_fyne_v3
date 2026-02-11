package dialogs

import (
	"image/color"

	"obj_catalog_fyne_v3/pkg/utils"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/widget"
)

// ShowColorPaletteDialog відкриває діалог налаштування кольорів подій та об'єктів
// для поточної теми (isDark визначає, темна це чи світла тема).
// onChanged викликається після кожної зміни кольору, щоб оновити UI.
func ShowColorPaletteDialog(win fyne.Window, isDark bool, onChanged func()) {
	themeLabel := "Світла тема"
	if isDark {
		themeLabel = "Темна тема"
	}

	info := widget.NewLabel("Налаштування кольорів подій та об'єктів.\n" +
		"Зміни застосовуються лише до поточної теми: " + themeLabel + ".")

	type eventColorCategory struct {
		Name  string
		Codes []int
	}

	// Логічні категорії, які використовують однакові кольори
	categories := []eventColorCategory{
		{Name: "Тривога", Codes: []int{1}},
		{Name: "Технічна тривога", Codes: []int{2}},
		{Name: "Відновлення / Норма", Codes: []int{5, 9, 13, 17}},
		{Name: "Інформаційні події", Codes: []int{6}},
		{Name: "Під охороною", Codes: []int{7, 8, 10}},
		{Name: "Немає зв'язку", Codes: []int{12}},
		{Name: "Частково знято / інший стан", Codes: []int{14, 18}},
	}

	var buttons []*widget.Button

	for _, cat := range categories {
		category := cat // локальна копія для замикання
		btn := widget.NewButton(category.Name, func() {
			if len(category.Codes) == 0 {
				return
			}

			// Поточний колір фону беремо з першого коду категорії
			currentRow := utils.GetEventRowColor(category.Codes[0], isDark)

			picker := dialog.NewColorPicker(
				"Вибір кольору: "+category.Name,
				"Оберіть колір ФОНУ рядка для цієї категорії.\n"+
					"Текст буде автоматично підібраний із стандартної палітри.",
				func(c color.Color) {
					if c == nil {
						return
					}
					nrgba := color.NRGBAModel.Convert(c).(color.NRGBA)
					for _, code := range category.Codes {
						utils.SetEventRowColor(code, isDark, nrgba)
					}
					if onChanged != nil {
						onChanged()
					}
				},
				win,
			)
			picker.Advanced = true
			picker.SetColor(currentRow)
			picker.Show()
		})

		buttons = append(buttons, btn)
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
		widget.NewLabel("Категорії подій / об'єктів:"),
	)

	for _, b := range buttons {
		content.Add(b)
	}

	content.Add(widget.NewSeparator())
	content.Add(resetBtn)

	d := dialog.NewCustom(
		"Кольори подій та об'єктів",
		"Закрити",
		content,
		win,
	)

	d.Resize(fyne.NewSize(520, 420))
	d.Show()
}
