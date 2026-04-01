package phoenix

import (
	"time"
)

// ObjectInfoRow представляє рядок результату запиту до vwRealPanel
type ObjectInfoRow struct {
	PanelID     string     `db:"panel_id"`
	GroupNum    int        `db:"group_"`
	Message     *string    `db:"message"`
	Address     *string    `db:"Address"`
	CompanyName *string    `db:"CompanyName"`
	Status      int        `db:"status"`
	StatusTime  *time.Time `db:"status_time"`
	ObjectID    int        `db:"ObjectID"` // Уточнення: потрібен числовий ID для внутрішніх структур
}

// ArchiveRow представляє рядок результату запиту до vwArchives
type ArchiveRow struct {
	EventID    int       `db:"Event_id"`
	PanelID    string    `db:"Panel_id"`
	TimeEvent  time.Time `db:"TimeEvent"`
	Group      int       `db:"Group_"`
	Message    *string   `db:"Message"`
	CodeMessage *string  `db:"CodeMessage"`
	IDTCode    int       `db:"idTCode"`
	Zone       *int      `db:"Zone"`
}

// AlarmRow представляє рядок результату запиту до CurrentAlarms
type AlarmRow struct {
	ID        int       `db:"ID"`
	PanelID   string    `db:"Panel_id"`
	Group     int       `db:"Group_"`
	Zone      int       `db:"Zone"`
	Code      string    `db:"Code"`
	TimeEvent time.Time `db:"TimeEvent"`
	Text      *string   `db:"Text"`
}
