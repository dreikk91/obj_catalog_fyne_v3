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

const caslPermanentBlockUnix int64 = 2_554_790_050

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
	runCASLBlockMutation(parent, func(ctx context.Context) error {
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
	runCASLBlockMutation(parent, func(ctx context.Context) error {
		return provider.UnblockCASLDevice(ctx, deviceID)
	}, "Об'єкт CASL розблоковано.", onSuccess)
}

func runCASLBlockMutation(
	parent *qt.QWidget,
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
}

func formatCASLBlockTime(unixTime int64) string {
	if unixTime >= caslPermanentBlockUnix-50 {
		return "безстроково"
	}
	return time.Unix(unixTime, 0).Local().Format("02.01.2006 15:04")
}

func buildCASLDeviceBlockRequest(
	device contracts.CASLDeviceDetails,
	hours int,
	minutes int,
	reason string,
	permanent bool,
	now time.Time,
) (contracts.CASLDeviceBlockRequest, error) {
	reason = strings.TrimSpace(reason)
	if len([]rune(reason)) < 3 {
		return contracts.CASLDeviceBlockRequest{}, fmt.Errorf("причина блокування має містити щонайменше 3 символи")
	}
	if strings.TrimSpace(device.DeviceID) == "" {
		return contracts.CASLDeviceBlockRequest{}, fmt.Errorf("ідентифікатор приладу CASL відсутній")
	}
	until := caslPermanentBlockUnix
	if !permanent {
		durationMinutes := hours*60 + minutes
		if durationMinutes <= 0 || durationMinutes > 24*60 {
			return contracts.CASLDeviceBlockRequest{}, fmt.Errorf("тривалість має бути в межах від 1 хвилини до 24 годин")
		}
		until = now.Add(time.Duration(durationMinutes) * time.Minute).Unix()
	}
	return contracts.CASLDeviceBlockRequest{
		DeviceID:     strings.TrimSpace(device.DeviceID),
		DeviceNumber: device.Number,
		TimeUnblock:  until,
		Message:      reason,
	}, nil
}
