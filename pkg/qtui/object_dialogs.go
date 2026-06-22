//go:build qt

package qtui

import (
	"fmt"
	"strconv"
	"strings"

	qt "github.com/mappu/miqt/qt6"

	"obj_catalog_fyne_v3/pkg/contracts"
	"obj_catalog_fyne_v3/pkg/models"
	"obj_catalog_fyne_v3/pkg/simoperator"
	"obj_catalog_fyne_v3/pkg/ui/viewmodels"
)

func ShowObjectEditDialog(parent *qt.QWidget, card contracts.AdminObjectCard) (contracts.AdminObjectCard, bool) {
	dialog := qt.NewQDialog(parent)
	dialog.SetWindowTitle("Редагування об'єкта")
	dialog.Resize(720, 560)

	layout := qt.NewQVBoxLayout(dialog.QWidget)
	form := qt.NewQFormLayout2()
	form.SetFieldGrowthPolicy(qt.QFormLayout__AllNonFixedFieldsGrow)

	shortName := newLineEdit(card.ShortName)
	fullName := newLineEdit(card.FullName)
	address := newLineEdit(card.Address)
	phones := newLineEdit(card.Phones)
	contract := newLineEdit(card.Contract)
	location := newLineEdit(card.Location)
	notes := qt.NewQTextEdit3(card.Notes)
	notes.SetMinimumHeight(80)
	sim1 := newLineEdit(card.GSMPhone1)
	sim2 := newLineEdit(card.GSMPhone2)
	subServerA := newLineEdit(card.SubServerA)
	subServerB := newLineEdit(card.SubServerB)
	channel := newSpinBox(int(card.ChannelCode), 0, 999999)
	ppkID := newSpinBox(int(card.PPKID), 0, 999999)
	hidden := newSpinBox(int(card.GSMHiddenN), 0, 999999999)
	testEnabled := qt.NewQCheckBox3("Увімкнено")
	testEnabled.SetChecked(card.TestControlEnabled)
	testInterval := newSpinBox(int(card.TestIntervalMin), 0, 1440*31)

	form.AddRow3("Коротка назва", shortName.QWidget)
	form.AddRow3("Повна назва", fullName.QWidget)
	form.AddRow3("Адреса", address.QWidget)
	form.AddRow3("Телефони", phones.QWidget)
	form.AddRow3("Договір", contract.QWidget)
	form.AddRow3("Розташування", location.QWidget)
	form.AddRow3("Примітки", notes.QWidget)
	form.AddRow3("SIM 1", sim1.QWidget)
	form.AddRow3("SIM 2", sim2.QWidget)
	form.AddRow3("GSM hidden", hidden.QWidget)
	form.AddRow3("Підсервер A", subServerA.QWidget)
	form.AddRow3("Підсервер B", subServerB.QWidget)
	form.AddRow3("Канал", channel.QWidget)
	form.AddRow3("PPK ID", ppkID.QWidget)
	form.AddRow3("Контроль тесту", testEnabled.QWidget)
	form.AddRow3("Інтервал тесту, хв", testInterval.QWidget)

	buttons := qt.NewQDialogButtonBox4(qt.QDialogButtonBox__Save | qt.QDialogButtonBox__Cancel)
	buttons.OnAccepted(dialog.Accept)
	buttons.OnRejected(dialog.Reject)

	layout.AddLayout(form.QLayout)
	layout.AddWidget(buttons.QWidget)
	dialog.SetLayout(layout.QLayout)

	if dialog.Exec() != int(qt.QDialog__Accepted) {
		return card, false
	}

	updated := card
	updated.ShortName = strings.TrimSpace(shortName.Text())
	updated.FullName = strings.TrimSpace(fullName.Text())
	updated.Address = strings.TrimSpace(address.Text())
	updated.Phones = strings.TrimSpace(phones.Text())
	updated.Contract = strings.TrimSpace(contract.Text())
	updated.Location = strings.TrimSpace(location.Text())
	updated.Notes = strings.TrimSpace(notes.ToPlainText())
	updated.GSMPhone1 = strings.TrimSpace(sim1.Text())
	updated.GSMPhone2 = strings.TrimSpace(sim2.Text())
	updated.GSMHiddenN = int64(hidden.Value())
	updated.SubServerA = strings.TrimSpace(subServerA.Text())
	updated.SubServerB = strings.TrimSpace(subServerB.Text())
	updated.ChannelCode = int64(channel.Value())
	updated.PPKID = int64(ppkID.Value())
	updated.TestControlEnabled = testEnabled.IsChecked()
	updated.TestIntervalMin = int64(testInterval.Value())
	return updated, true
}

func ShowSIMManagementDialog(
	parent *qt.QWidget,
	object models.Object,
	usageText string,
	vfService contracts.AdminObjectVodafoneService,
	ksService contracts.AdminObjectKyivstarService,
) {
	dialog := qt.NewQDialog(parent)
	dialog.SetWindowTitle("SIM-карти об'єкта")
	dialog.Resize(640, 480)

	layout := qt.NewQVBoxLayout(dialog.QWidget)
	form := qt.NewQFormLayout2()
	form.SetFieldGrowthPolicy(qt.QFormLayout__AllNonFixedFieldsGrow)
	form.AddRow3("Об'єкт", qt.NewQLabel3(fmt.Sprintf("<b>%s</b> №%s", htmlEscape(strings.TrimSpace(object.Name)), viewmodels.ObjectDisplayNumber(object))).QWidget)

	// SIM 1 Layout with Action Buttons if supported
	sim1Widget := qt.NewQWidget2()
	sim1Layout := qt.NewQHBoxLayout(sim1Widget)
	sim1Layout.SetContentsMargins(0, 0, 0, 0)
	sim1Layout.AddWidget(qt.NewQLabel3(emptyDash(object.SIM1)).QWidget)
	sim1Operator := simoperator.Detect(object.SIM1)
	if sim1Operator == simoperator.Vodafone && vfService != nil {
		btn := qt.NewQPushButton3("Vodafone M2M")
		btn.OnClicked(func() {
			ShowVodafoneSIMDialog(dialog.QWidget, vfService, object.SIM1, viewmodels.ObjectDisplayNumber(object), object.Name)
		})
		sim1Layout.AddWidget(btn.QWidget)
	} else if sim1Operator == simoperator.Kyivstar && ksService != nil {
		btn := qt.NewQPushButton3("Kyivstar M2M")
		btn.OnClicked(func() {
			ShowKyivstarSIMDialog(dialog.QWidget, ksService, object.SIM1, viewmodels.ObjectDisplayNumber(object), object.Name)
		})
		sim1Layout.AddWidget(btn.QWidget)
	}
	sim1Layout.AddStretch()
	sim1Widget.SetLayout(sim1Layout.QLayout)
	form.AddRow3("SIM 1", sim1Widget)

	// SIM 2 Layout with Action Buttons if supported
	sim2Widget := qt.NewQWidget2()
	sim2Layout := qt.NewQHBoxLayout(sim2Widget)
	sim2Layout.SetContentsMargins(0, 0, 0, 0)
	sim2Layout.AddWidget(qt.NewQLabel3(emptyDash(object.SIM2)).QWidget)
	sim2Operator := simoperator.Detect(object.SIM2)
	if sim2Operator == simoperator.Vodafone && vfService != nil {
		btn := qt.NewQPushButton3("Vodafone M2M")
		btn.OnClicked(func() {
			ShowVodafoneSIMDialog(dialog.QWidget, vfService, object.SIM2, viewmodels.ObjectDisplayNumber(object), object.Name)
		})
		sim2Layout.AddWidget(btn.QWidget)
	} else if sim2Operator == simoperator.Kyivstar && ksService != nil {
		btn := qt.NewQPushButton3("Kyivstar M2M")
		btn.OnClicked(func() {
			ShowKyivstarSIMDialog(dialog.QWidget, ksService, object.SIM2, viewmodels.ObjectDisplayNumber(object), object.Name)
		})
		sim2Layout.AddWidget(btn.QWidget)
	}
	sim2Layout.AddStretch()
	sim2Widget.SetLayout(sim2Layout.QLayout)
	form.AddRow3("SIM 2", sim2Widget)

	usage := qt.NewQTextEdit3(strings.TrimSpace(usageText))
	usage.SetReadOnly(true)
	usage.SetMinimumHeight(180)
	if strings.TrimSpace(usageText) == "" {
		usage.SetPlainText("Збігів використання SIM-номерів не знайдено.")
	}

	buttons := qt.NewQDialogButtonBox4(qt.QDialogButtonBox__Ok)
	buttons.OnAccepted(dialog.Accept)

	layout.AddLayout(form.QLayout)
	layout.AddWidget(usage.QWidget)
	layout.AddWidget(buttons.QWidget)
	dialog.SetLayout(layout.QLayout)
	dialog.Exec()
}


func newLineEdit(value string) *qt.QLineEdit {
	edit := qt.NewQLineEdit3(strings.TrimSpace(value))
	edit.SetClearButtonEnabled(true)
	return edit
}

func newSpinBox(value int, min int, max int) *qt.QSpinBox {
	spin := qt.NewQSpinBox2()
	spin.SetRange(min, max)
	spin.SetValue(value)
	return spin
}

func emptyDash(value string) string {
	if text := strings.TrimSpace(value); text != "" {
		return text
	}
	return "-"
}

func ObjectSIMUsageText(lookup viewmodels.SIMPhoneUsageLookup, object models.Object) string {
	exclude := int64(object.ID)
	vm := viewmodels.NewSIMPhoneUsageViewModel()
	parts := make([]string, 0, 2)
	for _, sim := range []struct {
		label string
		phone string
	}{
		{label: "SIM 1", phone: object.SIM1},
		{label: "SIM 2", phone: object.SIM2},
	} {
		phone := strings.TrimSpace(sim.phone)
		if phone == "" {
			parts = append(parts, sim.label+": номер не задано")
			continue
		}
		text := vm.ResolveUsageText(lookup, phone, &exclude)
		if strings.TrimSpace(text) == "" {
			text = "номер вільний"
		}
		parts = append(parts, sim.label+" "+phone+": "+text)
	}
	return strings.Join(parts, "\n")
}

func contactPositionText(contact models.Contact) string {
	position := strings.TrimSpace(contact.Position)
	switch strings.ToUpper(position) {
	case "IN_CHARGE":
		return "Відповідальна особа"
	case "OWNER":
		return "Власник"
	case "ADMIN", "MANAGER":
		return "Адміністратор"
	case "USER":
		return "Користувач"
	}
	if position != "" {
		if _, err := strconv.Atoi(position); err == nil {
			return "Відповідальна особа " + position
		}
		return position
	}
	if contact.Priority > 0 {
		return fmt.Sprintf("Відповідальна особа %d", contact.Priority)
	}
	return "-"
}
