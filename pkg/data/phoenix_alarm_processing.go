package data

import (
	"context"
	"database/sql"
	"fmt"
	"strconv"
	"strings"
	"time"

	"obj_catalog_fyne_v3/pkg/contracts"
	"obj_catalog_fyne_v3/pkg/models"

	"github.com/rs/zerolog/log"
)

// StateEvent values in Phoenix Temp table.
const (
	phoenixStateUnassigned   = int64(0) // active alarm without an operator
	phoenixStateInWork       = int64(1) // operator took the alarm
	phoenixStateGroupSent    = int64(2) // response group dispatched
	phoenixStateGroupArrived = int64(3) // response group arrived
)

// GroupResponse status_id values (from StatusGroupResponse).
const (
	phoenixGroupStatusFree       = int64(1) // Вільна
	phoenixGroupStatusDispatched = int64(2) // На виїзді
	phoenixGroupStatusArrived    = int64(3) // Прибула
)

// PickAlarm marks the alarm as being actively worked on or transfers ownership
// from another operator. An already dispatched response group stays dispatched.
func (p *PhoenixDataProvider) PickAlarm(ctx context.Context, alarm models.Alarm, user string) error {
	panelID := strings.TrimSpace(alarm.ObjectNumber)
	if panelID == "" {
		return fmt.Errorf("phoenix: PickAlarm: порожній panel_id")
	}
	_, operatorName, err := p.alarmOperatorIdentity()
	if err != nil {
		return err
	}
	eventID := p.activePhoenixAlarmEventID(ctx, panelID)
	if err := p.sendAlarmState(ctx, panelID, eventID, phoenixStateInWork, "", operatorName, false); err != nil {
		return fmt.Errorf("phoenix: PickAlarm %s: %w", panelID, err)
	}
	log.Debug().Str("panelID", panelID).Str("user", operatorName).Msg("Phoenix PickAlarm sent to Control Center")
	return nil
}

// GetAlarmProcessingOptions returns available next states for the alarm.
func (p *PhoenixDataProvider) GetAlarmProcessingOptions(ctx context.Context, alarm models.Alarm) ([]contracts.AlarmProcessingOption, error) {
	panelID := strings.TrimSpace(alarm.ObjectNumber)
	if panelID == "" {
		return nil, fmt.Errorf("phoenix: GetAlarmProcessingOptions: порожній panel_id")
	}

	currentState := phoenixStateInWork
	var stateVal sql.NullInt64
	if err := p.db.QueryRowContext(ctx,
		`SELECT TOP 1 StateEvent FROM Temp WITH (NOLOCK)
		 WHERE Panel_id = @p1 AND StateEvent IN (1, 2, 3)
		 ORDER BY StateEvent DESC`,
		panelID,
	).Scan(&stateVal); err == nil && stateVal.Valid {
		currentState = stateVal.Int64
	}

	var rows []phoenixAvailableStateRow
	if err := p.db.SelectContext(ctx, &rows, phoenixAvailableStatesQuery,
		currentState,
	); err != nil {
		return nil, fmt.Errorf("phoenix: GetAlarmProcessingOptions %s: %w", panelID, err)
	}

	opts := make([]contracts.AlarmProcessingOption, 0, len(rows))
	for _, row := range rows {
		if row.StateID <= 0 {
			continue
		}
		label := strings.TrimSpace(row.StateName)
		if label == "" {
			label = fmt.Sprintf("Стан %d", row.StateID)
		}
		opts = append(opts, contracts.AlarmProcessingOption{
			Code:  strconv.FormatInt(row.StateID, 10),
			Label: label,
		})
	}
	return opts, nil
}

// ProcessAlarmWithRequest closes the alarm using the operator-chosen state.
func (p *PhoenixDataProvider) ProcessAlarmWithRequest(ctx context.Context, alarm models.Alarm, user string, request contracts.AlarmProcessingRequest) error {
	panelID := strings.TrimSpace(alarm.ObjectNumber)
	if panelID == "" {
		return fmt.Errorf("phoenix: ProcessAlarmWithRequest: порожній panel_id")
	}

	stateID, err := strconv.ParseInt(strings.TrimSpace(request.CauseCode), 10, 64)
	if err != nil || stateID <= 0 {
		return fmt.Errorf("phoenix: ProcessAlarmWithRequest: невалідний stateID %q", request.CauseCode)
	}
	_, operatorName, err := p.alarmOperatorIdentity()
	if err != nil {
		return err
	}

	var eventID sql.NullInt64
	_ = p.db.QueryRowContext(ctx,
		`SELECT TOP 1 Event_id FROM Temp WITH (NOLOCK)
		 WHERE Panel_id = @p1 AND StateEvent IN (1, 2, 3)
		 ORDER BY StateEvent DESC, Event_id DESC`,
		panelID,
	).Scan(&eventID)
	if !eventID.Valid {
		return fmt.Errorf("phoenix: ProcessAlarmWithRequest %s: активна тривога не знайдена", panelID)
	}
	cause := phoenixFinishCause(ctx, p, alarm, request.CauseCode)
	note := strings.TrimSpace(request.Note)
	status := fmt.Sprintf("%d=%s\n %s\n", stateID, cause, note)
	if err := p.sendAlarmState(ctx, panelID, eventID.Int64, 4, status, operatorName, false); err != nil {
		return fmt.Errorf("phoenix: ProcessAlarmWithRequest %s: %w", panelID, err)
	}
	log.Debug().Str("panelID", panelID).Int64("stateID", stateID).Msg("Phoenix finish sent to Control Center")
	return nil
}

func phoenixFinishCause(
	ctx context.Context,
	p *PhoenixDataProvider,
	alarm models.Alarm,
	causeCode string,
) string {
	options, err := p.GetAlarmProcessingOptions(ctx, alarm)
	if err == nil {
		for _, option := range options {
			if strings.TrimSpace(option.Code) == strings.TrimSpace(causeCode) {
				if label := strings.TrimSpace(option.Label); label != "" {
					return label
				}
			}
		}
	}
	return "Причину вказано оператором"
}

func (p *PhoenixDataProvider) phoenixAlarmOwnershipError(ctx context.Context, panelID string, user string) error {
	var owner sql.NullString
	err := p.db.QueryRowContext(ctx,
		`SELECT TOP 1 Computer
		 FROM Temp WITH (NOLOCK)
		 WHERE Panel_id = @p1 AND StateEvent IN (1, 2, 3)
		 ORDER BY StateEvent DESC, Event_id DESC`,
		panelID,
	).Scan(&owner)
	if err != nil {
		return fmt.Errorf("phoenix: активна тривога %s не знайдена", panelID)
	}
	ownerText := strings.TrimSpace(nullString(owner))
	if ownerText == "" {
		ownerText = "інший оператор"
	}
	return fmt.Errorf("%w: %s (поточний власник: %s, оператор: %s)",
		contracts.ErrAlarmOwnershipConflict, panelID, ownerText, strings.TrimSpace(user))
}

// ProcessAlarm closes the alarm without a reason (legacy / cancel path).
func (p *PhoenixDataProvider) ProcessAlarm(id string, user string, note string) error {
	panelID, err := p.resolvePanelIDFromAlarmID(id)
	if err != nil {
		return fmt.Errorf("phoenix: ProcessAlarm: %w", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 8*time.Second)
	defer cancel()
	_, operatorName, err := p.alarmOperatorIdentity()
	if err != nil {
		return err
	}
	eventID := p.activePhoenixAlarmEventID(ctx, panelID)
	status := "21=Порушень немає\n " + strings.TrimSpace(note) + "\n"
	if err := p.sendAlarmState(ctx, panelID, eventID, 4, status, operatorName, false); err != nil {
		return fmt.Errorf("phoenix: ProcessAlarm %s: %w", panelID, err)
	}
	log.Debug().Str("panelID", panelID).Msg("Phoenix legacy finish sent to Control Center")
	return nil
}

// ListResponseGroups returns all Phoenix response groups with their current status.
func (p *PhoenixDataProvider) ListResponseGroups(ctx context.Context) ([]contracts.ResponseGroup, error) {
	if p == nil || p.db == nil {
		return nil, fmt.Errorf("phoenix: ListResponseGroups: база не ініціалізована")
	}

	var rows []phoenixResponseGroupRow
	if err := p.db.SelectContext(ctx, &rows, phoenixResponseGroupsQuery); err != nil {
		return nil, fmt.Errorf("phoenix: ListResponseGroups: %w", err)
	}

	groups := make([]contracts.ResponseGroup, 0, len(rows))
	for _, row := range rows {
		status := phoenixResponseGroupStatus(row.StatusID.Int64)
		groups = append(groups, contracts.ResponseGroup{
			ID:              strconv.FormatInt(row.GroupID, 10),
			Name:            strings.TrimSpace(row.Description),
			Callsign:        strings.TrimSpace(nullString(row.Callsign)),
			Source:          contracts.FrontendSourcePhoenix,
			Status:          status,
			StatusText:      responseGroupStatusText(status, strings.TrimSpace(nullString(row.StatusText))),
			ObjectNumber:    strings.TrimSpace(nullString(row.PanelID)),
			Latitude:        strings.TrimSpace(nullString(row.Latitude)),
			Longitude:       strings.TrimSpace(nullString(row.Longitude)),
			StatusChangedAt: nullTime(row.TimeArriveToObject),
		})
	}
	return groups, nil
}

func phoenixResponseGroupStatus(statusID int64) contracts.ResponseGroupStatus {
	switch statusID {
	case phoenixGroupStatusFree:
		return contracts.ResponseGroupStatusFree
	case phoenixGroupStatusDispatched:
		return contracts.ResponseGroupStatusDispatched
	case phoenixGroupStatusArrived:
		return contracts.ResponseGroupStatusArrived
	default:
		return contracts.ResponseGroupStatusUnknown
	}
}

func responseGroupStatusText(status contracts.ResponseGroupStatus, sourceText string) string {
	if sourceText = strings.TrimSpace(sourceText); sourceText != "" {
		return sourceText
	}
	switch status {
	case contracts.ResponseGroupStatusFree:
		return "Вільна"
	case contracts.ResponseGroupStatusDispatched:
		return "Направлена"
	case contracts.ResponseGroupStatusArrived:
		return "Прибула"
	default:
		return "Стан невідомий"
	}
}

// AssignResponseGroup dispatches a response group to the alarm (StateEvent 1→2).
func (p *PhoenixDataProvider) AssignResponseGroup(ctx context.Context, alarm models.Alarm, groupID string) error {
	panelID := strings.TrimSpace(alarm.ObjectNumber)
	if panelID == "" {
		return fmt.Errorf("phoenix: AssignResponseGroup: порожній panel_id")
	}
	gid, err := strconv.ParseInt(strings.TrimSpace(groupID), 10, 64)
	if err != nil || gid <= 0 {
		return fmt.Errorf("phoenix: AssignResponseGroup: невалідний groupID %q", groupID)
	}
	_, operatorName, err := p.alarmOperatorIdentity()
	if err != nil {
		return err
	}

	var eventID sql.NullInt64
	var owner sql.NullString
	if err := p.db.QueryRowContext(ctx,
		`SELECT TOP 1 Event_id, Computer FROM Temp WITH (NOLOCK)
		 WHERE Panel_id = @p1 AND StateEvent IN (1, 2, 3)
		 ORDER BY StateEvent DESC, Event_id DESC`,
		panelID,
	).Scan(&eventID, &owner); err != nil {
		return fmt.Errorf("phoenix: AssignResponseGroup %s: активна тривога не знайдена: %w", panelID, err)
	}
	if !eventID.Valid {
		return fmt.Errorf("phoenix: AssignResponseGroup %s: немає активної тривоги в Temp", panelID)
	}
	if !strings.EqualFold(strings.TrimSpace(nullString(owner)), operatorName) {
		return p.phoenixAlarmOwnershipError(ctx, panelID, operatorName)
	}

	if err := p.sendAlarmState(
		ctx, panelID, eventID.Int64, phoenixStateGroupSent,
		phoenixResponseGroupNotifyStatus(gid), operatorName, true,
	); err != nil {
		return fmt.Errorf("phoenix: AssignResponseGroup %s (group %d): %w", panelID, gid, err)
	}
	log.Debug().Str("panelID", panelID).Int64("groupID", gid).Int64("eventID", eventID.Int64).Msg("Phoenix assign group sent to Control Center")
	return nil
}

// NotifyGroupArrived marks response group as arrived at the object (status 2→3).
func (p *PhoenixDataProvider) NotifyGroupArrived(ctx context.Context, alarm models.Alarm) error {
	panelID := strings.TrimSpace(alarm.ObjectNumber)
	if panelID == "" {
		return fmt.Errorf("phoenix: NotifyGroupArrived: порожній panel_id")
	}
	_, operatorName, err := p.alarmOperatorIdentity()
	if err != nil {
		return err
	}
	eventID := p.activePhoenixAlarmEventID(ctx, panelID)
	groupID, err := strconv.ParseInt(strings.TrimSpace(alarm.ResponseGroupID), 10, 64)
	if err != nil || groupID <= 0 {
		return fmt.Errorf("phoenix: NotifyGroupArrived %s: не визначено ГМР", panelID)
	}
	if err := p.sendAlarmState(
		ctx, panelID, eventID, phoenixStateGroupArrived,
		phoenixResponseGroupNotifyStatus(groupID), operatorName, true,
	); err != nil {
		return fmt.Errorf("phoenix: NotifyGroupArrived %s: %w", panelID, err)
	}
	log.Debug().Str("panelID", panelID).Int64("groupID", groupID).Msg("Phoenix group arrival sent to Control Center")
	return nil
}

// CancelResponseGroup cancels a dispatched response group, returning it to free status.
func (p *PhoenixDataProvider) CancelResponseGroup(ctx context.Context, alarm models.Alarm) error {
	panelID := strings.TrimSpace(alarm.ObjectNumber)
	if panelID == "" {
		return fmt.Errorf("phoenix: CancelResponseGroup: порожній panel_id")
	}
	_, operatorName, err := p.alarmOperatorIdentity()
	if err != nil {
		return err
	}
	eventID := p.activePhoenixAlarmEventID(ctx, panelID)
	// Phoenix represents removing the response group as returning EVENT to the
	// in-work state, followed by STATUS_OBJECT so Control Center refreshes group state.
	if err := p.sendAlarmState(
		ctx, panelID, eventID, phoenixStateInWork, "", operatorName, true,
	); err != nil {
		return fmt.Errorf("phoenix: CancelResponseGroup %s: %w", panelID, err)
	}
	log.Debug().Str("panelID", panelID).Msg("Phoenix cancel group sent to Control Center")
	return nil
}

// resolvePanelIDFromAlarmID extracts Phoenix panel_id from the stable alarm ID.
func (p *PhoenixDataProvider) resolvePanelIDFromAlarmID(alarmID string) (string, error) {
	// Alarm ID may be numeric (registered object ID) or stable Phoenix ID.
	// Try as numeric object ID first.
	objID, err := strconv.Atoi(strings.TrimSpace(alarmID))
	if err == nil {
		if panelID, ok := p.panelByID[objID]; ok {
			return panelID, nil
		}
	}

	// alarmID might be a string panel_id directly.
	if strings.TrimSpace(alarmID) != "" {
		return strings.TrimSpace(alarmID), nil
	}
	return "", fmt.Errorf("не вдалося визначити panel_id з alarm ID %q", alarmID)
}
