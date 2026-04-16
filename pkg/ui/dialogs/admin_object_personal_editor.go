package dialogs

import (
	"fmt"
	"strconv"
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/widget"

	"obj_catalog_fyne_v3/pkg/contracts"
)

func showObjectPersonalEditor(
	parent fyne.Window,
	provider contracts.AdminObjectPersonalService,
	title string,
	initial contracts.AdminObjectPersonal,
	onSave func(item contracts.AdminObjectPersonal) error,
	statusLabel *widget.Label,
	onDone func(),
) {
	numberEntry := widget.NewEntry()
	if initial.Number > 0 {
		numberEntry.SetText(strconv.FormatInt(initial.Number, 10))
	}
	numberEntry.SetPlaceHolder("1..999")
	surnameEntry := widget.NewEntry()
	surnameEntry.SetText(initial.Surname)
	nameEntry := widget.NewEntry()
	nameEntry.SetText(initial.Name)
	secNameEntry := widget.NewEntry()
	secNameEntry.SetText(initial.SecName)
	addressEntry := widget.NewEntry()
	addressEntry.SetText(initial.Address)
	phonesEntry := widget.NewEntry()
	phonesEntry.SetText(initial.Phones)
	phoneLookupLabel := widget.NewLabel("")
	phoneLookupLabel.Wrapping = fyne.TextWrapWord
	positionEntry := widget.NewEntry()
	positionEntry.SetText(initial.Position)
	notesEntry := widget.NewEntry()
	notesEntry.SetText(initial.Notes)
	isRangCheck := widget.NewCheck("ISRANG (старший/ранг)", nil)
	isRangCheck.SetChecked(initial.IsRang)
	if initial.ID == 0 {
		isRangCheck.SetChecked(true)
	}
	accessCheck := widget.NewCheck("Повний доступ до адмін-функцій (ACCESS1=1)", nil)
	accessCheck.SetChecked(initial.Access1 > 0)
	viberIDEntry := widget.NewEntry()
	viberIDEntry.SetText(initial.ViberID)
	viberIDEntry.SetPlaceHolder("Viber ID (необов'язково)")
	telegramIDEntry := widget.NewEntry()
	telegramIDEntry.SetText(initial.TelegramID)
	telegramIDEntry.SetPlaceHolder("Telegram ID (необов'язково)")
	createdAtLabel := widget.NewLabel(initial.CreatedAt)
	if strings.TrimSpace(initial.CreatedAt) == "" {
		createdAtLabel.SetText("буде встановлено автоматично")
	}
	trkCheck := widget.NewCheck("Перевіряючий ТРК", nil)
	trkCheck.SetChecked(initial.IsTRKTester)

	digitsCount := func(s string) int {
		cnt := 0
		for _, r := range s {
			if r >= '0' && r <= '9' {
				cnt++
			}
		}
		return cnt
	}

	applyPersonalLookup := func(found *contracts.AdminObjectPersonal) {
		if found == nil {
			return
		}
		if strings.TrimSpace(numberEntry.Text) == "" && found.Number > 0 {
			numberEntry.SetText(strconv.FormatInt(found.Number, 10))
		}
		surnameEntry.SetText(strings.TrimSpace(found.Surname))
		nameEntry.SetText(strings.TrimSpace(found.Name))
		secNameEntry.SetText(strings.TrimSpace(found.SecName))
		addressEntry.SetText(strings.TrimSpace(found.Address))
		positionEntry.SetText(strings.TrimSpace(found.Position))
		notesEntry.SetText(strings.TrimSpace(found.Notes))
		isRangCheck.SetChecked(found.IsRang)
		accessCheck.SetChecked(found.Access1 > 0)
		viberIDEntry.SetText(strings.TrimSpace(found.ViberID))
		telegramIDEntry.SetText(strings.TrimSpace(found.TelegramID))
		trkCheck.SetChecked(found.IsTRKTester)
		if strings.TrimSpace(createdAtLabel.Text) == "" || createdAtLabel.Text == "буде встановлено автоматично" {
			if strings.TrimSpace(found.CreatedAt) != "" {
				createdAtLabel.SetText(found.CreatedAt)
			}
		}

		source := "Знайдено контакт у базі, дані підтягнуто автоматично"
		if found.SourceObjN > 0 {
			source = fmt.Sprintf("Знайдено контакт у базі (об'єкт #%d), дані підтягнуто автоматично", found.SourceObjN)
		}
		phoneLookupLabel.SetText(source)
	}

	lastPhoneLookupRaw := ""
	tryLookupByPhone := func(raw string) {
		raw = strings.TrimSpace(raw)
		if raw == "" {
			lastPhoneLookupRaw = ""
			phoneLookupLabel.SetText("")
			return
		}
		if digitsCount(raw) < 10 {
			phoneLookupLabel.SetText("")
			return
		}
		if raw == lastPhoneLookupRaw {
			return
		}
		lastPhoneLookupRaw = raw

		found, err := provider.FindPersonalByPhone(raw)
		if err != nil {
			phoneLookupLabel.SetText("Не вдалося перевірити телефон у базі")
			return
		}
		if found == nil {
			phoneLookupLabel.SetText("")
			return
		}
		applyPersonalLookup(found)
	}
	phonesEntry.OnChanged = func(text string) {
		tryLookupByPhone(text)
	}
	phonesEntry.OnSubmitted = func(text string) {
		tryLookupByPhone(text)
	}

	form := widget.NewForm(
		widget.NewFormItem("№:", numberEntry),
		widget.NewFormItem("Створено:", createdAtLabel),
		widget.NewFormItem("Прізвище:", surnameEntry),
		widget.NewFormItem("Ім'я:", nameEntry),
		widget.NewFormItem("По батькові:", secNameEntry),
		widget.NewFormItem("Адреса:", addressEntry),
		widget.NewFormItem("Телефон:", container.NewVBox(phonesEntry, phoneLookupLabel)),
		widget.NewFormItem("Посада:", positionEntry),
		widget.NewFormItem("Примітка:", notesEntry),
		widget.NewFormItem("", isRangCheck),
		widget.NewFormItem("", accessCheck),
		widget.NewFormItem("Viber ID:", viberIDEntry),
		widget.NewFormItem("Telegram ID:", telegramIDEntry),
		widget.NewFormItem("", trkCheck),
	)

	dlg := dialog.NewCustomConfirm(title, "Зберегти", "Відміна", form, func(ok bool) {
		if !ok {
			return
		}

		numRaw := strings.TrimSpace(numberEntry.Text)
		number := int64(0)
		if numRaw != "" {
			n, err := strconv.ParseInt(numRaw, 10, 64)
			if err != nil {
				statusLabel.SetText("Некоректний номер В/О")
				return
			}
			number = n
		}

		item := contracts.AdminObjectPersonal{
			Number:      number,
			Surname:     strings.TrimSpace(surnameEntry.Text),
			Name:        strings.TrimSpace(nameEntry.Text),
			SecName:     strings.TrimSpace(secNameEntry.Text),
			Address:     strings.TrimSpace(addressEntry.Text),
			Phones:      strings.TrimSpace(phonesEntry.Text),
			Position:    strings.TrimSpace(positionEntry.Text),
			Notes:       strings.TrimSpace(notesEntry.Text),
			IsRang:      isRangCheck.Checked,
			Access1:     boolToInt64(accessCheck.Checked),
			ViberID:     strings.TrimSpace(viberIDEntry.Text),
			TelegramID:  strings.TrimSpace(telegramIDEntry.Text),
			CreatedAt:   strings.TrimSpace(createdAtLabel.Text),
			IsTRKTester: trkCheck.Checked,
		}
		if err := onSave(item); err != nil {
			dialog.ShowError(err, parent)
			statusLabel.SetText("Не вдалося зберегти В/О")
			return
		}
		onDone()
	}, parent)
	dlg.Show()
}
