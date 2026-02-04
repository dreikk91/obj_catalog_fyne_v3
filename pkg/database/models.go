package database


import (
	"time"
)

// EvLog - модель для таблиці EVLOG
type EvLog struct {
	ID                 int64      `db:"ID"`
	EvTime1            *time.Time `db:"EVTIME1"`
	ObjUin             *int64     `db:"OBJUIN"`
	Grpn               *int64     `db:"GRPN"`
	Zonen              *int64     `db:"ZONEN"`
	EvUin              *int64     `db:"EVUIN"`
	Info1              *string    `db:"INFO1"`
	ResByte1           *int64     `db:"RESBYTE1"`
	ResBigInt1         *int64     `db:"RESBIGINT1"`
	ResBigInt2         *int64     `db:"RESBIGINT2"`
	ResVarChar1        *string    `db:"RESVARCHAR1"`
	ResVarChar2        *string    `db:"RESVARCHAR2"`
	FirstActDTime      *time.Time `db:"FIRSTACTDTIME"`
	FirstActWasAlerted *int16     `db:"FIRSTACTWASALERTED"`
	Xonum              int64      `db:"XONUM"`
}

// MessList - модель для таблиці MESSLIST
type MessList struct {
	Uin         int64   `db:"UIN"`
	ProtId      *int64  `db:"PROTID"`
	MessId      *int64  `db:"MESSID"`
	Rus1        *string `db:"RUS1"`
	Eng1        *string `db:"ENG1"`
	Sc1         *int64  `db:"SC1"`
	Objn        *int64  `db:"OBJN"`
	Grpn        *int64  `db:"GRPN"`
	Zonen       *int16  `db:"ZONEN"`
	ResByte1    *int64  `db:"RESBYTE1"`
	Ukr1        *string `db:"UKR1"`
	MessIdHex   *string `db:"MESSIDHEX"`
	ResVarChar1 *string `db:"RESVARCHAR1"`
	ForAdminOnly *int16 `db:"FORADMINONLY"`
	Mc220v      int16   `db:"MC220V"`
}

// ObjType - модель для таблиці OBJTYPES
type ObjType struct {
	ID            int64   `db:"ID"`
	ObjType1      *string `db:"OBJTYPE1"`
	ResByte1      *int64  `db:"RESBYTE1"`
	BankObjType   int16   `db:"BANK_OBJ_TYPE"`
	BankMfo       string  `db:"BANK_MFO"`
}

// ObjectGps - модель для таблиці OBJECTS_GPS
type ObjectGps struct {
	ID        *int64  `db:"ID"`
	ObjUin    *int64  `db:"OBJUIN"`
	Objn      *int64  `db:"OBJN"`
	Grpn      *int64  `db:"GRPN"`
	Latitude  *string `db:"LATITUDE"`
	Longitude *string `db:"LONGITUDE"`
	NavId     *int64  `db:"NAV_ID"`
}

// ObjectInfo - модель для таблиці OBJECTS_INFO
type ObjectInfo struct {
	ObjUin          int64      `db:"OBJUIN"`
	Objn            *int64     `db:"OBJN"`
	Grpn            *int64     `db:"GRPN"`
	ObjFullName1    *string    `db:"OBJFULLNAME1"`
	ObjShortName1   *string    `db:"OBJSHORTNAME1"`
	ObjTypeId       *int64     `db:"OBJTYPEID"`
	Address1        *string    `db:"ADDRESS1"`
	Phones1         *string    `db:"PHONES1"`
	ObjRegId        *int16     `db:"OBJREGID"`
	Contract1       *string    `db:"CONTRACT1"`
	Location1       *string    `db:"LOCATION1"`
	Notes1          *string    `db:"NOTES1"`
	ATimeOn1        *time.Time `db:"ATIMEON1"`
	ATimeOff1       *time.Time `db:"ATIMEOFF1"`
	ATimeOn2        *time.Time `db:"ATIMEON2"`
	ATimeOff2       *time.Time `db:"ATIMEOFF2"`
	ATimeOn3        *time.Time `db:"ATIMEON3"`
	ATimeOff3       *time.Time `db:"ATIMEOFF3"`
	ATimeOn4        *time.Time `db:"ATIMEON4"`
	ATimeOff4       *time.Time `db:"ATIMEOFF4"`
	ATimeOn5        *time.Time `db:"ATIMEON5"`
	ATimeOff5       *time.Time `db:"ATIMEOFF5"`
	ATimeOn6        *time.Time `db:"ATIMEON6"`
	ATimeOff6       *time.Time `db:"ATIMEOFF6"`
	ATimeOn7        *time.Time `db:"ATIMEON7"`
	ATimeOff7       *time.Time `db:"ATIMEOFF7"`
	TimeControl1    *int64     `db:"TIMECONTROL1"`
	ObjPriority1    *int64     `db:"OBJPRIORITY1"`
	PpkId           *int64     `db:"PPKID"`
	Eng1            *int64     `db:"ENG1"`
	Eng2            *int64     `db:"ENG2"`
	Eng3            *int64     `db:"ENG3"`
	XozOrgId        *int64     `db:"XOZORGID"`
	ReservText      *string    `db:"RESERVTEXT"`
	ReservLong2     *int64     `db:"RESERVLONG2"`
	ReservData1     *time.Time `db:"RESERVDATA1"`
	ReservData2     *time.Time `db:"RESERVDATA2"`
	ReservPhone1    *string    `db:"RESERVPHONE1"`
	ReservPhone2    *string    `db:"RESERVPHONE2"`
	GsmPhone        *string    `db:"GSMPHONE"`
	GsmAccount      *time.Time `db:"GSMACCOUNT"`
	GsmHidenInt     *int64     `db:"GSMHIDENINT"`
	ObjChan         *int64     `db:"OBJCHAN"`
	ResByte1        *int64     `db:"RESBYTE1"`
	ResBigInt1      *int64     `db:"RESBIGINT1"`
	ResBigInt2      *int64     `db:"RESBIGINT2"`
	ResVarChar1     *string    `db:"RESVARCHAR1"`
	ResVarChar2     *string    `db:"RESVARCHAR2"`
	ObjChan2        *int64     `db:"OBJCHAN2"`
	ResByte2        *int64     `db:"RESBYTE2"`
	IdRiskCategory  *int64     `db:"ID_RISK_CATEGORY"`
	GsmPhone2       *string    `db:"GSMPHONE2"`
	GuardedDays     int16      `db:"GUARDED_DAYS"`
	Sbsa            *string    `db:"SBSA"`
	Sbsb            *string    `db:"SBSB"`
	NovaMfCode      string     `db:"NOVA_MF_CODE"`
	Cctv            int16      `db:"CCTV"`
	Installer       int64      `db:"INSTALLER"`
	Manager         int64      `db:"MANAGER"`
	ExtendObj       int16      `db:"EXTENDOBJ"`
}

// ObjectState - модель для таблиці OBJECTS_STATE
type ObjectState struct {
	ID                     int64      `db:"ID"`
	ObjUin                 *int64     `db:"OBJUIN"`
	Objn                   *int64     `db:"OBJN"`
	Grpn                   *int64     `db:"GRPN"`
	GuardState1            *int64     `db:"GUARDSTATE1"`
	AlarmState1            *int64     `db:"ALARMSTATE1"`
	TechAlarmState1        *int64     `db:"TECHALARMSTATE1"`
	IsConnState1           *int64     `db:"ISCONNSTATE1"`
	LastEvId               *int64     `db:"LASTEVID"`
	LastEvTime1            *time.Time `db:"LASTEVTIME1"`
	TestControl1           *int64     `db:"TESTCONTROL1"`
	TestTime1              *int64     `db:"TESTTIME1"`
	NextTestTime1          *time.Time `db:"NEXTTESTTIME1"`
	LastTestTime1          *time.Time `db:"LASTTESTTIME1"`
	GrpPers                *int64     `db:"GRPPERS"`
	OldGrpPers             *int64     `db:"OLDGRPPERS"`
	PowerFault             *int64     `db:"POWERFAULT"`
	ReservLong1            *int64     `db:"RESERVLONG1"`
	GsmFlatCodeInt         *int64     `db:"GSMFLATCODEINT"`
	ObjChan                *int64     `db:"OBJCHAN"`
	FcNew                  *int64     `db:"FCNEW"`
	FcOld                  *int64     `db:"FCOLD"`
	GprsIdps               *int16     `db:"GPRS_IDPS"`
	GprsIdpsOld            *int16     `db:"GPRS_IDPS_OLD"`
	Dat0s                  *int16     `db:"DAT0S"`
	Dat1s                  *int16     `db:"DAT1S"`
	Dat0sOld               *int16     `db:"DAT0S_OLD"`
	Dat1sOld               *int16     `db:"DAT1S_OLD"`
	FcNew2                 *int64     `db:"FCNEW2"`
	FcOld2                 *int64     `db:"FCOLD2"`
	TestControl2           *int64     `db:"TESTCONTROL2"`
	TestTime2              *int64     `db:"TESTTIME2"`
	CurrentChan            *int16     `db:"CURRENT_CHAN"`
	LastActiveGsmPhone     int16      `db:"LASTACTIVEGSMPHONE"`
	UseSetAfterUnsetControl int16      `db:"USE_SET_AFTER_UNSET_CONTROL"`
	UsaucInterval          int64      `db:"USAUC_INTERVAL"`
	UsaucUnsetDt           *time.Time `db:"USAUC_UNSET_DT"`
	UsaucAlarm             int16      `db:"USAUC_ALARM"`
	IsZoneOff              int16      `db:"ISZONEOFF"`
	AkbState               int16      `db:"AKBSTATE"`
	TmprAlarm              int16      `db:"TMPRALARM"`
	BlockedArmedOnOff      int16      `db:"BLOCKEDARMED_ON_OFF"`
	TrkIsChecking          int16      `db:"TRKISCHECKING"`
	TrkIsCheckingDt        *time.Time `db:"TRKISCHECKINGDT"`
	TrkIsChecked           int16      `db:"TRKISCHECKED"`
	AonAlarm               int16      `db:"AON_ALARM"`
	GsmLevel               int16      `db:"GSM_LEVEL"`
	ReadBuffFault          int16      `db:"READBUFFFAULT"`
	PsoFailed              int16      `db:"PSOFAILED"`
	NtfState               int16      `db:"NTFSTATE"`
}

// Personal - модель для таблиці PERSONAL
type Personal struct {
	
	ID            int64      `db:"ID"`
	Surname1      *string    `db:"SURNAME1"`
	Name1         *string    `db:"NAME1"`
	SecName1      *string    `db:"SECNAME1"`
	Address1      *string    `db:"ADDRESS1"`
	Phones1       *string    `db:"PHONES1"`
	Status1       *string    `db:"STATUS1"`
	Access1       *int64     `db:"ACCESS1"`
	Passw1        *string    `db:"PASSW1"`
	Birthday1     *time.Time `db:"BIRTHDAY1"`
	Order1        *int16     `db:"ORDER1"`
	Notes1        *string    `db:"NOTES1"`
	ObjUin        *int64     `db:"OBJUIN"`
	IsRang        *int64     `db:"ISRANG"`
	ResByte1      *int64     `db:"RESBYTE1"`
	GrpPers       *int64     `db:"GRPPERS"`
	LstAct        *time.Time `db:"LSTACT"`
	PConnected    *int16     `db:"PCONNECTED"`
	LstMessInd    *int64     `db:"LST_MESS_IND"`
	RecipientSms  int16      `db:"RECIPIENTSMS"`
	SmsLng        int16      `db:"SMS_LNG"`
	SmsMax        int64      `db:"SMS_MAX"`
	SmsAlarm      int16      `db:"SMS_ALARM"`
	SmsTechAlarm  int16      `db:"SMS_TECHALARM"`
	SmsArming     int16      `db:"SMS_ARMING"`
	SmsOther      int16      `db:"SMS_OTHER"`
	ViberId       *string    `db:"VIBER_ID"`
	TelegramId    *string    `db:"TELEGRAM_ID"`
	RecSms        int16      `db:"REC_SMS"`
	RecViber      int16      `db:"REC_VIBER"`
	RecTelegram   int16      `db:"REC_TELEGRAM"`
	HoGrp         *int64     `db:"HOGRP"`
	ItAl          int16      `db:"IT_AL"`
}

// Ppk - модель для таблиці PPK
type Ppk struct {
	ID             int64   `db:"ID"`
	PanelMark1     *string `db:"PANELMARK1"`
	PanelType0     *int64  `db:"PANELTYPE0"`
	ZonesCount1    *int64  `db:"ZONESCOUNT1"`
	RelayCount1    *int64  `db:"RELAYCOUNT1"`
	PgmCount1      *int64  `db:"PGMCOUNT1"`
	InCount1       *int64  `db:"INCOUNT1"`
	AsptOutCount1  *int64  `db:"ASPTOUTCOUNT1"`
	OpovOutCount1  *int64  `db:"OPOVOUTCOUNT1"`
	VentOutCount1  *int64  `db:"VENTOUTCOUNT1"`
	ControlOutCount1 *int64 `db:"CONTROLOUTCOUNT1"`
	ResByte1       *int64  `db:"RESBYTE1"`
}

// Zone - модель для таблиці ZONES
type Zone struct {
	ID              int64      `db:"ID"`
	ObjUin          *int64     `db:"OBJUIN"`
	Objn            *int64     `db:"OBJN"`
	Grpn            *int64     `db:"GRPN"`
	Zonen           *int64     `db:"ZONEN"`
	ZoneState1      *int64     `db:"ZONESTATE1"`
	ZoneType1       *int64     `db:"ZONETYPE1"`
	ZoneDescr1      *string    `db:"ZONEDESCR1"`
	AlarmState1     *int64     `db:"ALARMSTATE1"`
	TimeCtrl1       *time.Time `db:"TIMECTRL1"`
	TimeCtrl2       *time.Time `db:"TIMECTRL2"`
	ResBigInt1      *int64     `db:"RESBIGINT1"`
	ResBigInt2      *int64     `db:"RESBIGINT2"`
	ResVarChar1     *string    `db:"RESVARCHAR1"`
	ResVarChar2     *string    `db:"RESVARCHAR2"`
	ZoneOn          int16      `db:"ZONEON"`
	TechAlarmState1 *int16     `db:"TECHALARMSTATE1"`
	ZnGrp           *int64     `db:"ZNGRP"`
}