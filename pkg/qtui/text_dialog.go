//go:build qt

package qtui

import (
	"strings"

	qt "github.com/mappu/miqt/qt6"
)

func ShowTextDialog(parent *qt.QWidget, title string, text string) {
	dialog := qt.NewQDialog(parent)
	dialog.SetWindowTitle(strings.TrimSpace(title))
	dialog.Resize(720, 560)

	editor := qt.NewQTextEdit3(strings.TrimSpace(text))
	editor.SetReadOnly(true)
	editor.SetLineWrapMode(qt.QTextEdit__NoWrap)

	buttons := qt.NewQDialogButtonBox4(qt.QDialogButtonBox__Close)
	copyButton := buttons.AddButton2("Копіювати", qt.QDialogButtonBox__ActionRole)
	copyButton.OnClicked(func() {
		setClipboardText(editor.ToPlainText())
	})
	buttons.OnRejected(dialog.Reject)

	layout := qt.NewQVBoxLayout(dialog.QWidget)
	layout.AddWidget(editor.QWidget)
	layout.AddWidget(buttons.QWidget)
	dialog.SetLayout(layout.QLayout)
	dialog.Exec()
}
