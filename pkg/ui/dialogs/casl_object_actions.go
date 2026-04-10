package dialogs

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/widget"

	"obj_catalog_fyne_v3/pkg/contracts"
)

const caslEndlessBlockUnix = int64(2554790050)

func ShowCASLObjectDeleteDialog(parent fyne.Window, provider contracts.CASLObjectEditorProvider, objectID int64, onSuccess func()) {
	if parent == nil || provider == nil {
		return
	}
	if objectID <= 0 {
		ShowInfoDialog(parent, "Об'єкт не вибрано", "Виберіть CASL-об'єкт у списку.")
		return
	}

	loadCASLObjectSnapshot(parent, provider, objectID, func(snapshot contracts.CASLObjectEditorSnapshot) {
		name := firstNonEmpty(strings.TrimSpace(snapshot.Object.Name), "Без назви")
		objID := firstNonEmpty(strings.TrimSpace(snapshot.Object.ObjID), strconv.FormatInt(objectID, 10))
		message := fmt.Sprintf(
			"Видалити об'єкт \"%s\" [obj_id=%s]?\n\nПеред видаленням об'єкт буде збережений у корзину CASL.",
			name,
			objID,
		)
		if strings.TrimSpace(snapshot.Object.Device.DeviceID) != "" {
			message += "\n\nУ об'єкта є прив'язаний прилад. Видалення виконується так само, як в оригінальному CASL: save_in_basket + delete_grd_object."
		}

		dialog.ShowConfirm("Видалення об'єкта CASL", message, func(confirmed bool) {
			if !confirmed {
				return
			}
			go func() {
				ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
				defer cancel()

				err := provider.DeleteCASLObject(ctx, objectID)
				fyne.Do(func() {
					if err != nil {
						dialog.ShowError(err, parent)
						return
					}
					if onSuccess != nil {
						onSuccess()
					}
				})
			}()
		}, parent)
	})
}

func ShowCASLObjectBasketDialog(parent fyne.Window, provider contracts.CASLObjectEditorProvider) {
	if parent == nil || provider == nil {
		return
	}

	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
		defer cancel()

		items, err := provider.ReadCASLObjectBasket(ctx)
		fyne.Do(func() {
			if err != nil {
				dialog.ShowError(err, parent)
				return
			}

			if len(items) == 0 {
				ShowInfoDialog(parent, "Корзина CASL", "У корзині немає видалених об'єктів.")
				return
			}

			rows := make([]string, 0, len(items))
			for _, item := range items {
				name := firstNonEmpty(strings.TrimSpace(item.Name), "Без назви")
				address := strings.TrimSpace(item.Address)
				line := fmt.Sprintf("#%d | %s", item.BasketID, name)
				if address != "" {
					line += " | " + address
				}
				if item.ObjID != "" {
					line += " | obj_id=" + item.ObjID
				}
				if item.DeletedRaw != "" {
					line += " | " + item.DeletedRaw
				}
				rows = append(rows, line)
			}

			content := widget.NewMultiLineEntry()
			content.SetText(strings.Join(rows, "\n"))
			content.Disable()
			visibleRows := len(rows) + 1
			if visibleRows > 14 {
				visibleRows = 14
			}
			content.SetMinRowsVisible(visibleRows)

			dialog.ShowCustom("Корзина CASL", "Закрити", container.NewBorder(
				widget.NewLabel("Видалені об'єкти збережені в корзину CASL."),
				nil, nil, nil,
				content,
			), parent)
		})
	}()
}

func ShowCASLObjectBlockDialog(parent fyne.Window, provider contracts.CASLObjectEditorProvider, objectID int64, onSuccess func()) {
	if parent == nil || provider == nil {
		return
	}
	if objectID <= 0 {
		ShowInfoDialog(parent, "Об'єкт не вибрано", "Виберіть CASL-об'єкт у списку.")
		return
	}

	loadCASLObjectSnapshot(parent, provider, objectID, func(snapshot contracts.CASLObjectEditorSnapshot) {
		if strings.TrimSpace(snapshot.Object.Device.DeviceID) == "" {
			ShowInfoDialog(parent, "Недоступно", "У вибраного об'єкта немає прив'язаного приладу CASL.")
			return
		}
		if snapshot.Object.DeviceBlocked {
			showCASLObjectUnblockDialog(parent, provider, snapshot, onSuccess)
			return
		}
		showCASLObjectBlockFormDialog(parent, provider, snapshot, onSuccess)
	})
}

func loadCASLObjectSnapshot(parent fyne.Window, provider contracts.CASLObjectEditorProvider, objectID int64, onLoaded func(contracts.CASLObjectEditorSnapshot)) {
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
		defer cancel()

		snapshot, err := provider.GetCASLObjectEditorSnapshot(ctx, objectID)
		fyne.Do(func() {
			if err != nil {
				dialog.ShowError(err, parent)
				return
			}
			if onLoaded != nil {
				onLoaded(snapshot)
			}
		})
	}()
}

func showCASLObjectUnblockDialog(parent fyne.Window, provider contracts.CASLObjectEditorProvider, snapshot contracts.CASLObjectEditorSnapshot, onSuccess func()) {
	objectLabel := caslActionObjectLabel(snapshot.Object)
	until := formatCASLDialogBlockedUntil(snapshot.Object.TimeUnblock)
	reason := strings.TrimSpace(snapshot.Object.BlockMessage)

	info := []fyne.CanvasObject{
		widget.NewLabel(fmt.Sprintf("Об'єкт %s зараз заблокований так само, як у CASL через DEVICE_BLOCK.", objectLabel)),
	}
	if reason != "" {
		info = append(info, widget.NewLabel("Причина: "+reason))
	}
	if until != "" {
		info = append(info, widget.NewLabel("До: "+until))
	}

	statusLabel := widget.NewLabel("")
	var actionBtn *widget.Button
	actionBtn = widget.NewButton("Розблокувати об'єкт", func() {
		statusLabel.SetText("Розблокування...")
		actionBtn.Disable()

		go func(deviceID string) {
			ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
			defer cancel()

			err := provider.UnblockCASLDevice(ctx, deviceID)
			fyne.Do(func() {
				if err != nil {
					statusLabel.SetText("Помилка")
					actionBtn.Enable()
					dialog.ShowError(err, parent)
					return
				}
				if onSuccess != nil {
					onSuccess()
				}
			})
		}(snapshot.Object.Device.DeviceID)
	})

	content := container.NewVBox(
		container.NewVBox(info...),
		widget.NewSeparator(),
		statusLabel,
		container.NewHBox(layout.NewSpacer(), actionBtn),
	)
	dialog.ShowCustom("Розблокування об'єкта CASL", "Закрити", content, parent)
}

func showCASLObjectBlockFormDialog(parent fyne.Window, provider contracts.CASLObjectEditorProvider, snapshot contracts.CASLObjectEditorSnapshot, onSuccess func()) {
	objectLabel := caslActionObjectLabel(snapshot.Object)
	hoursEntry := widget.NewEntry()
	hoursEntry.SetText("00")
	minutesEntry := widget.NewEntry()
	minutesEntry.SetText("30")
	reasonEntry := widget.NewMultiLineEntry()
	reasonEntry.SetMinRowsVisible(3)
	endlessCheck := widget.NewCheck("Безстроково", nil)
	timeHint := widget.NewLabel("")
	reasonHint := widget.NewLabel("")
	statusLabel := widget.NewLabel("")

	setHints := func(err error) {
		timeHint.SetText("")
		reasonHint.SetText("")
		statusLabel.SetText("")
		if err == nil {
			return
		}
		msg := err.Error()
		switch {
		case strings.Contains(msg, "причина"):
			reasonHint.SetText(msg)
		default:
			timeHint.SetText(msg)
		}
	}

	canSubmit := func() bool {
		if _, err := buildCASLObjectBlockRequest(snapshot, hoursEntry.Text, minutesEntry.Text, reasonEntry.Text, endlessCheck.Checked); err != nil {
			setHints(err)
			return false
		}
		setHints(nil)
		return true
	}

	var submitBtn *widget.Button
	refreshState := func() {
		if canSubmit() {
			submitBtn.Enable()
		} else {
			submitBtn.Disable()
		}
		if endlessCheck.Checked {
			hoursEntry.Disable()
			minutesEntry.Disable()
			return
		}
		hoursEntry.Enable()
		minutesEntry.Enable()
	}

	hoursEntry.OnChanged = func(string) { refreshState() }
	minutesEntry.OnChanged = func(string) { refreshState() }
	reasonEntry.OnChanged = func(string) { refreshState() }
	endlessCheck.OnChanged = func(bool) { refreshState() }

	submitBtn = widget.NewButton("Заблокувати об'єкт", func() {
		request, err := buildCASLObjectBlockRequest(snapshot, hoursEntry.Text, minutesEntry.Text, reasonEntry.Text, endlessCheck.Checked)
		if err != nil {
			setHints(err)
			refreshState()
			return
		}

		statusLabel.SetText("Блокування...")
		submitBtn.Disable()
		hoursEntry.Disable()
		minutesEntry.Disable()
		reasonEntry.Disable()
		endlessCheck.Disable()

		go func() {
			ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
			defer cancel()

			err := provider.BlockCASLDevice(ctx, request)
			fyne.Do(func() {
				if err != nil {
					statusLabel.SetText("Помилка")
					dialog.ShowError(err, parent)
					endlessCheck.Enable()
					reasonEntry.Enable()
					refreshState()
					return
				}
				if onSuccess != nil {
					onSuccess()
				}
			})
		}()
	})

	content := container.NewVBox(
		widget.NewLabel(fmt.Sprintf("Блокування об'єкта %s виконується так само, як у CASL: через DEVICE_BLOCK для ППК №%d.", objectLabel, snapshot.Object.Device.Number)),
		widget.NewLabel("Заблокований ППК не має потрапляти в стрічку подій, поки блокування активне."),
		widget.NewForm(
			widget.NewFormItem("Години", hoursEntry),
			widget.NewFormItem("Хвилини", minutesEntry),
			widget.NewFormItem("", endlessCheck),
			widget.NewFormItem("Причина", reasonEntry),
		),
		timeHint,
		reasonHint,
		statusLabel,
		container.NewHBox(layout.NewSpacer(), submitBtn),
	)

	refreshState()
	dialog.ShowCustom("Блокування об'єкта CASL", "Закрити", content, parent)
}

func buildCASLObjectBlockRequest(snapshot contracts.CASLObjectEditorSnapshot, hoursRaw string, minutesRaw string, reasonRaw string, endless bool) (contracts.CASLDeviceBlockRequest, error) {
	reason := strings.TrimSpace(reasonRaw)
	if utf8.RuneCountInString(reason) < 3 {
		return contracts.CASLDeviceBlockRequest{}, fmt.Errorf("причина блокування має містити щонайменше 3 символи")
	}

	until := caslEndlessBlockUnix
	if !endless {
		hours, err := parseCASLEditorInt(hoursRaw)
		if err != nil || hours < 0 || hours > 24 {
			return contracts.CASLDeviceBlockRequest{}, fmt.Errorf("години блокування мають бути в межах 0..24")
		}
		minutes, err := parseCASLEditorInt(minutesRaw)
		if err != nil || minutes < 0 || minutes > 59 {
			return contracts.CASLDeviceBlockRequest{}, fmt.Errorf("хвилини блокування мають бути в межах 0..59")
		}
		until = time.Now().Unix() + int64(hours*3600+minutes*60)
	}

	return contracts.CASLDeviceBlockRequest{
		DeviceID:     strings.TrimSpace(snapshot.Object.Device.DeviceID),
		DeviceNumber: snapshot.Object.Device.Number,
		TimeUnblock:  until,
		Message:      reason,
	}, nil
}

func caslActionObjectLabel(object contracts.CASLGuardObjectDetails) string {
	name := firstNonEmpty(strings.TrimSpace(object.Name), "Без назви")
	objID := firstNonEmpty(strings.TrimSpace(object.ObjID), "n/a")
	return fmt.Sprintf("\"%s\" [obj_id=%s]", name, objID)
}

func formatCASLDialogBlockedUntil(unixTS int64) string {
	if unixTS <= 0 {
		return ""
	}
	if unixTS >= caslEndlessBlockUnix {
		return "безстроково"
	}
	return time.Unix(unixTS, 0).Local().Format("02.01.2006 15:04")
}
