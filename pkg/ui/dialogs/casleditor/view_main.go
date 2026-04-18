package casleditor

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/widget"
	"obj_catalog_fyne_v3/pkg/contracts"
)

type MainView struct {
	vm *EditorViewModel

	win fyne.Window

	headerLabel *widget.Label
	statusLabel *widget.Label
	titleLabel  *widget.Label

	objectView *ObjectView
	deviceView *DeviceView
	linesView  *LinesView
	roomsView  *RoomsView
	wizardView *WizardView
}

func NewMainView(win fyne.Window, vm *EditorViewModel) *MainView {
	v := &MainView{
		vm:  vm,
		win: win,

		headerLabel: widget.NewLabel(""),
		statusLabel: widget.NewLabel(""),
		titleLabel:  widget.NewLabel(""),
	}

	v.objectView = NewObjectView(vm)
	v.deviceView = NewDeviceView(vm)
	v.linesView = NewLinesView(vm)
	v.roomsView = NewRoomsView(vm)

	v.setupLayout()
	v.bind()

	return v
}

func (v *MainView) setupLayout() {
	refreshBtn := widget.NewButton("Оновити", v.vm.Reload)
	closeBtn := widget.NewButton("Закрити", func() { v.win.Close() })

	var body fyne.CanvasObject
	if v.vm.IsCreating() {
		steps := []WizardStep{
			v.objectView,
			newDeviceWizardStep(v.deviceView, v.linesView),
			v.roomsView,
		}
		v.wizardView = NewWizardView(v.vm, steps)
		body = container.NewBorder(
			container.NewVBox(
				newWizardToolbar(func() { v.win.Close() }),
				newWizardTitle("Майстер створення об'єкта охорони"),
			),
			container.NewHBox(v.statusLabel),
			nil,
			nil,
			v.wizardView.Container,
		)
	} else {
		tabs := container.NewAppTabs(
			container.NewTabItem("Об'єкт", v.objectView.Container),
			container.NewTabItem("Обладнання", v.deviceView.Container),
			container.NewTabItem("Зони", v.linesView.Container),
			container.NewTabItem("Зв'язки", v.roomsView.Container),
		)
		body = tabs
	}
	if v.vm.IsCreating() {
		v.win.SetContent(newWizardShell(body))
		return
	}

	v.win.SetContent(container.NewBorder(
		container.NewVBox(
			v.headerLabel,
			container.NewHBox(refreshBtn, v.statusLabel, widget.NewSeparator(), closeBtn),
		),
		nil, nil, nil,
		body,
	))
}

func (v *MainView) bind() {
	v.vm.AddStatusUpdateListener(func(msg string) {
		v.statusLabel.SetText(msg)
	})

	v.vm.AddHeaderUpdateListener(func(msg string) {
		v.headerLabel.SetText(msg)
	})
	v.vm.AddErrorListener(func(err error) {
		dialog.ShowError(err, v.win)
	})
	v.vm.AddAlertListener(func(title, msg string) {
		dialog.ShowInformation(title, msg, v.win)
	})
}

func OpenEditor(parent fyne.Window, provider contracts.CASLObjectEditorProvider, objectID int64, onChanged func()) {
	title := "CASL: Редактор"
	if objectID <= 0 {
		title = "CASL: Створення об'єкта"
	}
	win := fyne.CurrentApp().NewWindow(title)
	win.Resize(fyne.NewSize(1024, 768))

	vm := NewEditorViewModel(win, provider, objectID, onChanged)
	NewMainView(win, vm)

	vm.LoadGeoZones()
	vm.Reload()
	win.Show()
}
