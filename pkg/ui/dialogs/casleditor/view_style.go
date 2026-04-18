package casleditor

import (
	"image/color"
	"strconv"
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
)

var (
	wizardModalBackground = color.NRGBA{R: 250, G: 250, B: 250, A: 255}
	wizardPanelBackground = color.NRGBA{R: 255, G: 255, B: 255, A: 255}
	wizardMutedText       = color.NRGBA{R: 101, G: 112, B: 133, A: 255}
	wizardAccent          = color.NRGBA{R: 18, G: 58, B: 188, A: 255}
	wizardAccentSoft      = color.NRGBA{R: 221, G: 230, B: 255, A: 255}
	wizardBorder          = color.NRGBA{R: 219, G: 224, B: 235, A: 255}
	wizardDanger          = color.NRGBA{R: 220, G: 53, B: 69, A: 255}
)

func newWizardShell(content fyne.CanvasObject) fyne.CanvasObject {
	bg := canvas.NewRectangle(wizardModalBackground)
	return container.NewStack(bg, container.NewPadded(content))
}

func newWizardPanel(title string, content fyne.CanvasObject) fyne.CanvasObject {
	titleLabel := widget.NewLabel(title)
	titleLabel.TextStyle = fyne.TextStyle{Bold: true}
	return newWizardPanelWithHeader(titleLabel, content)
}

func newWizardPanelWithHeader(header fyne.CanvasObject, content fyne.CanvasObject) fyne.CanvasObject {
	bg := canvas.NewRectangle(wizardPanelBackground)
	border := canvas.NewRectangle(wizardBorder)
	border.SetMinSize(fyne.NewSize(1, 1))

	body := container.NewBorder(
		header,
		nil,
		nil,
		nil,
		content,
	)
	return container.NewStack(
		border,
		container.NewPadded(container.NewStack(bg, container.NewPadded(body))),
	)
}

func newWizardField(title string, input fyne.CanvasObject) fyne.CanvasObject {
	label := widget.NewLabel(title)
	label.TextStyle = fyne.TextStyle{Bold: true}
	label.Importance = widget.MediumImportance
	return newWizardFieldWithHeader(label, input, nil)
}

func newWizardFieldWithStatus(title string, input fyne.CanvasObject, status fyne.CanvasObject) fyne.CanvasObject {
	label := widget.NewLabel(title)
	label.TextStyle = fyne.TextStyle{Bold: true}
	label.Importance = widget.MediumImportance
	return newWizardFieldWithHeader(label, input, status)
}

func newWizardFieldWithHeader(label fyne.CanvasObject, input fyne.CanvasObject, status fyne.CanvasObject) fyne.CanvasObject {
	frame := canvas.NewRectangle(wizardPanelBackground)
	border := canvas.NewRectangle(wizardBorder)
	border.SetMinSize(fyne.NewSize(1, 42))

	field := container.NewStack(
		border,
		container.NewPadded(container.NewStack(frame, container.NewPadded(input))),
	)
	header := label
	if status != nil {
		gap := canvas.NewRectangle(color.Transparent)
		gap.SetMinSize(fyne.NewSize(6, 1))
		header = container.NewHBox(label, gap, status, layout.NewSpacer())
	}
	return container.NewVBox(header, field)
}

func newValidationStatusText() *canvas.Text {
	txt := canvas.NewText("", wizardDanger)
	txt.TextSize = 10
	txt.Alignment = fyne.TextAlignLeading
	return txt
}

func setValidationStatus(status *canvas.Text, err error) {
	if status == nil {
		return
	}
	setValidationMessage(status, validationMessageFromError(err), wizardDanger)
}

func setValidationMessage(status *canvas.Text, message string, clr color.Color) {
	if status == nil {
		return
	}
	status.Text = strings.TrimSpace(message)
	status.Color = clr
	status.Refresh()
}

func validationMessageFromError(err error) string {
	if err == nil {
		return ""
	}
	return "* " + strings.TrimSpace(err.Error())
}

func newWizardToolbar(closeAction func()) fyne.CanvasObject {
	closeBtn := newWizardHeaderAction(theme.CancelIcon(), wizardMutedText, closeAction)

	disabledIcon := func(res fyne.Resource) *widget.Button {
		btn := widget.NewButtonWithIcon("", res, func() {})
		btn.Importance = widget.LowImportance
		btn.Disable()
		return btn
	}

	return container.NewHBox(
		layout.NewSpacer(),
		disabledIcon(theme.InfoIcon()),
		disabledIcon(theme.DocumentCreateIcon()),
		disabledIcon(theme.DeleteIcon()),
		disabledIcon(theme.ContentClearIcon()),
		disabledIcon(theme.HelpIcon()),
		closeBtn,
	)
}

func newWizardHeaderAction(icon fyne.Resource, tint color.Color, onTap func()) fyne.CanvasObject {
	iconView := widget.NewIcon(icon)
	if tint != nil {
		iconView.Resource = theme.NewThemedResource(icon)
	}
	spacer := canvas.NewRectangle(color.Transparent)
	spacer.SetMinSize(fyne.NewSize(22, 22))
	return newCASLImageTapTarget(container.NewCenter(container.NewStack(spacer, iconView)), onTap)
}

func newCompactWizardTab(label string, active bool, onTap func()) fyne.CanvasObject {
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
	border.SetMinSize(fyne.NewSize(156, 28))
	text := canvas.NewText(label, textColor)
	text.TextSize = 10
	text.Alignment = fyne.TextAlignCenter
	text.TextStyle = fyne.TextStyle{Bold: active}
	spacer := canvas.NewRectangle(color.Transparent)
	spacer.SetMinSize(fyne.NewSize(156, 28))

	content := container.NewStack(
		border,
		container.NewPadded(container.NewStack(bg, container.NewCenter(text))),
		spacer,
	)
	return newCASLImageTapTarget(content, onTap)
}

func newWizardTitle(text string) *widget.Label {
	lbl := widget.NewLabel(text)
	lbl.TextStyle = fyne.TextStyle{Bold: true}
	lbl.Alignment = fyne.TextAlignCenter
	return lbl
}

func newWizardProgressBar(labels []string, currentStep int) fyne.CanvasObject {
	items := make([]fyne.CanvasObject, 0, len(labels)*2)
	for idx, label := range labels {
		step := idx + 1
		items = append(items, newWizardProgressItem(step, label, step < currentStep, step == currentStep))
		if idx < len(labels)-1 {
			lineColor := wizardBorder
			if step < currentStep {
				lineColor = wizardAccent
			}
			line := canvas.NewRectangle(lineColor)
			line.SetMinSize(fyne.NewSize(48, 4))
			items = append(items, container.NewCenter(line))
		}
	}
	return container.NewHBox(items...)
}

func newWizardProgressItem(step int, label string, done bool, active bool) fyne.CanvasObject {
	circleColor := wizardPanelBackground
	textColor := wizardMutedText
	if done || active {
		circleColor = wizardAccentSoft
		textColor = wizardAccent
	}

	circle := canvas.NewCircle(circleColor)
	spacer := canvas.NewRectangle(color.Transparent)
	spacer.SetMinSize(fyne.NewSize(38, 38))
	number := widget.NewLabelWithStyle(
		strconv.Itoa(step),
		fyne.TextAlignCenter,
		fyne.TextStyle{Bold: true},
	)
	number.Importance = widget.HighImportance
	text := canvas.NewText(label, textColor)
	text.TextSize = 11
	text.Alignment = fyne.TextAlignCenter

	return container.NewVBox(
		container.NewCenter(container.NewStack(spacer, circle, container.NewCenter(number))),
		container.NewCenter(text),
	)
}

func newWizardFooter(prevBtn, nextBtn fyne.CanvasObject) fyne.CanvasObject {
	return container.NewHBox(prevBtn, layout.NewSpacer(), nextBtn)
}

type wizardWrapLayout struct {
	hGap float32
	vGap float32
}

func newWizardWrapLayout(hGap, vGap float32) fyne.Layout {
	return &wizardWrapLayout{
		hGap: hGap,
		vGap: vGap,
	}
}

func (l *wizardWrapLayout) Layout(objects []fyne.CanvasObject, size fyne.Size) {
	x := float32(0)
	y := float32(0)
	rowHeight := float32(0)

	for _, obj := range objects {
		if obj == nil || !obj.Visible() {
			continue
		}
		min := obj.MinSize()
		if x > 0 && x+min.Width > size.Width {
			x = 0
			y += rowHeight + l.vGap
			rowHeight = 0
		}
		obj.Move(fyne.NewPos(x, y))
		obj.Resize(min)
		x += min.Width + l.hGap
		if min.Height > rowHeight {
			rowHeight = min.Height
		}
	}
}

func (l *wizardWrapLayout) MinSize(objects []fyne.CanvasObject) fyne.Size {
	width := float32(0)
	height := float32(0)
	maxHeight := float32(0)
	count := 0

	for _, obj := range objects {
		if obj == nil || !obj.Visible() {
			continue
		}
		min := obj.MinSize()
		if min.Width > width {
			width = min.Width
		}
		height += min.Height
		if count > 0 {
			height += l.vGap
		}
		if min.Height > maxHeight {
			maxHeight = min.Height
		}
		count++
	}
	if count == 1 {
		height = maxHeight
	}
	return fyne.NewSize(width, height)
}
