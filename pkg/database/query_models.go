package database

import "time"

type ObjectInfoRow struct {
	// Поля з OBJECTS_INFO
	ObjUin        int64   `db:"OBJUIN"`
	Objn          int64   `db:"OBJN"`
	ObjFullName1  *string `db:"OBJFULLNAME1"`
	ObjShortName1 *string `db:"OBJSHORTNAME1"`
	Address1      *string `db:"ADDRESS1"`
	Contract1     *string `db:"CONTRACT1"`
	GsmPhone      *string `db:"GSMPHONE"`
	GsmPhone2     *string `db:"GSMPHONE2"`

	// Поля з OBJECTS_STATE
	AlarmState1     *int64 `db:"ALARMSTATE1"`
	GuardState1     *int64 `db:"GUARDSTATE1"`
	TechAlarmState1 *int64 `db:"TECHALARMSTATE1"`

	// Поля з OBJECTS_LA
	IsConnState1 *int64 `db:"ISCONNSTATE1"`
}

type ObjectDetailRow struct {
	// Поля з OBJECTS_INFO (oi)
	ObjUin        int64   `db:"OBJUIN"`
	Objn          int64   `db:"OBJN"`
	ObjFullName1  *string `db:"OBJFULLNAME1"`
	ObjShortName1 *string `db:"OBJSHORTNAME1"`
	Address1      *string `db:"ADDRESS1"`
	Contract1     *string `db:"CONTRACT1"`
	Phones1       *string `db:"PHONES1"`
	Notes1        *string `db:"NOTES1"`
	ReservText    *string `db:"RESERVTEXT"`
	GsmPhone      *string `db:"GSMPHONE"`
	GsmPhone2     *string `db:"GSMPHONE2"`
	Location1     *string `db:"LOCATION1"`

	// Поле з OBJTYPES (ot)
	ObjType1 *string `db:"OBJTYPE1"`

	// Поля стану
	AlarmState1     *int64 `db:"ALARMSTATE1"`
	TechAlarmState1 *int64 `db:"TECHALARMSTATE1"`
	IsConnState1    *int64 `db:"ISCONNSTATE1"`

	AkbState     *int64 `db:"AKBSTATE"`
	TestControl1 *int64 `db:"TESTCONTROL1"`
	TestTime1    *int64 `db:"TESTTIME1"`
	PowerFault   *int64 `db:"POWERFAULT"`

	PpkName *string `db:"PPKNAME"`

	ObjChan    *int    `db:"OBJCHAN"`
	PanelMark1 *string `db:"PANELMARK1"`
}

type ObjectDbPath struct {
	Objn   int64   `db:"OBJN"`
	Sbpdb  *string `db:"SBPDB"`
	SbType *int    `db:"SBTYPE"`
}

type TestMessageRow struct {
	MsgDTime1 *time.Time `db:"MSGDTIME1"`
	MsgInfo   *string    `db:"MSGINFO"`
	MsgText   *string    `db:"MSGTEXT"`
}

type TestControlRow struct {
	IsConnState   *int64     `db:"ISCONNSTATE"`
	LastTestTime1 *time.Time `db:"LASTTESTTIME1"`
	LastMessTime1 *time.Time `db:"LASTMESSTIME1"`
}

type EventRow struct {
	EvTime1       *time.Time `db:"EVTIME1"`
	ObjShortName1 *string    `db:"OBJSHORTNAME1"`
	ObjN          *int64     `db:"OBJN"`
	Zonen         *int64     `db:"ZONEN"`
	Ukr1          *string    `db:"UKR1"`
	Info1         *string    `db:"INFO1"`
	Sc1           *int       `db:"SC1"`
	ObjChan       *int       `db:"OBJCHAN"`
	ID            int64      `db:"ID"`
}

type ActAlarmsRow struct {
	EvTime1       *time.Time `db:"EVTIME1"`
	ObjN          *int64     `db:"OBJN"`
	ObjShortName1 *string    `db:"OBJSHORTNAME1"`
	Address1      *string    `db:"ADDRESS1"`
	Zonen         *int64     `db:"ZONEN"`
	Info1         *string    `db:"INFO1"`
	Ukr1          *string    `db:"UKR1"`
	Sc1           *int       `db:"SC1"`
}
