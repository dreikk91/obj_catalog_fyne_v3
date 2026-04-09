package database

import (
	"context"
	"fmt"

	"github.com/jmoiron/sqlx"
)

// GetActiveAlarmRowsByObject returns only currently active ACTALARMS rows for a single bridge object.
func GetActiveAlarmRowsByObject(ctx context.Context, db *sqlx.DB, objn int64) ([]ActAlarmsRow, error) {
	var results []ActAlarmsRow

	query := `
		SELECT
			a.EVTIME1,
			a.OBJN,
			a.ZONEN,
			a.INFO1,
			m.UKR1,
			m.SC1
		FROM ACTALARMS a
		JOIN MESSLIST m ON m.UIN = a.EVUIN
		WHERE a.OBJN = ?
		ORDER BY a.EVTIME1 DESC
	`

	if err := db.SelectContext(ctx, &results, db.Rebind(query), objn); err != nil {
		return nil, fmt.Errorf("failed to select active alarm rows: %w", err)
	}

	return results, nil
}
