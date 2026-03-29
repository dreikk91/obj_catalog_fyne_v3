package dialogs

import (
	"fmt"
	"strconv"
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/widget"

	"obj_catalog_fyne_v3/pkg/contracts"
)

func Show220VConstructorDialog(parent fyne.Window, provider contracts.AdminProvider) {
	win := fyne.CurrentApp().NewWindow("Пропажа / відновлення 220В")
	win.Resize(fyne.NewSize(980, 700))

	var (
		freeMessages    []contracts.AdminMessage
		alarmMessages   []contracts.AdminMessage
		restoreMessages []contracts.AdminMessage
		selectedFree    = -1
		selectedAlarm   = -1
		selectedRestore = -1
	)

	statusLabel := widget.NewLabel("Готово")

	filterEntry := widget.NewEntry()
	filterEntry.SetPlaceHolder("Фільтр (код / hex / текст)")

	chkMost := widget.NewCheck("Мост", nil)
	chkMost.SetChecked(true)
	chkCID := widget.NewCheck("Contact ID", nil)
	chkCID.SetChecked(true)
	chk20BPS := widget.NewCheck("20BPS / Ademco-Express", nil)
	chk20BPS.SetChecked(true)

	messageDisplay := func(m contracts.AdminMessage) string {
		code := strconv.FormatInt(m.UIN, 10)
		if m.MessageID != nil {
			code = strconv.FormatInt(*m.MessageID, 10)
		}
		text := strings.TrimSpace(m.Text)
		if text == "" {
			text = "(без тексту)"
		}
		proto := "—"
		if m.ProtocolID != nil {
			proto = protocolDisplayName(*m.ProtocolID)
		}
		return fmt.Sprintf("%s  %s  [%s]", code, text, proto)
	}

	selectedProtocols := func() []int64 {
		out := make([]int64, 0, 3)
		if chkCID.Checked {
			out = append(out, 18)
		}
		if chk20BPS.Checked {
			out = append(out, 3)
		}
		if chkMost.Checked {
			out = append(out, 4)
		}
		return out
	}

	var freeList *widget.List
	var alarmList *widget.List
	var restoreList *widget.List

	freeList = widget.NewList(
		func() int { return len(freeMessages) },
		func() fyne.CanvasObject { return widget.NewLabel("message") },
		func(id widget.ListItemID, obj fyne.CanvasObject) {
			label := obj.(*widget.Label)
			if id < 0 || id >= len(freeMessages) {
				label.SetText("")
				return
			}
			label.SetText(messageDisplay(freeMessages[id]))
		},
	)
	alarmList = widget.NewList(
		func() int { return len(alarmMessages) },
		func() fyne.CanvasObject { return widget.NewLabel("message") },
		func(id widget.ListItemID, obj fyne.CanvasObject) {
			label := obj.(*widget.Label)
			if id < 0 || id >= len(alarmMessages) {
				label.SetText("")
				return
			}
			label.SetText(messageDisplay(alarmMessages[id]))
		},
	)
	restoreList = widget.NewList(
		func() int { return len(restoreMessages) },
		func() fyne.CanvasObject { return widget.NewLabel("message") },
		func(id widget.ListItemID, obj fyne.CanvasObject) {
			label := obj.(*widget.Label)
			if id < 0 || id >= len(restoreMessages) {
				label.SetText("")
				return
			}
			label.SetText(messageDisplay(restoreMessages[id]))
		},
	)

	freeList.OnSelected = func(id widget.ListItemID) {
		selectedFree = id
		selectedAlarm = -1
		selectedRestore = -1
		alarmList.UnselectAll()
		restoreList.UnselectAll()
	}
	alarmList.OnSelected = func(id widget.ListItemID) {
		selectedAlarm = id
		selectedFree = -1
		selectedRestore = -1
		freeList.UnselectAll()
		restoreList.UnselectAll()
	}
	restoreList.OnSelected = func(id widget.ListItemID) {
		selectedRestore = id
		selectedFree = -1
		selectedAlarm = -1
		freeList.UnselectAll()
		alarmList.UnselectAll()
	}

	reload := func(preferredBucket string, preferredUIN int64) {
		protocolIDs := selectedProtocols()
		if len(protocolIDs) == 0 {
			freeMessages = freeMessages[:0]
			alarmMessages = alarmMessages[:0]
			restoreMessages = restoreMessages[:0]
			selectedFree, selectedAlarm, selectedRestore = -1, -1, -1
			freeList.UnselectAll()
			alarmList.UnselectAll()
			restoreList.UnselectAll()
			freeList.Refresh()
			alarmList.Refresh()
			restoreList.Refresh()
			statusLabel.SetText("Оберіть хоча б один протокол для перегляду")
			return
		}

		buckets, err := provider.List220VMessageBuckets(protocolIDs, strings.TrimSpace(filterEntry.Text))
		if err != nil {
			dialog.ShowError(err, win)
			statusLabel.SetText("Помилка завантаження 220В-повідомлень")
			return
		}

		freeMessages = buckets.Free
		alarmMessages = buckets.Alarm
		restoreMessages = buckets.Restore
		selectedFree, selectedAlarm, selectedRestore = -1, -1, -1
		freeList.UnselectAll()
		alarmList.UnselectAll()
		restoreList.UnselectAll()
		freeList.Refresh()
		alarmList.Refresh()
		restoreList.Refresh()

		if preferredUIN > 0 {
			switch preferredBucket {
			case "free":
				for i := range freeMessages {
					if freeMessages[i].UIN == preferredUIN {
						freeList.Select(i)
						break
					}
				}
			case "alarm":
				for i := range alarmMessages {
					if alarmMessages[i].UIN == preferredUIN {
						alarmList.Select(i)
						break
					}
				}
			case "restore":
				for i := range restoreMessages {
					if restoreMessages[i].UIN == preferredUIN {
						restoreList.Select(i)
						break
					}
				}
			}
		}

		statusLabel.SetText(fmt.Sprintf(
			"Завантажено: вільні %d, пропажа 220В %d, відновлення 220В %d",
			len(freeMessages), len(alarmMessages), len(restoreMessages),
		))
	}

	toAlarmBtn := widget.NewButton(">", func() {
		if selectedFree < 0 || selectedFree >= len(freeMessages) {
			statusLabel.SetText("Виберіть повідомлення у списку «Всі вільні повідомлення»")
			return
		}
		msg := freeMessages[selectedFree]
		if err := provider.SetMessage220VMode(msg.UIN, contracts.Admin220VAlarm); err != nil {
			dialog.ShowError(err, win)
			statusLabel.SetText("Не вдалося додати повідомлення до «Пропажа 220В»")
			return
		}
		reload("alarm", msg.UIN)
	})

	fromAlarmBtn := widget.NewButton("<", func() {
		if selectedAlarm < 0 || selectedAlarm >= len(alarmMessages) {
			statusLabel.SetText("Виберіть повідомлення у списку «Пропажа 220В»")
			return
		}
		msg := alarmMessages[selectedAlarm]
		if err := provider.SetMessage220VMode(msg.UIN, contracts.Admin220VNone); err != nil {
			dialog.ShowError(err, win)
			statusLabel.SetText("Не вдалося прибрати повідомлення з «Пропажа 220В»")
			return
		}
		reload("free", msg.UIN)
	})

	toRestoreBtn := widget.NewButton(">", func() {
		if selectedFree < 0 || selectedFree >= len(freeMessages) {
			statusLabel.SetText("Виберіть повідомлення у списку «Всі вільні повідомлення»")
			return
		}
		msg := freeMessages[selectedFree]
		if err := provider.SetMessage220VMode(msg.UIN, contracts.Admin220VRestore); err != nil {
			dialog.ShowError(err, win)
			statusLabel.SetText("Не вдалося додати повідомлення до «Відновлення 220В»")
			return
		}
		reload("restore", msg.UIN)
	})

	fromRestoreBtn := widget.NewButton("<", func() {
		if selectedRestore < 0 || selectedRestore >= len(restoreMessages) {
			statusLabel.SetText("Виберіть повідомлення у списку «Відновлення 220В»")
			return
		}
		msg := restoreMessages[selectedRestore]
		if err := provider.SetMessage220VMode(msg.UIN, contracts.Admin220VNone); err != nil {
			dialog.ShowError(err, win)
			statusLabel.SetText("Не вдалося прибрати повідомлення з «Відновлення 220В»")
			return
		}
		reload("free", msg.UIN)
	})

	closeBtn := widget.NewButton("Закрити", func() { win.Close() })
	refreshBtn := widget.NewButton("Оновити", func() { reload("", 0) })

	filterEntry.OnChanged = func(_ string) { reload("", 0) }
	chkMost.OnChanged = func(_ bool) { reload("", 0) }
	chkCID.OnChanged = func(_ bool) { reload("", 0) }
	chk20BPS.OnChanged = func(_ bool) { reload("", 0) }

	freeCard := widget.NewCard("Всі вільні повідомлення", "", freeList)
	alarmCard := widget.NewCard("Пропажа 220В", "", alarmList)
	restoreCard := widget.NewCard("Відновлення 220В", "", restoreList)

	rightColumn := container.NewGridWithRows(2, alarmCard, restoreCard)
	arrowColumn := container.NewVBox(
		layout.NewSpacer(),
		toAlarmBtn,
		fromAlarmBtn,
		layout.NewSpacer(),
		toRestoreBtn,
		fromRestoreBtn,
		layout.NewSpacer(),
	)

	center := container.NewBorder(
		nil,
		nil,
		nil,
		nil,
		container.NewHBox(
			container.NewGridWrap(fyne.NewSize(560, 520), freeCard),
			container.NewPadded(arrowColumn),
			container.NewGridWrap(fyne.NewSize(320, 520), rightColumn),
		),
	)

	filterBox := widget.NewCard(
		"Фільтрація",
		"",
		container.NewVBox(
			filterEntry,
			container.NewHBox(chkMost, chkCID, chk20BPS),
		),
	)

	content := container.NewBorder(
		nil,
		container.NewVBox(
			filterBox,
			container.NewHBox(statusLabel, layout.NewSpacer(), refreshBtn, closeBtn),
		),
		nil, nil,
		center,
	)

	win.SetContent(content)
	reload("", 0)
	win.Show()
}
