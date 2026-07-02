//go:build qt

package qtapp

import "testing"

func TestAlarmResponseDialogGuardPreventsDuplicateOpen(t *testing.T) {
	app := &Application{}

	if !app.beginAlarmResponseDialog(42) {
		t.Fatal("first response dialog request must be accepted")
	}
	if app.beginAlarmResponseDialog(42) {
		t.Fatal("duplicate response dialog request must be rejected")
	}
	if app.beginAlarmResponseDialog(43) {
		t.Fatal("another response dialog must be rejected while the first is active")
	}

	app.endAlarmResponseDialog(43)
	if app.beginAlarmResponseDialog(43) {
		t.Fatal("finishing a different alarm must not release the guard")
	}

	app.endAlarmResponseDialog(42)
	if !app.beginAlarmResponseDialog(43) {
		t.Fatal("guard must be released after the active dialog closes")
	}
}
