package database

import (
	"context"
	"fmt"

	"github.com/jmoiron/sqlx"
)

// GetObjectEmployees fetches employees for a given object number (OBJN)
func GetObjectEmployees(ctx context.Context, db *sqlx.DB, objUin int64) ([]Personal, error) {
	var results []Personal
	query := `
		SELECT 
			ID, SURNAME1, NAME1, SECNAME1, ADDRESS1, PHONES1, STATUS1
		FROM 
			PERSONAL
		WHERE 
			OBJUIN = ?
		ORDER BY 
			ORDER1
	`
	// Using Rebind for driver compatibility if needed, though ? works for Firebird usually
	// (sqlx might need named args or ? depending on driver. Assuming ? is fine for now or db.Rebind)
	// Actually sqlx.Select handles slice scanning.
	// Firebird driver often uses ? or @p1.
	err := db.SelectContext(ctx, &results, db.Rebind(query), objUin)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch employees: %w", err)
	}
	return results, nil
}

// GetObjectZones fetches zones for a given object number (OBJN)
func GetObjectZones(ctx context.Context, db *sqlx.DB, objn int64) ([]Zone, error) {
	var results []Zone
	query := `
		SELECT 
			ID, ZONEN, ZONEDESCR1, ZONETYPE1, ALARMSTATE1, TECHALARMSTATE1
		FROM 
			ZONES
		WHERE 
			OBJN = ?
		ORDER BY 
			ZONEN
	`
	err := db.SelectContext(ctx, &results, db.Rebind(query), objn)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch zones: %w", err)
	}
	return results, nil
}

// GetObjectEvents fetches recent events for a given object number (OBJN)
func GetObjectEvents(ctx context.Context, db *sqlx.DB, objuinInt int64) ([]EventRow, error) {
	var results []EventRow
	// Simple query for last 20 events
	query := `
		SELECT e.EVTIME1, e.ZONEN, e.INFO1, m.UKR1, m.SC1   
		FROM EVLOG e
		JOIN MESSLIST m ON m.UIN = e.EVUIN 
		WHERE e.OBJUIN = ?
		ORDER BY e.EVTIME1 DESC 
	`
	err := db.SelectContext(ctx, &results, db.Rebind(query), objuinInt)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch events: %w", err)
	}
	return results, nil
}

// GetGlobalEvents fetches the most recent global events
// GetGlobalEvents fetches the most recent global events starting from lastID
func GetGlobalEvents(ctx context.Context, db *sqlx.DB, lastID int64) ([]EventRow, error) {
	var results []EventRow
	query := `
		SELECT
			e.EVTIME1,
			oi.OBJSHORTNAME1,
			oi.OBJN,
			e.ZONEN,
			m.UKR1,
			e.INFO1,
			m.SC1,
			oi.OBJCHAN,
			e.ID
		FROM (
			SELECT ID, EVTIME1, OBJUIN, EVUIN, ZONEN, INFO1
			FROM EVLOG
			WHERE ID > ?
			ORDER BY ID
			ROWS 2000
		) e
		JOIN OBJECTS_INFO oi ON oi.OBJUIN = e.OBJUIN
		JOIN MESSLIST m ON m.UIN = e.EVUIN
		WHERE EXISTS (
			SELECT 1
			FROM OBJECTS_STATE os
			WHERE os.OBJUIN = oi.OBJUIN
		)
		ORDER BY e.ID
	`
	err := db.SelectContext(ctx, &results, db.Rebind(query), lastID)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch global events: %w", err)
	}
	return results, nil
}

// GetLastEventID повертає ID останньої події в базі
func GetLastEventID(ctx context.Context, db *sqlx.DB) (int64, error) {
	var lastID int64
	query := `SELECT FIRST 1 ID FROM EVLOG ORDER BY ID DESC`
	err := db.GetContext(ctx, &lastID, query)
	if err != nil {
		return 0, fmt.Errorf("failed to get last event id: %w", err)
	}
	return lastID, nil
}

type ObjectDetail struct {
	ObjectInfo
	// Add other fields if needed from other tables
}

func GetObjectDetail(ctx context.Context, db *sqlx.DB, objn int64) (*ObjectDetailRow, error) {
	var result ObjectDetailRow

	query := `
		SELECT
			oi.OBJUIN,
			oi.OBJN,
			oi.OBJFULLNAME1,
			oi.OBJSHORTNAME1,
			oi.ADDRESS1,
			oi.CONTRACT1,
			oi.PHONES1,
			oi.NOTES1,
			oi.RESERVTEXT,
			oi.GSMPHONE,
			oi.GSMPHONE2,
			oi.LOCATION1,
			ot.OBJTYPE1,
			os.ALARMSTATE1, os.TECHALARMSTATE1,
			os.AKBSTATE, os.TESTCONTROL1, os.TESTTIME1, os.POWERFAULT,
			ol.ISCONNSTATE1,
			oi.OBJCHAN,
			p.PANELMARK1 
		FROM
			OBJECTS_INFO oi
		LEFT JOIN OBJTYPES ot ON
			ot.ID = oi.OBJTYPEID
		LEFT JOIN OBJECTS_STATE os ON os.OBJUIN = oi.OBJUIN
		LEFT JOIN OBJECTS_LA ol ON ol.OBJUIN = oi.OBJUIN
		LEFT JOIN PPK p ON oi.PPKID = p.ID + 100
		WHERE
			oi.OBJN = ?
			`

	err := db.GetContext(ctx, &result, query, objn)
	if err != nil {
		return nil, fmt.Errorf("failed to select object detail: %w", err)
	}

	return &result, nil
}

func GetObjectDbPath(ctx context.Context, db *sqlx.DB, objn int64) (*ObjectDbPath, error) {
	var result ObjectDbPath
	query := `
		SELECT oi.OBJN, sb.SBPDB, sb.SBTYPE  
		FROM OBJECTS_INFO oi
		JOIN SBS sb ON TRIM(oi.SBSA) = TRIM(sb.SBIND)
		WHERE oi.OBJN = ?
	`
	err := db.GetContext(ctx, &result, query, objn)
	if err != nil {
		return nil, err
	}
	return &result, nil
}

func GetTestMessages(ctx context.Context, db *sqlx.DB, objn int64) ([]TestMessageRow, error) {
	var results []TestMessageRow
	query := `
		SELECT t.MSGDTIME1, t.MSGINFO, t.MSGTEXT   
		FROM TRPMSG t 
		WHERE t.OBJN = ?
		ORDER BY t.MSGDTIME1 DESC
	`
	err := db.SelectContext(ctx, &results, query, objn)
	if err != nil {
		return nil, err
	}
	return results, nil
}

func GetTestControl(ctx context.Context, db *sqlx.DB, objn int64) (*TestControlRow, error) {
	var result TestControlRow
	query := `
		SELECT tt.ISCONNSTATE, tt.LASTTESTTIME1, tt.LASTMESSTIME1   
		FROM TBL_TESTCONTROL tt 
		WHERE tt.OBJN = ?
	`
	err := db.GetContext(ctx, &result, query, objn)
	if err != nil {
		return nil, err
	}
	return &result, nil
}
