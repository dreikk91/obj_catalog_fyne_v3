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
	phoenixStateActive    = int64(1) // alarm is active, not yet picked up
	phoenixStateInWork    = int64(2) // operator took the alarm
	phoenixStateGroupSent = int64(3) // response group dispatched
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

	result, err := p.db.ExecContext(ctx,
		`UPDATE Temp
		 SET StateEvent = CASE WHEN StateEvent = @p1 THEN @p2 ELSE StateEvent END,
		     Computer = @p3
		 WHERE Panel_id = @p4 AND StateEvent IN (@p1, @p2, @p5)`,
		phoenixStateActive, phoenixStateInWork, strings.TrimSpace(user), panelID, phoenixStateGroupSent,
	)
	if err != nil {
		return fmt.Errorf("phoenix: PickAlarm %s: %w", panelID, err)
	}
	n, _ := result.RowsAffected()
	if n == 0 {
		return p.phoenixAlarmOwnershipError(ctx, panelID, user)
	}
	log.Debug().Str("panelID", panelID).Str("user", user).Int64("rows", n).Msg("Phoenix PickAlarm")
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

	// Get event_id before changing state (for ArchiveAdditionInfo).
	var eventID sql.NullInt64
	_ = p.db.QueryRowContext(ctx,
		`SELECT TOP 1 Event_id FROM Temp WITH (NOLOCK)
		 WHERE Panel_id = @p1 AND StateEvent IN (1, 2, 3)
		 ORDER BY StateEvent DESC, Event_id DESC`,
		panelID,
	).Scan(&eventID)

	result, err := p.db.ExecContext(ctx,
		`UPDATE Temp SET StateEvent = @p1, Computer = @p2
		 WHERE Panel_id = @p3
		   AND StateEvent IN (1, 2, 3)
		   AND NOT EXISTS (
				SELECT 1
				FROM Temp ownerRow
				WHERE ownerRow.Panel_id = @p3
				  AND ownerRow.StateEvent IN (2, 3)
				  AND ownerRow.Computer IS NOT NULL
				  AND LTRIM(RTRIM(ownerRow.Computer)) <> ''
				  AND ownerRow.Computer <> @p2
		   )
		   AND (
				StateEvent = 1
				OR Computer IS NULL
				OR LTRIM(RTRIM(Computer)) = ''
				OR Computer = @p2
		   )`,
		stateID, user, panelID,
	)
	if err != nil {
		return fmt.Errorf("phoenix: ProcessAlarmWithRequest %s: %w", panelID, err)
	}
	n, _ := result.RowsAffected()
	if n == 0 {
		return p.phoenixAlarmOwnershipError(ctx, panelID, user)
	}
	log.Debug().Str("panelID", panelID).Int64("stateID", stateID).Int64("rows", n).Msg("Phoenix ProcessAlarmWithRequest")

	// Attach operator note to the archived event.
	note := strings.TrimSpace(request.Note)
	if note != "" && eventID.Valid && eventID.Int64 > 0 {
		if _, err := p.db.ExecContext(ctx,
			`INSERT INTO ArchiveAdditionInfo (ArchiveEventID, info) VALUES (@p1, @p2)`,
			eventID.Int64, note,
		); err != nil {
			log.Warn().Err(err).Int64("eventID", eventID.Int64).Msg("Phoenix: не вдалося зберегти примітку до архіву")
		}
	}
	return nil
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

	// Use state 5 (first non-active available state) for quick cancellation.
	// State 5 is the typical "Скасування" state in Phoenix AvailableStates for idTCode=1.
	cancelState := int64(5)

	result, err := p.db.ExecContext(ctx,
		`UPDATE Temp SET StateEvent = @p1, Computer = @p2
		 WHERE Panel_id = @p3 AND StateEvent IN (1, 2, 3)`,
		cancelState, user, panelID,
	)
	if err != nil {
		return fmt.Errorf("phoenix: ProcessAlarm %s: %w", panelID, err)
	}
	n, _ := result.RowsAffected()
	log.Debug().Str("panelID", panelID).Int64("rows", n).Msg("Phoenix ProcessAlarm (cancel)")

	if note != "" {
		var eventID sql.NullInt64
		_ = p.db.QueryRowContext(ctx,
			`SELECT TOP 1 Event_id FROM Temp WITH (NOLOCK)
			 WHERE Panel_id = @p1 AND StateEvent = @p2
			 ORDER BY Event_id DESC`,
			panelID, cancelState,
		).Scan(&eventID)
		if eventID.Valid && eventID.Int64 > 0 {
			_, _ = p.db.ExecContext(ctx,
				`INSERT INTO ArchiveAdditionInfo (ArchiveEventID, info) VALUES (@p1, @p2)`,
				eventID.Int64, note,
			)
		}
	}
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

// AssignResponseGroup dispatches a response group to the alarm (GroupResponse status 1→2, StateEvent 2→3).
func (p *PhoenixDataProvider) AssignResponseGroup(ctx context.Context, alarm models.Alarm, groupID string) error {
	panelID := strings.TrimSpace(alarm.ObjectNumber)
	if panelID == "" {
		return fmt.Errorf("phoenix: AssignResponseGroup: порожній panel_id")
	}
	gid, err := strconv.ParseInt(strings.TrimSpace(groupID), 10, 64)
	if err != nil || gid <= 0 {
		return fmt.Errorf("phoenix: AssignResponseGroup: невалідний groupID %q", groupID)
	}

	// Find current active event_id and group_ for this panel.
	var eventID sql.NullInt64
	var groupNo sql.NullInt64
	if err := p.db.QueryRowContext(ctx,
		`SELECT TOP 1 Event_id, Group_ FROM Temp WITH (NOLOCK)
		 WHERE Panel_id = @p1 AND StateEvent IN (1, 2, 3)
		 ORDER BY StateEvent DESC, Event_id DESC`,
		panelID,
	).Scan(&eventID, &groupNo); err != nil {
		return fmt.Errorf("phoenix: AssignResponseGroup %s: активна тривога не знайдена: %w", panelID, err)
	}
	if !eventID.Valid {
		return fmt.Errorf("phoenix: AssignResponseGroup %s: немає активної тривоги в Temp", panelID)
	}

	grp := int64(1)
	if groupNo.Valid && groupNo.Int64 > 0 {
		grp = groupNo.Int64
	}

	// Link response group to the alarm event.
	if _, err := p.db.ExecContext(ctx,
		`UPDATE GroupResponse
		 SET Event_id = @p1, Panel_id = @p2, Group_ = @p3, Status_id = @p4
		 WHERE Group_id = @p5`,
		eventID.Int64, panelID, grp, phoenixGroupStatusDispatched, gid,
	); err != nil {
		return fmt.Errorf("phoenix: AssignResponseGroup %s (group %d): %w", panelID, gid, err)
	}

	// Advance alarm state to "group dispatched".
	if _, err := p.db.ExecContext(ctx,
		`UPDATE Temp SET StateEvent = @p1
		 WHERE Panel_id = @p2 AND StateEvent IN (1, 2)`,
		phoenixStateGroupSent, panelID,
	); err != nil {
		log.Warn().Err(err).Str("panelID", panelID).Msg("Phoenix: не вдалося перевести StateEvent=3")
	}

	log.Debug().Str("panelID", panelID).Int64("groupID", gid).Int64("eventID", eventID.Int64).Msg("Phoenix AssignResponseGroup")
	return nil
}

// NotifyGroupArrived marks response group as arrived at the object (status 2→3).
func (p *PhoenixDataProvider) NotifyGroupArrived(ctx context.Context, alarm models.Alarm) error {
	panelID := strings.TrimSpace(alarm.ObjectNumber)
	if panelID == "" {
		return fmt.Errorf("phoenix: NotifyGroupArrived: порожній panel_id")
	}

	result, err := p.db.ExecContext(ctx,
		`UPDATE GroupResponse
		 SET Status_id = @p1, TimeArriveToObject = GETDATE()
		 WHERE Panel_id = @p2 AND Status_id = @p3`,
		phoenixGroupStatusArrived, panelID, phoenixGroupStatusDispatched,
	)
	if err != nil {
		return fmt.Errorf("phoenix: NotifyGroupArrived %s: %w", panelID, err)
	}
	n, _ := result.RowsAffected()
	log.Debug().Str("panelID", panelID).Int64("rows", n).Msg("Phoenix NotifyGroupArrived")
	return nil
}

// CancelResponseGroup cancels a dispatched response group, returning it to free status.
func (p *PhoenixDataProvider) CancelResponseGroup(ctx context.Context, alarm models.Alarm) error {
	panelID := strings.TrimSpace(alarm.ObjectNumber)
	if panelID == "" {
		return fmt.Errorf("phoenix: CancelResponseGroup: порожній panel_id")
	}

	result, err := p.db.ExecContext(ctx,
		`UPDATE GroupResponse
		 SET Status_id = @p1, Event_id = NULL, Panel_id = NULL, Group_ = NULL,
		     TimeArriveToObject = NULL, TimeStayOnObj = NULL
		 WHERE Panel_id = @p2 AND Status_id IN (@p3, @p4)`,
		phoenixGroupStatusFree, panelID, phoenixGroupStatusDispatched, phoenixGroupStatusArrived,
	)
	if err != nil {
		return fmt.Errorf("phoenix: CancelResponseGroup %s: %w", panelID, err)
	}
	n, _ := result.RowsAffected()

	// Revert alarm state if group was the only dispatch.
	if n > 0 {
		if _, err := p.db.ExecContext(ctx,
			`UPDATE Temp SET StateEvent = @p1
			 WHERE Panel_id = @p2 AND StateEvent = @p3`,
			phoenixStateInWork, panelID, phoenixStateGroupSent,
		); err != nil {
			log.Warn().Err(err).Str("panelID", panelID).Msg("Phoenix: не вдалося відкотити StateEvent=2")
		}
	}

	log.Debug().Str("panelID", panelID).Int64("rows", n).Msg("Phoenix CancelResponseGroup")
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
