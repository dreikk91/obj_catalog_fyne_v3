package casleditor

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
)

type deviceWizardStep struct {
	deviceView *DeviceView
	linesView  *LinesView
	content    fyne.CanvasObject
}

func newDeviceWizardStep(deviceView *DeviceView, linesView *LinesView) WizardStep {
	split := container.NewHSplit(deviceView.Container, linesView.Container)
	split.SetOffset(0.48)
	return &deviceWizardStep{
		deviceView: deviceView,
		linesView:  linesView,
		content:    split,
	}
}

func (s *deviceWizardStep) Title() string { return "Крок 2. Обладнання та зони" }

func (s *deviceWizardStep) ProgressLabel() string {
	return "Створення об'єктового обладнання"
}

func (s *deviceWizardStep) Content() fyne.CanvasObject { return s.content }

func (s *deviceWizardStep) CommitDraft() error {
	return s.deviceView.CommitDraft()
}
