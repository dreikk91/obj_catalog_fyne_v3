package phoenix

import (
	"context"
	"fmt"

	"github.com/jmoiron/sqlx"
)

// GetObjectsList отримує список об'єктів з Phoenix (MSSQL)
func GetObjectsList(ctx context.Context, db *sqlx.DB) ([]ObjectInfoRow, error) {
	var results []ObjectInfoRow

	query := `
		SELECT
			RealPanel.panel_id,
			groups.group_,
			groups.message,
			Company.Address AS Address,
			Company.CompanyName,
			Groups.isOpen AS [status],
			Groups.TimeEvent AS status_time
		FROM vwRealPanel AS RealPanel WITH (NOLOCK)
		LEFT OUTER JOIN Groups WITH(NOLOCK) ON RealPanel.Realpanel_id = groups.panel_id
		LEFT OUTER JOIN company WITH(NOLOCK) ON groups.companyID = company.ID
		ORDER BY RealPanel.panel_id, groups.group_
	`

	err := db.SelectContext(ctx, &results, db.Rebind(query))
	if err != nil {
		return nil, fmt.Errorf("failed to select phoenix objects: %w", err)
	}

	return results, nil
}

// GetAlarmsList отримує список активних тривог з Phoenix
func GetAlarmsList(ctx context.Context, db *sqlx.DB) ([]AlarmRow, error) {
	var results []AlarmRow

	query := `
		SELECT ID, Panel_id, Group_, Zone, Code, TimeEvent, Text
		FROM CurrentAlarms WITH (NOLOCK)
	`

	err := db.SelectContext(ctx, &results, db.Rebind(query))
	if err != nil {
		return nil, fmt.Errorf("failed to select phoenix alarms: %w", err)
	}

	return results, nil
}
