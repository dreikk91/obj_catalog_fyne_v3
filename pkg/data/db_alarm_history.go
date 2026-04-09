package data

import (
	"context"
	"strconv"
	"time"

	"obj_catalog_fyne_v3/pkg/database"
	"obj_catalog_fyne_v3/pkg/models"

	"github.com/rs/zerolog/log"
)

func (p *DBDataProvider) GetActiveAlarmSourceMessages(alarm models.Alarm) []models.AlarmMsg {
	if p == nil || p.db == nil {
		return nil
	}

	objn := int64(alarm.ObjectID)
	if objn <= 0 {
		if parsed, err := strconv.ParseInt(alarm.ObjectNumber, 10, 64); err == nil && parsed > 0 {
			objn = parsed
		}
	}
	if objn <= 0 {
		return nil
	}

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	rows, err := database.GetActiveAlarmRowsByObject(ctx, p.db, objn)
	if err != nil {
		log.Error().Err(err).Int64("objn", objn).Msg("Помилка отримання активної хронології з ACTALARMS")
		return nil
	}

	return buildDBActiveAlarmSourceMessages(rows)
}

func buildDBActiveAlarmSourceMessages(rows []database.ActAlarmsRow) []models.AlarmMsg {
	if len(rows) == 0 {
		return nil
	}

	messages := make([]dbAlarmMessage, 0, len(rows))
	for _, row := range rows {
		messages = append(messages, buildDBAlarmMessageFromActiveRow(row))
	}
	sortDBAlarmMessages(messages)
	return mapDBAlarmMessagesToSourceMsgs(messages)
}
