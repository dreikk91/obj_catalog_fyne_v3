//go:build qt

package qtui

import (
	"context"
	"fmt"
	"strings"
	"time"

	qt "github.com/mappu/miqt/qt6"

	"obj_catalog_fyne_v3/pkg/contracts"
)

// ShowCASLObjectBlockDialog loads the current CASL block state and offers the valid action.
func ShowCASLObjectBlockDialog(
	parent *qt.QWidget,
	provider contracts.CASLObjectEditorProvider,
	objectID int64,
	onSuccess func(),
) {
	if parent == nil || provider == nil || objectID <= 0 {
		return
	}
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		snapshot, err := provider.GetCASLObjectEditorSnapshot(ctx, objectID)
		RunOnMainThread(func() {
			if err != nil {
				qt.QMessageBox_Critical(parent, "Блокування CASL", err.Error())
				return
			}
			DeferOnMainThread(func() {
				if snapshot.Object.DeviceBlocked {
					showCASLObjectUnblockAction(parent, provider, snapshot, onSuccess)
					return
				}
				showCASLObjectBlockAction(parent, provider, snapshot, onSuccess)
			})
		})
	}()
}

func showCASLObjectBlockAction(
	parent *qt.QWidget,
	provider contracts.CASLObjectEditorProvider,
	snapshot contracts.CASLObjectEditorSnapshot,
	onSuccess func(),
) {
	if strings.TrimSpace(snapshot.Object.Device.DeviceID) == "" {
		qt.QMessageBox_Information(parent, "Блокування CASL", "До об'єкта не прив'язаний прилад CASL.")
		return
	}
	request, accepted := showCASLBlockDialog(parent, snapshot.Object.Device)
	if !accepted {
		return
	}
	runCASLBlockMutation(parent, "Блокування об'єкта CASL...", func(ctx context.Context) error {
		return provider.BlockCASLDevice(ctx, request)
	}, "Об'єкт CASL заблоковано.", onSuccess)
}

func showCASLObjectUnblockAction(
	parent *qt.QWidget,
	provider contracts.CASLObjectEditorProvider,
	snapshot contracts.CASLObjectEditorSnapshot,
	onSuccess func(),
) {
	deviceID := strings.TrimSpace(snapshot.Object.Device.DeviceID)
	if deviceID == "" {
		qt.QMessageBox_Information(parent, "Блокування CASL", "Ідентифікатор приладу CASL відсутній.")
		return
	}
	details := []string{"Об'єкт CASL зараз заблокований."}
	if reason := strings.TrimSpace(snapshot.Object.BlockMessage); reason != "" {
		details = append(details, "Причина: "+reason)
	}
	if snapshot.Object.TimeUnblock > 0 {
		details = append(details, "До: "+formatCASLBlockTime(snapshot.Object.TimeUnblock))
	}
	details = append(details, "", "Розблокувати об'єкт?")
	if qt.QMessageBox_Question(parent, "Блокування CASL", strings.Join(details, "\n")) != qt.QMessageBox__Yes {
		return
	}
	runCASLBlockMutation(parent, "Розблокування об'єкта CASL...", func(ctx context.Context) error {
		return provider.UnblockCASLDevice(ctx, deviceID)
	}, "Об'єкт CASL розблоковано.", onSuccess)
}

func runCASLBlockMutation(
	parent *qt.QWidget,
	progress string,
	mutation func(context.Context) error,
	success string,
	onSuccess func(),
) {
	qt.QGuiApplication_SetOverrideCursor(qt.NewQCursor2(qt.WaitCursor))
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		err := mutation(ctx)
		RunOnMainThread(func() {
			qt.QGuiApplication_RestoreOverrideCursor()
			if err != nil {
				qt.QMessageBox_Critical(parent, "Блокування CASL", err.Error())
				return
			}
			qt.QMessageBox_Information(parent, "Блокування CASL", success)
			if onSuccess != nil {
				onSuccess()
			}
		})
	}()
	_ = progress
}

func formatCASLBlockTime(unixTime int64) string {
	if unixTime >= 2_554_790_000 {
		return "безстроково"
	}
	return time.Unix(unixTime, 0).Local().Format("02.01.2006 15:04")
}

func caslBlockDescription(snapshot contracts.CASLObjectEditorSnapshot) string {
	return fmt.Sprintf("№%d %s", snapshot.Object.Device.Number, strings.TrimSpace(snapshot.Object.Name))
}
