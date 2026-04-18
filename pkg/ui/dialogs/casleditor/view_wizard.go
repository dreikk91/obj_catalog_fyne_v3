package casleditor

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"
)

type WizardStep interface {
	Title() string
	ProgressLabel() string
	Content() fyne.CanvasObject
	CommitDraft() error
}

type WizardView struct {
	vm *EditorViewModel

	Container *fyne.Container

	content   *fyne.Container
	stepTitle *widget.Label
	progress  *fyne.Container
	prevBtn   *widget.Button
	nextBtn   *widget.Button
	steps     []WizardStep
}

func NewWizardView(vm *EditorViewModel, steps []WizardStep) *WizardView {
	v := &WizardView{
		vm:    vm,
		steps: steps,
	}

	v.content = container.NewMax()
	v.stepTitle = newWizardTitle("")
	v.progress = container.NewHBox()
	v.prevBtn = widget.NewButton("Назад", v.handlePrev)
	v.nextBtn = widget.NewButton("Далі", v.handleNext)
	v.prevBtn.Importance = widget.LowImportance
	v.nextBtn.Importance = widget.HighImportance

	v.Container = container.NewBorder(
		container.NewVBox(v.stepTitle, v.progress),
		newWizardFooter(v.prevBtn, v.nextBtn),
		nil,
		nil,
		v.content,
	)

	v.refreshStep()
	return v
}

func (v *WizardView) handlePrev() {
	if v.vm.WizardStep > 1 {
		v.vm.WizardStep--
		v.refreshStep()
	}
}

func (v *WizardView) handleNext() {
	if len(v.steps) == 0 {
		return
	}
	step := v.steps[v.vm.WizardStep-1]
	if err := step.CommitDraft(); err != nil {
		v.vm.showError(err)
		return
	}
	if v.vm.WizardStep == len(v.steps) {
		v.vm.CommitCreationWizard()
		return
	}
	v.vm.WizardStep++
	v.refreshStep()
}

func (v *WizardView) refreshStep() {
	if len(v.steps) == 0 {
		v.content.Objects = []fyne.CanvasObject{widget.NewLabel("Кроки майстра відсутні")}
		v.content.Refresh()
		return
	}
	if v.vm.WizardStep < 1 {
		v.vm.WizardStep = 1
	}
	if v.vm.WizardStep > len(v.steps) {
		v.vm.WizardStep = len(v.steps)
	}

	idx := v.vm.WizardStep - 1
	step := v.steps[idx]
	v.stepTitle.SetText(step.Title())
	labels := make([]string, 0, len(v.steps))
	for _, item := range v.steps {
		labels = append(labels, item.ProgressLabel())
	}
	v.progress.Objects = []fyne.CanvasObject{newWizardProgressBar(labels, v.vm.WizardStep)}
	v.progress.Refresh()
	v.content.Objects = []fyne.CanvasObject{step.Content()}
	v.content.Refresh()

	v.prevBtn.Enable()
	if v.vm.WizardStep == 1 {
		v.prevBtn.Disable()
	}
	if v.vm.WizardStep == len(v.steps) {
		v.nextBtn.SetText("Створити")
	} else {
		v.nextBtn.SetText("Далі")
	}
}
