package casleditor

import (
	"fmt"
	"image/color"
	"sort"
	"strconv"
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"

	"obj_catalog_fyne_v3/pkg/contracts"
)

type LinesView struct {
	vm *EditorViewModel

	Container *fyne.Container

	quickLineNameEntry *widget.Entry
	quickLineHintLabel *widget.Label
	quickLineStatus    *canvas.Text
	quickTabsHost      *fyne.Container
	selectedTabKey     int
	syncing            bool
}

var quickLineNameValidator fyne.StringValidator = func(value string) error {
	if strings.TrimSpace(value) == "" {
		return fmt.Errorf("вкажіть назву зони")
	}
	return nil
}

func NewLinesView(vm *EditorViewModel) *LinesView {
	v := &LinesView{
		vm: vm,

		quickLineNameEntry: widget.NewEntry(),
		quickLineHintLabel: widget.NewLabel(""),
		quickLineStatus:    newValidationStatusText(),
		quickTabsHost:      container.NewMax(),
	}

	v.quickLineNameEntry.SetPlaceHolder("Введіть назву зони та натисніть Enter")
	v.quickLineNameEntry.OnChanged = func(value string) {
		if strings.TrimSpace(value) == "" {
			setValidationMessage(v.quickLineStatus, "", wizardDanger)
			return
		}
		setValidationMessage(v.quickLineStatus, "", wizardDanger)
	}
	v.quickLineNameEntry.OnSubmitted = v.handleQuickLineSubmit

	v.setupLayout()
	v.bind()

	return v
}

func (v *LinesView) setupLayout() {
	headerRow := container.NewHBox(
		newQuickLineHeader("Тип адаптера", 120),
		newQuickLineHeader("№ адапт.", 72),
		newQuickLineHeader("№ групи", 72),
		newQuickLineHeader("№ зони", 72),
		newQuickLineHeader("Тип шлейфа", 170),
		newQuickLineHeader("Назва зони", 320),
		newQuickLineHeader("Стан", 120),
	)

	listCard := newWizardPanel("Зони", container.NewBorder(
		container.NewVBox(
			v.quickLineHintLabel,
			newWizardFieldWithStatus("Назва наступної зони", v.quickLineNameEntry, v.quickLineStatus),
			headerRow,
		),
		nil,
		nil,
		nil,
		v.quickTabsHost,
	))
	v.Container = container.NewBorder(nil, nil, nil, nil, listCard)
}

func (v *LinesView) bind() {
	v.vm.AddDataChangedListener(func() {
		v.syncing = true
		v.refreshQuickLineHint()
		v.rebuildQuickRows()
		v.syncing = false
	})
}

func (v *LinesView) rebuildQuickRows() {
	deviceType := strings.TrimSpace(v.vm.Snapshot.Object.Device.Type)
	canEditAdapter := canEditCASLAdapterForDevice(v.vm.Snapshot.Dictionary, deviceType)
	adapterOptions := caslAdapterOptionLabelsForDevice(v.vm.Snapshot.Dictionary, deviceType, v.vm.AdapterOptionToID)

	type indexedLine struct {
		index int
		line  contracts.CASLDeviceLineDetails
	}
	lines := make([]indexedLine, 0, len(v.vm.Snapshot.Object.Device.Lines))
	for idx, line := range v.vm.Snapshot.Object.Device.Lines {
		lines = append(lines, indexedLine{index: idx, line: line})
	}
	sort.Slice(lines, func(i, j int) bool {
		if lines[i].line.LineNumber != lines[j].line.LineNumber {
			return lines[i].line.LineNumber < lines[j].line.LineNumber
		}
		return lines[i].index < lines[j].index
	})

	groupedRows := map[int][]fyne.CanvasObject{}
	tabKeys := make([]int, 0)
	for _, item := range lines {
		idx := item.index
		line := item.line
		index := idx
		current := line
		lineNumberValidator := fyne.StringValidator(func(value string) error {
			num, err := ParseCASLEditorInt(value)
			if err != nil {
				return fmt.Errorf("номер зони має містити лише цифри")
			}
			if err := ValidateCASLLineNumberRange(num); err != nil {
				return err
			}
			return ValidateCASLLineNumberUnique(v.vm.Snapshot.Object.Device.Lines, num, index)
		})
		lineNameValidator := fyne.StringValidator(ValidateCASLLineDescription)

		adapterSelect := widget.NewSelect(adapterOptions, func(selected string) {
			if !canEditAdapter {
				return
			}
			mapped := v.vm.AdapterOptionToID[selected]
			if mapped == "" {
				mapped = strings.TrimSpace(selected)
			}
			v.vm.Snapshot.Object.Device.Lines[index].AdapterType = mapped
		})
		selectedAdapterLabel := optionLabelByValue(current.AdapterType, v.vm.AdapterOptionToID)
		if !stringSliceContains(adapterOptions, selectedAdapterLabel) && len(adapterOptions) > 0 {
			selectedAdapterLabel = adapterOptions[0]
		}
		adapterSelect.SetSelected(selectedAdapterLabel)
		if !canEditAdapter {
			adapterSelect.Disable()
		}

		groupEntry := widget.NewEntry()
		groupEntry.SetText(strconv.Itoa(current.GroupNumber))
		groupEntry.OnChanged = func(value string) {
			num, err := ParseCASLEditorInt(value)
			if err != nil {
				return
			}
			v.vm.Snapshot.Object.Device.Lines[index].GroupNumber = num
		}

		var nameEntry *widget.Entry
		var stateStatus *canvas.Text

		numberEntry := widget.NewEntry()
		numberEntry.SetText(strconv.Itoa(current.LineNumber))
		numberEntry.OnChanged = func(value string) {
			num, err := ParseCASLEditorInt(value)
			if err != nil {
				updateLineRowStatus(stateStatus, lineNumberValidator(numberEntry.Text), lineNameValidator(nameEntry.Text))
				return
			}
			if ValidateCASLLineNumberUnique(v.vm.Snapshot.Object.Device.Lines, num, index) == nil {
				v.vm.Snapshot.Object.Device.Lines[index].LineNumber = num
				v.refreshQuickLineHint()
				v.vm.emitDataChanged()
			}
			updateLineRowStatus(stateStatus, lineNumberValidator(numberEntry.Text), lineNameValidator(nameEntry.Text))
		}

		typeEntry := widget.NewSelectEntry(v.vm.LineTypeOptions)
		typeEntry.SetText(optionLabelByValue(current.LineType, v.vm.LineTypeOptionToID))
		typeEntry.OnChanged = func(value string) {
			mapped := v.vm.LineTypeOptionToID[value]
			if mapped == "" {
				mapped = strings.TrimSpace(value)
			}
			v.vm.Snapshot.Object.Device.Lines[index].LineType = mapped
		}

		nameEntry = widget.NewEntry()
		nameEntry.SetText(current.Description)
		nameEntry.OnChanged = func(value string) {
			v.vm.Snapshot.Object.Device.Lines[index].Description = strings.TrimSpace(value)
			updateLineRowStatus(stateStatus, lineNumberValidator(numberEntry.Text), lineNameValidator(nameEntry.Text))
		}

		adapterNumberEntry := widget.NewEntry()
		if current.AdapterNumber > 0 {
			adapterNumberEntry.SetText(strconv.Itoa(current.AdapterNumber))
		}
		adapterNumberEntry.OnChanged = func(value string) {
			num, err := ParseCASLEditorInt(value)
			if err != nil {
				return
			}
			v.vm.Snapshot.Object.Device.Lines[index].AdapterNumber = num
		}
		if !canEditAdapter {
			adapterNumberEntry.Disable()
		}

		blockedCheck := widget.NewCheck("", func(checked bool) {
			v.vm.Snapshot.Object.Device.Lines[index].IsBlocked = checked
		})
		blockedCheck.SetChecked(current.IsBlocked)
		stateStatus = newValidationStatusText()
		updateLineRowStatus(stateStatus, lineNumberValidator(numberEntry.Text), lineNameValidator(nameEntry.Text))

		row := container.NewHBox(
			newQuickLineCell(wrapQuickLineField(adapterSelect), 120),
			newQuickLineCell(adapterNumberEntry, 72),
			newQuickLineCell(groupEntry, 72),
			newQuickLineCell(numberEntry, 72),
			newQuickLineCell(typeEntry, 170),
			newQuickLineCell(nameEntry, 320),
			newQuickLineCell(container.NewVBox(container.NewCenter(blockedCheck), container.NewCenter(stateStatus)), 120),
		)
		tabKey := quickLineTabIndex(current.LineNumber)
		if _, ok := groupedRows[tabKey]; !ok {
			tabKeys = append(tabKeys, tabKey)
		}
		groupedRows[tabKey] = append(groupedRows[tabKey], row)
	}

	if len(groupedRows) == 0 {
		v.quickTabsHost.Objects = []fyne.CanvasObject{
			container.NewVScroll(container.NewVBox(widget.NewLabel("Ще немає зон. Введіть назву вище та натисніть Enter."))),
		}
		v.quickTabsHost.Refresh()
		return
	}

	sort.Ints(tabKeys)
	if !intSliceContains(tabKeys, v.selectedTabKey) {
		v.selectedTabKey = tabKeys[0]
	}
	tabButtons := make([]fyne.CanvasObject, 0, len(tabKeys))
	for _, tabKey := range tabKeys {
		key := tabKey
		tabButtons = append(tabButtons, newQuickRangeTab(quickLineTabTitle(key), key == v.selectedTabKey, func() {
			if v.selectedTabKey == key {
				return
			}
			v.selectedTabKey = key
			v.rebuildQuickRows()
		}))
	}
	rowsBox := container.NewVBox(groupedRows[v.selectedTabKey]...)
	content := container.NewBorder(
		container.NewHBox(tabButtons...),
		nil,
		nil,
		nil,
		container.NewVScroll(rowsBox),
	)
	v.quickTabsHost.Objects = []fyne.CanvasObject{content}
	v.quickTabsHost.Refresh()
}

func (v *LinesView) handleQuickLineSubmit(name string) {
	name = strings.TrimSpace(name)
	if err := quickLineNameValidator(name); err != nil {
		setValidationStatus(v.quickLineStatus, err)
		return
	}
	setValidationMessage(v.quickLineStatus, "", wizardDanger)

	deviceType := strings.TrimSpace(v.vm.Snapshot.Object.Device.Type)
	limit := caslDeviceLineLimit(deviceType)
	if len(v.vm.Snapshot.Object.Device.Lines) >= limit {
		v.vm.showError(fmt.Errorf("для типу %s досягнуто ліміт шлейфів: %d", FirstNonEmpty(caslDeviceTypeDisplayName(deviceType), deviceType), limit))
		return
	}

	lineNumber, adapterType, adapterNumber, groupNumber, lineType := nextCASLLineDefaults(v.vm.Snapshot.Dictionary, deviceType, v.vm.Snapshot.Object.Device.Lines)
	if err := ValidateCASLLineNumberUnique(v.vm.Snapshot.Object.Device.Lines, lineNumber, -1); err != nil {
		v.vm.showError(err)
		return
	}

	v.vm.CreateLine(LineMutationData{
		LineNumber:    lineNumber,
		Description:   name,
		LineType:      lineType,
		GroupNumber:   groupNumber,
		AdapterType:   adapterType,
		AdapterNumber: adapterNumber,
	})
	v.quickLineNameEntry.SetText("")
	if v.vm.win != nil {
		v.vm.win.Canvas().Focus(v.quickLineNameEntry)
	}
}

func (v *LinesView) refreshQuickLineHint() {
	deviceType := strings.TrimSpace(v.vm.Snapshot.Object.Device.Type)
	nextNumber := NextCASLLineNumber(v.vm.Snapshot.Object.Device.Lines)
	limit := caslDeviceLineLimit(deviceType)
	v.quickLineHintLabel.SetText(fmt.Sprintf("Наступний вільний номер зони: %d. Створено %d з %d можливих шлейфів.", nextNumber, len(v.vm.Snapshot.Object.Device.Lines), limit))
}

func newQuickLineHeader(label string, width float32) fyne.CanvasObject {
	text := widget.NewLabel(label)
	text.TextStyle = fyne.TextStyle{Bold: true}
	return newQuickLineCell(text, width)
}

func newQuickLineCell(content fyne.CanvasObject, width float32) fyne.CanvasObject {
	spacer := canvas.NewRectangle(color.Transparent)
	spacer.SetMinSize(fyne.NewSize(width, 36))
	return container.NewStack(spacer, content)
}

func wrapQuickLineField(content fyne.CanvasObject) fyne.CanvasObject {
	return container.NewPadded(content)
}

func stringSliceContains(values []string, needle string) bool {
	for _, value := range values {
		if value == needle {
			return true
		}
	}
	return false
}

func quickLineTabIndex(lineNumber int) int {
	if lineNumber <= 0 {
		return 0
	}
	return (lineNumber - 1) / 8
}

func updateLineRowStatus(status *canvas.Text, numberErr error, nameErr error) {
	switch {
	case numberErr != nil:
		setValidationStatus(status, numberErr)
	case nameErr != nil:
		setValidationStatus(status, nameErr)
	default:
		setValidationMessage(status, "", wizardDanger)
	}
}

func quickLineTabTitle(tabIndex int) string {
	start := tabIndex*8 + 1
	end := start + 7
	return fmt.Sprintf("%d-%d", start, end)
}

func newQuickRangeTab(label string, active bool, onTap func()) fyne.CanvasObject {
	bgColor := wizardPanelBackground
	borderColor := wizardBorder
	textColor := wizardMutedText
	if active {
		bgColor = wizardAccentSoft
		borderColor = wizardAccent
		textColor = wizardAccent
	}

	bg := canvas.NewRectangle(bgColor)
	border := canvas.NewRectangle(borderColor)
	border.SetMinSize(fyne.NewSize(74, 28))
	text := canvas.NewText(label, textColor)
	text.TextSize = 10
	text.Alignment = fyne.TextAlignCenter
	text.TextStyle = fyne.TextStyle{Bold: active}
	spacer := canvas.NewRectangle(color.Transparent)
	spacer.SetMinSize(fyne.NewSize(74, 28))

	content := container.NewStack(
		border,
		container.NewPadded(container.NewStack(bg, container.NewCenter(text))),
		spacer,
	)
	return newCASLImageTapTarget(content, onTap)
}

func intSliceContains(values []int, needle int) bool {
	for _, value := range values {
		if value == needle {
			return true
		}
	}
	return false
}
