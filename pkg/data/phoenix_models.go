package data

import (
	"database/sql"
	"time"
)

type phoenixObjectGroupRow struct {
	PanelID       string         `db:"panel_id"`
	GroupNo       int            `db:"group_no"`
	GroupName     sql.NullString `db:"group_name"`
	IsOpen        sql.NullBool   `db:"is_open"`
	GroupTime     sql.NullTime   `db:"group_time_event"`
	GroupDisabled sql.NullBool   `db:"group_disabled"`
	CompanyName   sql.NullString `db:"company_name"`
	Address       sql.NullString `db:"company_address"`
	Telephones    sql.NullString `db:"telephones"`
	TypeName      sql.NullString `db:"type_name"`
	PanelDisabled sql.NullBool   `db:"panel_disabled"`
	TestPanel     sql.NullBool   `db:"test_panel"`
	PanelType     sql.NullInt64  `db:"panel_type"`
	StateEvent    sql.NullInt64  `db:"state_event"`
	CreateDate    sql.NullTime   `db:"create_date"`
	LastChange    sql.NullTime   `db:"date_last_change"`
	EngineerName  sql.NullString `db:"engineer_name"`
}

type phoenixChannelRow struct {
	PanelID               string         `db:"panel_id"`
	DeviceName            sql.NullString `db:"device_name"`
	ChannelType           sql.NullString `db:"channel_type"`
	ChannelNo             sql.NullString `db:"channel_no"`
	LastTest              sql.NullTime   `db:"last_test"`
	TestTimeout           sql.NullTime   `db:"test_timeout"`
	OpenInternetChannelID sql.NullInt64  `db:"open_internet_channel_id"`
	SignalLevel           sql.NullInt64  `db:"signal_level"`
	DeviceVersion         sql.NullInt64  `db:"device_version"`
	RadioVersion          sql.NullString `db:"radio_version"`
	Sim1Number            sql.NullString `db:"sim1_number"`
	Sim1OperatorName      sql.NullString `db:"sim1_operator_name"`
	Sim2Number            sql.NullString `db:"sim2_number"`
	Sim2OperatorName      sql.NullString `db:"sim2_operator_name"`
}

type phoenixZoneRow struct {
	PanelID         string         `db:"panel_id"`
	GroupNo         int            `db:"group_no"`
	GroupName       sql.NullString `db:"group_name"`
	GroupIsOpen     sql.NullBool   `db:"group_is_open"`
	GroupDisabled   sql.NullBool   `db:"group_disabled"`
	PanelDisabled   sql.NullBool   `db:"panel_disabled"`
	TestPanel       sql.NullBool   `db:"test_panel"`
	GroupTime       sql.NullTime   `db:"group_time_event"`
	ZoneNo          int            `db:"zone_no"`
	ZoneName        sql.NullString `db:"zone_name"`
	Status          sql.NullInt64  `db:"status"`
	IsPatrol        sql.NullBool   `db:"is_patrol"`
	IsAlarmButton   sql.NullBool   `db:"is_alarm_button"`
	IsBypass        sql.NullBool   `db:"is_bypass"`
	SignalLevel     sql.NullInt64  `db:"signal_level"`
	RadioZoneTypeID sql.NullInt64  `db:"zone_type_id"`
}

type phoenixResponsibleRow struct {
	PanelID         string         `db:"panel_id"`
	GroupNo         int            `db:"group_no"`
	GroupName       sql.NullString `db:"group_name"`
	GroupIsOpen     sql.NullBool   `db:"group_is_open"`
	GroupDisabled   sql.NullBool   `db:"group_disabled"`
	PanelDisabled   sql.NullBool   `db:"panel_disabled"`
	TestPanel       sql.NullBool   `db:"test_panel"`
	ResponsibleNo   sql.NullInt64  `db:"responsible_number"`
	ResponsibleName sql.NullString `db:"responsible_name"`
	ResponsibleAddr sql.NullString `db:"responsible_address"`
	CallOrder       sql.NullInt64  `db:"call_order"`
	ContactLabel    sql.NullString `db:"contact_label"`
	ContactValue    sql.NullString `db:"contact_value"`
	ContactKind     sql.NullString `db:"contact_kind"`
}

type phoenixEventRow struct {
	EventID     int64          `db:"event_id"`
	PanelID     string         `db:"panel_id"`
	GroupNo     sql.NullInt64  `db:"group_no"`
	ZoneNo      sql.NullInt64  `db:"zone_no"`
	TimeEvent   time.Time      `db:"time_event"`
	EventCode   sql.NullString `db:"event_code"`
	CodeMessage sql.NullString `db:"code_message"`
	TypeCodeID  sql.NullInt64  `db:"type_code_id"`
	GroupName   sql.NullString `db:"group_name"`
	ZoneName    sql.NullString `db:"zone_name"`
	CompanyName sql.NullString `db:"company_name"`
	Address     sql.NullString `db:"company_address"`
}

type phoenixActiveAlarmRow struct {
	EventID       sql.NullInt64  `db:"event_id"`
	PanelID       string         `db:"panel_id"`
	GroupNo       int            `db:"group_no"`
	ZoneNo        sql.NullInt64  `db:"zone_no"`
	TimeEvent     sql.NullTime   `db:"time_event"`
	EventCode     sql.NullString `db:"event_code"`
	CodeMessage   sql.NullString `db:"code_message"`
	TypeCodeID    sql.NullInt64  `db:"type_code_id"`
	TypeMessage   sql.NullString `db:"type_code_message"`
	GroupName     sql.NullString `db:"group_name"`
	GroupMessage  sql.NullString `db:"group_message"`
	ZoneName      sql.NullString `db:"zone_name"`
	CompanyName   sql.NullString `db:"company_name"`
	Address       sql.NullString `db:"company_address"`
	Line          sql.NullString `db:"line"`
	EventParentID sql.NullInt64  `db:"event_parent_id"`
	StateEvent    sql.NullInt64  `db:"state_event"`
	Priority      sql.NullInt64  `db:"priority"`
	ObjectStatus  sql.NullInt64  `db:"object_status"`
	UnknownObject sql.NullInt64  `db:"unknown_object"`
	IsAlarmButton sql.NullBool   `db:"is_alarm_button"`
	GroupDisabled sql.NullBool   `db:"group_disabled"`
	PanelDisabled sql.NullBool   `db:"panel_disabled"`
	TestPanel     sql.NullBool   `db:"test_panel"`
}
