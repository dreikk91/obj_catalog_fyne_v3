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
			oi.GSMPHONE, oi.GSMPHONE2, 
			os.ALARMSTATE1, os.GUARDSTATE1, os.TECHALARMSTATE1,
			ol.ISCONNSTATE1
		FROM OBJECTS_INFO oi
		JOIN OBJECTS_LA ol ON ol.OBJN = oi.OBJN
		JOIN OBJECTS_STATE os ON os.OBJN = oi.OBJN
		WHERE oi.OBJTYPEID <> 1
		ORDER BY oi.OBJN
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
		SELECT a.EVTIME1, a.OBJN, a.OBJSHORTNAME1, oi.ADDRESS1, m.UKR1, a.INFO1, m.SC1      
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
