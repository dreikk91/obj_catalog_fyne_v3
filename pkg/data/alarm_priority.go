package data

import "obj_catalog_fyne_v3/pkg/models"

// isPrimaryAlarmEventType returns true for "core" alarm events that must stay
// prioritized in grouped alarm rows even if newer technical events appear.
func isPrimaryAlarmEventType(eventType models.EventType) bool {
	switch eventType {
	case models.EventFire,
		models.EventBurglary,
		models.EventPanic,
		models.EventMedical,
		models.EventGas,
		models.EventTamper,
		models.EventAlarmNotification:
		return true
	default:
		return false
	}
}
