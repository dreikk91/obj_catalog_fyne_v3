package database

import (
	"context"
	"fmt"

	"github.com/jmoiron/sqlx"
)

func GetObjectsList(ctx context.Context, db *sqlx.DB) ([]ObjectInfoRow, error) {
	var results []ObjectInfoRow

	query := `
		SELECT
			oi.OBJUIN, oi.OBJN, oi.OBJFULLNAME1, oi.OBJSHORTNAME1, oi.ADDRESS1, oi.CONTRACT1, 
			oi.ENG1, oi.GSMPHONE, oi.GSMPHONE2, oi.RESERVLONG2, oi.SBSA, oi.SBSB,
			os.ALARMSTATE1, os.GUARDSTATE1, os.TECHALARMSTATE1,
			os.BLOCKEDARMED_ON_OFF,
			ol.ISCONNSTATE1
		FROM OBJECTS_INFO oi
		JOIN OBJECTS_LA ol ON ol.OBJUIN = oi.OBJUIN
		JOIN OBJECTS_STATE os ON os.OBJUIN = oi.OBJUIN
		WHERE oi.OBJTYPEID <> 1
		ORDER BY CAST(oi.OBJN AS VARCHAR(20))
	`

	err := db.SelectContext(ctx, &results, db.Rebind(query))
	if err != nil {
		return nil, fmt.Errorf("failed to select objects: %w", err)
	}

	return results, nil
}

// GetAlarmsList отримує тільки ті об'єкти, які знаходяться в стані тривоги
func GetAlarmsList(ctx context.Context, db *sqlx.DB) ([]ActAlarmsRow, error) {
	var results []ActAlarmsRow

	query := `
		SELECT a.EVTIME1, a.OBJN, a.OBJSHORTNAME1, oi.ADDRESS1, a.ZONEN, m.UKR1, a.INFO1, m.SC1      
		FROM ACTALARMS a 
		JOIN MESSLIST m ON m.UIN = a.EVUIN 
		JOIN OBJECTS_INFO oi ON a.OBJN = oi.OBJN 
	`

	err := db.SelectContext(ctx, &results, db.Rebind(query))
	if err != nil {
		return nil, fmt.Errorf("failed to select alarms: %w", err)
	}

	return results, nil
}

// GetActiveAlarmEvents returns event chronology for objects that currently exist in ACTALARMS.
func GetActiveAlarmEvents(ctx context.Context, db *sqlx.DB) ([]ActiveAlarmEventRow, error) {
	var results []ActiveAlarmEventRow

	query := `
		WITH ACTIVE_OBJECTS AS (
			SELECT
				a.OBJN,
				MIN(a.EVTIME1) AS START_TIME
			FROM ACTALARMS a
			GROUP BY a.OBJN
		)
		SELECT
			e.EVTIME1,
			ao.OBJN,
			e.ZONEN,
			e.INFO1,
			m.UKR1,
			m.SC1
		FROM ACTIVE_OBJECTS ao
		JOIN OBJECTS_INFO oi ON oi.OBJN = ao.OBJN
		JOIN EVLOG e ON e.OBJUIN = oi.OBJUIN AND e.EVTIME1 >= ao.START_TIME
		JOIN MESSLIST m ON m.UIN = e.EVUIN
		ORDER BY ao.OBJN, e.EVTIME1 DESC
	`

	err := db.SelectContext(ctx, &results, db.Rebind(query))
	if err != nil {
		return nil, fmt.Errorf("failed to select active alarm events: %w", err)
	}

	return results, nil
}
