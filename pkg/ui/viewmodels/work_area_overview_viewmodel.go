package viewmodels

import (
	"fmt"
	"sort"
	"strings"
	"time"

	"obj_catalog_fyne_v3/pkg/models"
)

// WorkAreaOverviewTone controls the visual priority of the operational summary.
type WorkAreaOverviewTone string

const (
	WorkAreaOverviewNormal   WorkAreaOverviewTone = "normal"
	WorkAreaOverviewWarning  WorkAreaOverviewTone = "warning"
	WorkAreaOverviewCritical WorkAreaOverviewTone = "critical"
)

// WorkAreaOverview contains the small operational subset needed by an operator.
type WorkAreaOverview struct {
	Source           string
	Summary          string
	SummaryTone      WorkAreaOverviewTone
	Device           string
	Channel          string
	Signal           string
	LastEvent        string
	LastTest         string
	TestControl      string
	Phone            string
	ResponseGroup    string
	Location         string
	AdditionalInfo   string
	GroupCount       int
	ZoneCount        int
	ProblemZoneCount int
	ContactCount     int
	ProblemZones     []models.Zone
	PriorityContacts []models.Contact
}

// WorkAreaOverviewViewModel builds a source-neutral dispatcher overview.
type WorkAreaOverviewViewModel struct{}

func NewWorkAreaOverviewViewModel() *WorkAreaOverviewViewModel {
	return &WorkAreaOverviewViewModel{}
}

func (vm *WorkAreaOverviewViewModel) Build(
	object models.Object,
	zones []models.Zone,
	contacts []models.Contact,
	device WorkAreaDevicePresentation,
) WorkAreaOverview {
	problemZones := operationalProblemZones(zones)
	priorityContacts := operationalPriorityContacts(contacts, 5)

	return WorkAreaOverview{
		Source:           ObjectSourceByID(object.ID),
		Summary:          operationalObjectSummary(object, problemZones, device),
		SummaryTone:      operationalObjectTone(object, problemZones, device),
		Device:           operationalDeviceText(device),
		Channel:          trimWorkAreaPrefix(device.ChannelText),
		Signal:           emptyWorkAreaValue(object.SignalStrength),
		LastEvent:        formatOperationalTime(object.LastMessageTime),
		LastTest:         formatOperationalTime(object.LastTestTime),
		TestControl:      trimWorkAreaPrefix(device.TestControlText),
		Phone:            emptyWorkAreaValue(device.PhoneCopyText),
		ResponseGroup:    operationalResponseGroup(object),
		Location:         emptyWorkAreaValue(object.Location1),
		AdditionalInfo:   emptyWorkAreaValue(object.Notes1),
		GroupCount:       operationalGroupCount(object, zones),
		ZoneCount:        len(zones),
		ProblemZoneCount: len(problemZones),
		ContactCount:     len(contacts),
		ProblemZones:     problemZones,
		PriorityContacts: priorityContacts,
	}
}

func operationalObjectSummary(
	object models.Object,
	problemZones []models.Zone,
	device WorkAreaDevicePresentation,
) string {
	switch {
	case object.Status == models.StatusFire || containsFireOrAlarmZone(problemZones):
		return "ТРИВОГА — перевірте активні зони та журнал подій"
	case object.ConnectionStatusValue() == models.ConnectionStatusOffline || object.Status == models.StatusOffline:
		return "НЕМАЄ ЗВ'ЯЗКУ — дані об'єкта можуть бути неактуальними"
	case object.MonitoringStatusValue() == models.MonitoringStatusBlocked:
		return "МОНІТОРИНГ ЗАБЛОКОВАНО"
	case strings.Contains(strings.ToLower(device.SummaryPowerText), "невідом"):
		return "СТАН ЖИВЛЕННЯ НЕ ВИЗНАЧЕНО"
	case strings.Contains(strings.ToLower(device.SummaryPowerText), "відсутн"),
		strings.Contains(strings.ToLower(device.SummaryPowerText), "тривога"),
		len(problemZones) > 0:
		return "Є ТЕХНІЧНІ ВІДХИЛЕННЯ — перевірте проблемні зони та живлення"
	default:
		return "ОПЕРАТИВНИЙ СТАН У НОРМІ"
	}
}

func operationalObjectTone(
	object models.Object,
	problemZones []models.Zone,
	device WorkAreaDevicePresentation,
) WorkAreaOverviewTone {
	switch {
	case object.Status == models.StatusFire || containsFireOrAlarmZone(problemZones):
		return WorkAreaOverviewCritical
	case object.ConnectionStatusValue() == models.ConnectionStatusOffline || object.Status == models.StatusOffline:
		return WorkAreaOverviewCritical
	case object.MonitoringStatusValue() == models.MonitoringStatusBlocked:
		return WorkAreaOverviewWarning
	case strings.Contains(strings.ToLower(device.SummaryPowerText), "невідом"):
		return WorkAreaOverviewWarning
	case strings.Contains(strings.ToLower(device.SummaryPowerText), "відсутн"),
		strings.Contains(strings.ToLower(device.SummaryPowerText), "тривога"),
		len(problemZones) > 0:
		return WorkAreaOverviewWarning
	default:
		return WorkAreaOverviewNormal
	}
}

func operationalProblemZones(zones []models.Zone) []models.Zone {
	result := make([]models.Zone, 0)
	for _, zone := range zones {
		if zone.IsBypassed ||
			zone.Status == models.ZoneAlarm ||
			zone.Status == models.ZoneFire ||
			zone.Status == models.ZoneBreak ||
			zone.Status == models.ZoneShort {
			result = append(result, zone)
		}
	}
	sort.SliceStable(result, func(i, j int) bool {
		left := operationalZonePriority(result[i])
		right := operationalZonePriority(result[j])
		if left != right {
			return left < right
		}
		return result[i].Number < result[j].Number
	})
	return result
}

func operationalZonePriority(zone models.Zone) int {
	switch zone.Status {
	case models.ZoneFire:
		return 0
	case models.ZoneAlarm:
		return 1
	case models.ZoneBreak, models.ZoneShort:
		return 2
	default:
		if zone.IsBypassed {
			return 3
		}
		return 4
	}
}

func containsFireOrAlarmZone(zones []models.Zone) bool {
	for _, zone := range zones {
		if zone.Status == models.ZoneFire || zone.Status == models.ZoneAlarm {
			return true
		}
	}
	return false
}

func operationalPriorityContacts(contacts []models.Contact, limit int) []models.Contact {
	result := append([]models.Contact(nil), contacts...)
	sort.SliceStable(result, func(i, j int) bool {
		left := result[i].Priority
		right := result[j].Priority
		if left <= 0 {
			left = int(^uint(0) >> 1)
		}
		if right <= 0 {
			right = int(^uint(0) >> 1)
		}
		return left < right
	})
	if limit > 0 && len(result) > limit {
		result = result[:limit]
	}
	return result
}

func operationalDeviceText(device WorkAreaDevicePresentation) string {
	deviceType := trimWorkAreaPrefix(device.DeviceTypeText)
	panelMark := trimWorkAreaPrefix(device.PanelMarkText)
	switch {
	case deviceType == "—":
		return panelMark
	case panelMark == "—" || panelMark == deviceType:
		return deviceType
	default:
		return deviceType + " / " + panelMark
	}
}

func operationalResponseGroup(object models.Object) string {
	if name := strings.TrimSpace(object.PreferredResponseGroupName); name != "" {
		return name
	}
	if id := strings.TrimSpace(object.PreferredResponseGroupID); id != "" {
		return id
	}
	return "—"
}

func operationalGroupCount(object models.Object, zones []models.Zone) int {
	if len(object.Groups) > 0 {
		return len(object.Groups)
	}
	keys := make(map[string]struct{})
	for _, zone := range zones {
		key := strings.TrimSpace(zone.GroupID)
		if key == "" {
			key = fmt.Sprintf("%d|%s", zone.GroupNumber, strings.TrimSpace(zone.GroupName))
		}
		keys[key] = struct{}{}
	}
	return len(keys)
}

func formatOperationalTime(value time.Time) string {
	if value.IsZero() {
		return "—"
	}
	return value.Format(workAreaDateTimeLayout)
}

func trimWorkAreaPrefix(value string) string {
	if idx := strings.Index(value, ": "); idx >= 0 {
		return strings.TrimSpace(value[idx+2:])
	}
	return emptyWorkAreaValue(value)
}
