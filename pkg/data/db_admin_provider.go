package data

import (
	"context"
	"database/sql"
	"fmt"
	"sort"
	"strconv"
	"strings"
	"time"

	"obj_catalog_fyne_v3/pkg/utils"

	"github.com/jmoiron/sqlx"
	"github.com/rs/zerolog/log"
)

const adminQueryTimeout = 5 * time.Second

type objTypeRow struct {
	ID       int64   `db:"ID"`
	ObjType1 *string `db:"OBJTYPE1"`
}

type regionRow struct {
	ID     int64   `db:"ID"`
	RgInd  *int64  `db:"RG_IND"`
	RgName *string `db:"RG_NAME"`
}

type objectDistrictRow struct {
	ID   int64   `db:"ID"`
	Reg1 *string `db:"REG1"`
}

type alarmReasonRow struct {
	ID      int64   `db:"ID"`
	Reason1 *string `db:"REASON1"`
}

type messageRow struct {
	UIN          int64   `db:"UIN"`
	ProtID       *int64  `db:"PROTID"`
	MessID       *int64  `db:"MESSID"`
	MessIDHex    *string `db:"MESSIDHEX"`
	Ukr1         *string `db:"UKR1"`
	Sc1          *int64  `db:"SC1"`
	ForAdminOnly *int16  `db:"FORADMINONLY"`
}

type message220Row struct {
	UIN          int64   `db:"UIN"`
	ProtID       *int64  `db:"PROTID"`
	MessID       *int64  `db:"MESSID"`
	MessIDHex    *string `db:"MESSIDHEX"`
	Ukr1         *string `db:"UKR1"`
	Sc1          *int64  `db:"SC1"`
	ForAdminOnly *int16  `db:"FORADMINONLY"`
	Mc220v       *int16  `db:"MC220V"`
}

type displayBlockRow struct {
	ObjN              *int64  `db:"OBJN"`
	ObjShortName1     *string `db:"OBJSHORTNAME1"`
	AlarmState1       *int64  `db:"ALARMSTATE1"`
	GuardState1       *int64  `db:"GUARDSTATE1"`
	TechAlarmState1   *int64  `db:"TECHALARMSTATE1"`
	IsConnState1      *int64  `db:"ISCONNSTATE1"`
	Eng1              *int64  `db:"ENG1"`
	BlockedArmedOnOff *int16  `db:"BLOCKEDARMED_ON_OFF"`
}

type objectRefRow struct {
	ObjUin int64  `db:"OBJUIN"`
	GrpN   *int64 `db:"GRPN"`
}

type pultInfoRow struct {
	ID        *int64  `db:"ID"`
	ParamName *string `db:"PARAM_NAME"`
	ParamVal  *string `db:"PARAM_VALUE"`
}

type ppkConstructorRow struct {
	ID          int64   `db:"ID"`
	PanelMark1  *string `db:"PANELMARK1"`
	ZoneCount1  *int64  `db:"ZONESCOUNT1"`
	ChannelCode *int64  `db:"RESBYTE1"`
}

type subServerRow struct {
	ID      int64   `db:"ID"`
	SBInfo  *string `db:"SBINFO"`
	SBInd   *string `db:"SBIND"`
	SBHost  *string `db:"SBHOST"`
	SBType  *int64  `db:"SBTYPE"`
	SBHost2 *string `db:"SBHOST2"`
}

type subServerObjectRow struct {
	ObjN         *int64  `db:"OBJN"`
	ObjShortName *string `db:"OBJSHORTNAME1"`
	Address1     *string `db:"ADDRESS1"`
	SBSA         *string `db:"SBSA"`
	SBSB         *string `db:"SBSB"`
}

type objectCardRow struct {
	ObjUin       *int64  `db:"OBJUIN"`
	ObjN         *int64  `db:"OBJN"`
	GrpN         *int64  `db:"GRPN"`
	ObjShortName *string `db:"OBJSHORTNAME1"`
	ObjFullName  *string `db:"OBJFULLNAME1"`
	ObjTypeID    *int64  `db:"OBJTYPEID"`
	ObjRegID     *int64  `db:"OBJREGID"`
	Address1     *string `db:"ADDRESS1"`
	Phones1      *string `db:"PHONES1"`
	Contract1    *string `db:"CONTRACT1"`
	ReservText   *string `db:"RESERVTEXT"`
	Location1    *string `db:"LOCATION1"`
	Notes1       *string `db:"NOTES1"`
	ObjChan      *int64  `db:"OBJCHAN"`
	PpkID        *int64  `db:"PPKID"`
	GsmPhone1    *string `db:"GSMPHONE"`
	GsmPhone2    *string `db:"GSMPHONE2"`
	GsmHiddenN   *int64  `db:"GSMHIDENINT"`
	SBSA         *string `db:"SBSA"`
	SBSB         *string `db:"SBSB"`
	TestControl1 *int64  `db:"TESTCONTROL1"`
	TestTime1    *int64  `db:"TESTTIME1"`
}

type objectPersonalRow struct {
	ID         int64      `db:"ID"`
	Order1     *int16     `db:"ORDER1"`
	Surname1   *string    `db:"SURNAME1"`
	Name1      *string    `db:"NAME1"`
	SecName1   *string    `db:"SECNAME1"`
	Address1   *string    `db:"ADDRESS1"`
	Phones1    *string    `db:"PHONES1"`
	Status1    *string    `db:"STATUS1"`
	Notes1     *string    `db:"NOTES1"`
	Access1    *int64     `db:"ACCESS1"`
	Birthday1  *time.Time `db:"BIRTHDAY1"`
	IsRang     *int64     `db:"ISRANG"`
	ViberID    *string    `db:"VIBER_ID"`
	TelegramID *string    `db:"TELEGRAM_ID"`
	TRKTester  *int16     `db:"IT_AL"`
}

type personalLookupRow struct {
	ObjN       *int64     `db:"OBJN"`
	ID         int64      `db:"ID"`
	Order1     *int16     `db:"ORDER1"`
	Surname1   *string    `db:"SURNAME1"`
	Name1      *string    `db:"NAME1"`
	SecName1   *string    `db:"SECNAME1"`
	Address1   *string    `db:"ADDRESS1"`
	Phones1    *string    `db:"PHONES1"`
	Status1    *string    `db:"STATUS1"`
	Notes1     *string    `db:"NOTES1"`
	Access1    *int64     `db:"ACCESS1"`
	Birthday1  *time.Time `db:"BIRTHDAY1"`
	IsRang     *int64     `db:"ISRANG"`
	ViberID    *string    `db:"VIBER_ID"`
	TelegramID *string    `db:"TELEGRAM_ID"`
	TRKTester  *int16     `db:"IT_AL"`
}

type objectZoneRow struct {
	ID         int64   `db:"ID"`
	ZoneNumber *int64  `db:"ZONEN"`
	ZoneType   *int64  `db:"ZONETYPE1"`
	ZoneDescr  *string `db:"ZONEDESCR1"`
	EntryDelay *int64  `db:"RESBIGINT1"`
}

type objectSIMUsageRow struct {
	ObjN         *int64  `db:"OBJN"`
	ObjShortName *string `db:"OBJSHORTNAME1"`
	GSMPhone     *string `db:"GSMPHONE"`
	GSMPhone2    *string `db:"GSMPHONE2"`
}

type objectGPSRow struct {
	ID        *int64  `db:"ID"`
	Latitude  *string `db:"LATITUDE"`
	Longitude *string `db:"LONGITUDE"`
}

func (p *DBDataProvider) GetObjectCard(objn int64) (AdminObjectCard, error) {
	if objn <= 0 {
		return AdminObjectCard{}, fmt.Errorf("invalid object number")
	}

	ctx, cancel := context.WithTimeout(context.Background(), adminQueryTimeout)
	defer cancel()

	const q = `
		SELECT FIRST 1
			oi.OBJUIN,
			oi.OBJN,
			COALESCE(oi.GRPN, 1) AS GRPN,
			oi.OBJSHORTNAME1,
			oi.OBJFULLNAME1,
			COALESCE(oi.OBJTYPEID, 0) AS OBJTYPEID,
			COALESCE(oi.OBJREGID, 0) AS OBJREGID,
			oi.ADDRESS1,
			oi.PHONES1,
			oi.CONTRACT1,
			oi.RESERVTEXT,
			oi.LOCATION1,
			oi.NOTES1,
			COALESCE(oi.OBJCHAN, 0) AS OBJCHAN,
			COALESCE(oi.PPKID, 0) AS PPKID,
			oi.GSMPHONE,
			oi.GSMPHONE2,
			COALESCE(oi.GSMHIDENINT, 0) AS GSMHIDENINT,
			oi.SBSA,
			oi.SBSB,
			COALESCE(os.TESTCONTROL1, 0) AS TESTCONTROL1,
			COALESCE(os.TESTTIME1, 0) AS TESTTIME1
		FROM OBJECTS_INFO oi
		LEFT JOIN OBJECTS_STATE os ON os.OBJUIN = oi.OBJUIN
		WHERE oi.OBJN = ?
	`

	var row objectCardRow
	if err := p.db.GetContext(ctx, &row, p.db.Rebind(q), objn); err != nil {
		if err == sql.ErrNoRows {
			return AdminObjectCard{}, fmt.Errorf("object #%d not found", objn)
		}
		return AdminObjectCard{}, fmt.Errorf("failed to load object card: %w", err)
	}

	return AdminObjectCard{
		ObjUIN:             ptrToInt64(row.ObjUin),
		ObjN:               ptrToInt64(row.ObjN),
		GrpN:               ptrToInt64(row.GrpN),
		ShortName:          ptrToString(row.ObjShortName),
		FullName:           ptrToString(row.ObjFullName),
		ObjTypeID:          ptrToInt64(row.ObjTypeID),
		ObjRegID:           ptrToInt64(row.ObjRegID),
		Address:            ptrToString(row.Address1),
		Phones:             ptrToString(row.Phones1),
		Contract:           ptrToString(row.Contract1),
		StartDate:          ptrToString(row.ReservText),
		Location:           ptrToString(row.Location1),
		Notes:              ptrToString(row.Notes1),
		ChannelCode:        ptrToInt64(row.ObjChan),
		PPKID:              ppkStoredToCatalogID(ptrToInt64(row.PpkID)),
		GSMPhone1:          ptrToString(row.GsmPhone1),
		GSMPhone2:          ptrToString(row.GsmPhone2),
		GSMHiddenN:         ptrToInt64(row.GsmHiddenN),
		SubServerA:         strings.TrimSpace(ptrToString(row.SBSA)),
		SubServerB:         strings.TrimSpace(ptrToString(row.SBSB)),
		TestControlEnabled: ptrToInt64(row.TestControl1) != 0,
		TestIntervalMin:    ptrToInt64(row.TestTime1),
	}, nil
}

func (p *DBDataProvider) CreateObject(card AdminObjectCard) error {
	normalized, err := normalizeAdminObjectCard(card)
	if err != nil {
		return err
	}

	ctx, cancel := context.WithTimeout(context.Background(), adminQueryTimeout)
	defer cancel()

	tx, err := p.db.BeginTxx(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to begin object create transaction: %w", err)
	}
	defer func() {
		_ = tx.Rollback()
	}()

	// У Firebird/legacy базі можуть залишатися "сирітські" записи в mirror-таблицях
	// (OBJECTS/OBJECTS_STATE/OBJECTS_LA) без відповідного OBJECTS_INFO.
	// Для цільового OBJN прибираємо такі рядки до перевірок та вставки.
	if err := cleanupOrphanObjectMirrorRowsByObjNTx(ctx, tx, p.db, normalized.ObjN); err != nil {
		return err
	}

	var exists int
	if err := tx.GetContext(
		ctx,
		&exists,
		p.db.Rebind(`SELECT COUNT(*) FROM OBJECTS_INFO WHERE OBJN = ?`),
		normalized.ObjN,
	); err != nil {
		return fmt.Errorf("failed to check object number uniqueness: %w", err)
	}
	if exists > 0 {
		return fmt.Errorf("object number #%d already exists", normalized.ObjN)
	}

	testControl := int64(0)
	testInterval := int64(0)
	if normalized.TestControlEnabled {
		testControl = 1
		testInterval = normalized.TestIntervalMin
	}

	storedPPKID := ppkCatalogToStoredID(normalized.PPKID)
	regionID, err := resolveDefaultRegionIDTx(ctx, tx, p.db, normalized.ObjRegID)
	if err != nil {
		return err
	}

	const qInsInfo = `
		INSERT INTO OBJECTS_INFO (
			OBJN, GRPN, OBJFULLNAME1, OBJSHORTNAME1, OBJTYPEID,
			ADDRESS1, PHONES1, OBJREGID, CONTRACT1, LOCATION1, NOTES1,
			TIMECONTROL1, OBJPRIORITY1, PPKID, ENG1, ENG2, ENG3, SBSA, SBSB,
			RESERVTEXT, RESERVPHONE1, RESERVPHONE2, GSMPHONE, GSMPHONE2, GSMHIDENINT, OBJCHAN, RESBYTE1, RESERVLONG2, GUARDED_DAYS
		)
		VALUES (
			?, ?, ?, ?, ?,
			?, ?, ?, ?, ?, ?,
			1, 2, ?, 0, 0, 0, ?, ?,
			?, ?, ?, ?, ?, ?, ?, 0, 0, 255
		)
		RETURNING OBJUIN
	`

	var objUIN int64
	if err := tx.QueryRowxContext(
		ctx,
		p.db.Rebind(qInsInfo),
		normalized.ObjN,
		normalized.GrpN,
		normalized.FullName,
		normalized.ShortName,
		normalized.ObjTypeID,
		normalized.Address,
		normalized.Phones,
		regionID,
		normalized.Contract,
		normalized.Location,
		normalized.Notes,
		storedPPKID,
		nullableTrimmedString(normalized.SubServerA),
		nullableTrimmedString(normalized.SubServerB),
		normalized.StartDate,
		"",
		"",
		normalized.GSMPhone1,
		normalized.GSMPhone2,
		nullableInt64(normalized.GSMHiddenN),
		normalized.ChannelCode,
	).Scan(&objUIN); err != nil {
		return fmt.Errorf("failed to insert OBJECTS_INFO: %w", err)
	}

	const qInsVInfo2 = `
		INSERT INTO OBJECTS_VINFO2 (OBJUIN, OBJN, GRPN)
		VALUES (?, ?, ?)
	`
	if _, err := tx.ExecContext(
		ctx,
		p.db.Rebind(qInsVInfo2),
		objUIN,
		normalized.ObjN,
		normalized.GrpN,
	); err != nil {
		return fmt.Errorf("failed to insert OBJECTS_VINFO2 row: %w", err)
	}

	const qInsGPS = `
		INSERT INTO OBJECTS_GPS (OBJUIN, OBJN, GRPN, LATITUDE, LONGITUDE, NAV_ID)
		VALUES (?, ?, NULL, '0', '0', 0)
	`
	if _, err := tx.ExecContext(
		ctx,
		p.db.Rebind(qInsGPS),
		objUIN,
		normalized.ObjN,
	); err != nil {
		return fmt.Errorf("failed to insert OBJECTS_GPS row: %w", err)
	}

	const qInsState = `
		INSERT INTO OBJECTS_STATE (
			OBJUIN, OBJN, GRPN, GUARDSTATE1, ALARMSTATE1, TECHALARMSTATE1,
			ISCONNSTATE1, TESTCONTROL1, TESTTIME1, OBJCHAN, AKBSTATE, TMPRALARM, BLOCKEDARMED_ON_OFF
		)
		VALUES (?, ?, ?, 1, 0, 0, 0, ?, ?, ?, 0, 0, 0)
	`
	if _, err := tx.ExecContext(
		ctx,
		p.db.Rebind(qInsState),
		objUIN,
		normalized.ObjN,
		normalized.GrpN,
		testControl,
		testInterval,
		normalized.ChannelCode,
	); err != nil {
		return fmt.Errorf("failed to insert OBJECTS_STATE row: %w", err)
	}

	const qInsLA = `
		INSERT INTO OBJECTS_LA (OBJUIN, OBJN, GRPN, ISCONNSTATE1)
		VALUES (?, ?, ?, 1)
	`
	if _, err := tx.ExecContext(
		ctx,
		p.db.Rebind(qInsLA),
		objUIN,
		normalized.ObjN,
		normalized.GrpN,
	); err != nil {
		return fmt.Errorf("failed to insert OBJECTS_LA row: %w", err)
	}

	createdRef := objectRefRow{ObjUin: objUIN, GrpN: &normalized.GrpN}
	if err := insertObjectCrudNotificationEventTx(ctx, tx, p.db, createdRef, 51); err != nil {
		return fmt.Errorf("failed to insert object-create notification event: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit object create transaction: %w", err)
	}
	p.notifyObjectCatalogChanged()
	return nil
}

func (p *DBDataProvider) UpdateObject(card AdminObjectCard) error {
	normalized, err := normalizeAdminObjectCard(card)
	if err != nil {
		return err
	}

	ctx, cancel := context.WithTimeout(context.Background(), adminQueryTimeout)
	defer cancel()

	tx, err := p.db.BeginTxx(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to begin object update transaction: %w", err)
	}
	defer func() {
		_ = tx.Rollback()
	}()

	// Аналогічно CreateObject: прибираємо сирітські mirror-рядки по цільовому OBJN.
	if err := cleanupOrphanObjectMirrorRowsByObjNTx(ctx, tx, p.db, normalized.ObjN); err != nil {
		return err
	}

	objUIN := normalized.ObjUIN
	if objUIN <= 0 {
		if err := tx.GetContext(
			ctx,
			&objUIN,
			p.db.Rebind(`SELECT FIRST 1 OBJUIN FROM OBJECTS_INFO WHERE OBJN = ?`),
			normalized.ObjN,
		); err != nil {
			if err == sql.ErrNoRows {
				return fmt.Errorf("object #%d not found", normalized.ObjN)
			}
			return fmt.Errorf("failed to resolve object UIN: %w", err)
		}
	}

	var duplicateCount int
	if err := tx.GetContext(
		ctx,
		&duplicateCount,
		p.db.Rebind(`SELECT COUNT(*) FROM OBJECTS_INFO WHERE OBJN = ? AND OBJUIN <> ?`),
		normalized.ObjN,
		objUIN,
	); err != nil {
		return fmt.Errorf("failed to check object number uniqueness: %w", err)
	}
	if duplicateCount > 0 {
		return fmt.Errorf("object number #%d already exists", normalized.ObjN)
	}

	testControl := int64(0)
	testInterval := int64(0)
	if normalized.TestControlEnabled {
		testControl = 1
		testInterval = normalized.TestIntervalMin
	}
	storedPPKID := ppkCatalogToStoredID(normalized.PPKID)
	regionID, err := resolveDefaultRegionIDTx(ctx, tx, p.db, normalized.ObjRegID)
	if err != nil {
		return err
	}

	const qUpdInfo = `
		UPDATE OBJECTS_INFO
		SET
			OBJN = ?, GRPN = ?, OBJFULLNAME1 = ?, OBJSHORTNAME1 = ?, OBJTYPEID = ?,
			ADDRESS1 = ?, PHONES1 = ?, OBJREGID = ?, CONTRACT1 = ?, LOCATION1 = ?, NOTES1 = ?,
			PPKID = ?, RESERVTEXT = ?, RESERVPHONE1 = '', RESERVPHONE2 = '',
			GSMPHONE = ?, GSMPHONE2 = ?, GSMHIDENINT = ?, OBJCHAN = ?, SBSA = ?, SBSB = ?
		WHERE OBJUIN = ?
	`
	if _, err := tx.ExecContext(
		ctx,
		p.db.Rebind(qUpdInfo),
		normalized.ObjN,
		normalized.GrpN,
		normalized.FullName,
		normalized.ShortName,
		normalized.ObjTypeID,
		normalized.Address,
		normalized.Phones,
		regionID,
		normalized.Contract,
		normalized.Location,
		normalized.Notes,
		storedPPKID,
		normalized.StartDate,
		normalized.GSMPhone1,
		normalized.GSMPhone2,
		nullableInt64(normalized.GSMHiddenN),
		normalized.ChannelCode,
		nullableTrimmedString(normalized.SubServerA),
		nullableTrimmedString(normalized.SubServerB),
		objUIN,
	); err != nil {
		return fmt.Errorf("failed to update OBJECTS_INFO: %w", err)
	}

	const qUpdState = `
		UPDATE OBJECTS_STATE
		SET OBJN = ?, GRPN = ?, TESTCONTROL1 = ?, TESTTIME1 = ?, OBJCHAN = ?
		WHERE OBJUIN = ?
	`
	stateRes, err := tx.ExecContext(
		ctx,
		p.db.Rebind(qUpdState),
		normalized.ObjN,
		normalized.GrpN,
		testControl,
		testInterval,
		normalized.ChannelCode,
		objUIN,
	)
	if err != nil {
		return fmt.Errorf("failed to update OBJECTS_STATE row: %w", err)
	}
	if n, _ := stateRes.RowsAffected(); n == 0 {
		const qInsState = `
			INSERT INTO OBJECTS_STATE (
				OBJUIN, OBJN, GRPN, GUARDSTATE1, ALARMSTATE1, TECHALARMSTATE1,
				ISCONNSTATE1, TESTCONTROL1, TESTTIME1, OBJCHAN, AKBSTATE, TMPRALARM, BLOCKEDARMED_ON_OFF
			)
			VALUES (?, ?, ?, 1, 0, 0, 0, ?, ?, ?, 0, 0, 0)
		`
		if _, err := tx.ExecContext(
			ctx,
			p.db.Rebind(qInsState),
			objUIN,
			normalized.ObjN,
			normalized.GrpN,
			testControl,
			testInterval,
			normalized.ChannelCode,
		); err != nil {
			return fmt.Errorf("failed to insert missing OBJECTS_STATE row: %w", err)
		}
	}

	const qUpdLA = `
		UPDATE OBJECTS_LA
		SET OBJN = ?, GRPN = ?
		WHERE OBJUIN = ?
	`
	laRes, err := tx.ExecContext(
		ctx,
		p.db.Rebind(qUpdLA),
		normalized.ObjN,
		normalized.GrpN,
		objUIN,
	)
	if err != nil {
		return fmt.Errorf("failed to update OBJECTS_LA row: %w", err)
	}
	if n, _ := laRes.RowsAffected(); n == 0 {
		const qInsLA = `
			INSERT INTO OBJECTS_LA (OBJUIN, OBJN, GRPN, ISCONNSTATE1)
			VALUES (?, ?, ?, 1)
		`
		if _, err := tx.ExecContext(
			ctx,
			p.db.Rebind(qInsLA),
			objUIN,
			normalized.ObjN,
			normalized.GrpN,
		); err != nil {
			return fmt.Errorf("failed to insert missing OBJECTS_LA row: %w", err)
		}
	}

	updatedRef := objectRefRow{ObjUin: objUIN, GrpN: &normalized.GrpN}
	if err := insertObjectCrudNotificationEventTx(ctx, tx, p.db, updatedRef, 52); err != nil {
		return fmt.Errorf("failed to insert object-update notification event: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit object update transaction: %w", err)
	}
	p.notifyObjectCatalogChanged()
	return nil
}

func (p *DBDataProvider) DeleteObject(objn int64) error {
	if objn <= 0 {
		return fmt.Errorf("invalid object number")
	}

	ctx, cancel := context.WithTimeout(context.Background(), adminQueryTimeout)
	defer cancel()

	tx, err := p.db.BeginTxx(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to begin object delete transaction: %w", err)
	}
	defer func() {
		_ = tx.Rollback()
	}()

	var objUIN int64
	if err := tx.GetContext(
		ctx,
		&objUIN,
		p.db.Rebind(`SELECT FIRST 1 OBJUIN FROM OBJECTS_INFO WHERE OBJN = ?`),
		objn,
	); err != nil {
		if err == sql.ErrNoRows {
			return fmt.Errorf("object #%d not found", objn)
		}
		return fmt.Errorf("failed to resolve object UIN for delete: %w", err)
	}

	var sbsa sql.NullString
	var sbsb sql.NullString
	if err := tx.QueryRowxContext(
		ctx,
		p.db.Rebind(`SELECT FIRST 1 SBSA, SBSB FROM OBJECTS_INFO WHERE OBJUIN = ?`),
		objUIN,
	).Scan(&sbsa, &sbsb); err != nil {
		return fmt.Errorf("failed to verify subserver binding before delete: %w", err)
	}
	sbsaVal := ""
	if sbsa.Valid {
		sbsaVal = strings.TrimSpace(sbsa.String)
	}
	sbsbVal := ""
	if sbsb.Valid {
		sbsbVal = strings.TrimSpace(sbsb.String)
	}
	if sbsaVal != "" || sbsbVal != "" {
		parts := make([]string, 0, 2)
		if sbsaVal != "" {
			parts = append(parts, fmt.Sprintf("SBSA=%s", sbsaVal))
		}
		if sbsbVal != "" {
			parts = append(parts, fmt.Sprintf("SBSB=%s", sbsbVal))
		}
		return fmt.Errorf("об'єкт #%d прив'язаний до підсервера (%s). Спочатку відв'яжіть його у вікні \"Керування об'єктами підсерверів\" (або в картці об'єкта)", objn, strings.Join(parts, ", "))
	}

	var deletedName sql.NullString
	var deletedAddress sql.NullString
	var deletedPhones sql.NullString
	if err := tx.QueryRowxContext(
		ctx,
		p.db.Rebind(`SELECT FIRST 1 OBJSHORTNAME1, ADDRESS1, PHONES1 FROM OBJECTS_INFO WHERE OBJUIN = ?`),
		objUIN,
	).Scan(&deletedName, &deletedAddress, &deletedPhones); err != nil {
		return fmt.Errorf("failed to read object data for DELREPORT: %w", err)
	}
	deletedNameVal := ""
	if deletedName.Valid {
		deletedNameVal = strings.TrimSpace(deletedName.String)
	}
	deletedAddressVal := ""
	if deletedAddress.Valid {
		deletedAddressVal = strings.TrimSpace(deletedAddress.String)
	}
	deletedPhonesVal := ""
	if deletedPhones.Valid {
		deletedPhonesVal = strings.TrimSpace(deletedPhones.String)
	}

	const qInsDelReport = `
		INSERT INTO DELREPORT (
			OBJN, OBJSHORTNAME1, ADDRESS1, PHONES1, REASONDEL, INFORMED, DT_DEL, DT_INFORMED
		)
		VALUES (?, ?, ?, ?, '', '', CURRENT_TIMESTAMP, NULL)
	`
	if _, err := tx.ExecContext(
		ctx,
		p.db.Rebind(qInsDelReport),
		objn,
		deletedNameVal,
		deletedAddressVal,
		deletedPhonesVal,
	); err != nil {
		return fmt.Errorf("failed to insert DELREPORT row: %w", err)
	}

	deletedRef, err := resolveObjectRefByObjNTx(ctx, tx, p.db, objn)
	if err != nil {
		return err
	}
	if err := insertObjectCrudNotificationEventTx(ctx, tx, p.db, deletedRef, 53); err != nil {
		return fmt.Errorf("failed to insert object-delete notification event: %w", err)
	}

	queries := []string{
		`DELETE FROM IMAGES WHERE OBJUIN = ?`,
		`DELETE FROM ZONES WHERE OBJUIN = ?`,
		`DELETE FROM PERSONAL WHERE OBJUIN = ?`,
		`DELETE FROM OBJECTS_GPS WHERE OBJUIN = ?`,
		`DELETE FROM OBJECTS_VINFO2 WHERE OBJUIN = ?`,
		`DELETE FROM OBJECTS_LA WHERE OBJUIN = ?`,
		`DELETE FROM OBJECTS_STATE WHERE OBJUIN = ?`,
		`DELETE FROM OBJECTS_INFO WHERE OBJUIN = ?`,
	}

	for _, q := range queries {
		if _, err := tx.ExecContext(ctx, p.db.Rebind(q), objUIN); err != nil {
			return fmt.Errorf("failed to delete object #%d: %w", objn, err)
		}
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit object delete transaction: %w", err)
	}
	p.notifyObjectCatalogChanged()
	return nil
}

func (p *DBDataProvider) ListObjectPersonals(objn int64) ([]AdminObjectPersonal, error) {
	if objn <= 0 {
		return nil, fmt.Errorf("invalid object number")
	}

	ctx, cancel := context.WithTimeout(context.Background(), adminQueryTimeout)
	defer cancel()

	objRef, err := p.resolveObjectRefByObjN(ctx, objn)
	if err != nil {
		return nil, err
	}

	const q = `
		SELECT
			ID,
			ORDER1,
			SURNAME1,
			NAME1,
			SECNAME1,
			ADDRESS1,
			PHONES1,
			STATUS1,
			NOTES1,
			COALESCE(ACCESS1, 0) AS ACCESS1,
			BIRTHDAY1,
			COALESCE(ISRANG, 0) AS ISRANG,
			VIBER_ID,
			TELEGRAM_ID,
			COALESCE(IT_AL, 0) AS IT_AL
		FROM PERSONAL
		WHERE OBJUIN = ?
		ORDER BY COALESCE(ORDER1, 32767), ID
	`

	var rows []objectPersonalRow
	if err := p.db.SelectContext(ctx, &rows, p.db.Rebind(q), objRef.ObjUin); err != nil {
		return nil, fmt.Errorf("failed to list object personals: %w", err)
	}

	items := make([]AdminObjectPersonal, 0, len(rows))
	for _, r := range rows {
		items = append(items, AdminObjectPersonal{
			ID:          r.ID,
			Number:      int64(ptrToInt16(r.Order1)),
			Surname:     ptrToString(r.Surname1),
			Name:        ptrToString(r.Name1),
			SecName:     ptrToString(r.SecName1),
			Address:     ptrToString(r.Address1),
			Phones:      ptrToString(r.Phones1),
			Position:    ptrToString(r.Status1),
			Notes:       ptrToString(r.Notes1),
			Access1:     ptrToInt64(r.Access1),
			IsRang:      ptrToInt64(r.IsRang) != 0,
			ViberID:     ptrToString(r.ViberID),
			TelegramID:  ptrToString(r.TelegramID),
			CreatedAt:   formatDateTimePtr(r.Birthday1),
			IsTRKTester: ptrToInt16(r.TRKTester) != 0,
		})
	}

	return items, nil
}

func (p *DBDataProvider) FindPersonalByPhone(phone string) (*AdminObjectPersonal, error) {
	normalized := normalizeUASimPhone(phone)
	if len(normalized) != 10 || !strings.HasPrefix(normalized, "0") {
		return nil, nil
	}

	ctx, cancel := context.WithTimeout(context.Background(), adminQueryTimeout)
	defer cancel()

	const q = `
		SELECT FIRST 5000
			OI.OBJN,
			P.ID,
			P.ORDER1,
			P.SURNAME1,
			P.NAME1,
			P.SECNAME1,
			P.ADDRESS1,
			P.PHONES1,
			P.STATUS1,
			P.NOTES1,
			COALESCE(P.ACCESS1, 0) AS ACCESS1,
			P.BIRTHDAY1,
			COALESCE(P.ISRANG, 0) AS ISRANG,
			P.VIBER_ID,
			P.TELEGRAM_ID,
			COALESCE(P.IT_AL, 0) AS IT_AL
		FROM PERSONAL P
		LEFT JOIN OBJECTS_INFO OI ON OI.OBJUIN = P.OBJUIN
		WHERE P.PHONES1 IS NOT NULL
		  AND CHAR_LENGTH(TRIM(P.PHONES1)) > 0
		ORDER BY P.ID DESC
	`

	var rows []personalLookupRow
	if err := p.db.SelectContext(ctx, &rows, p.db.Rebind(q)); err != nil {
		return nil, fmt.Errorf("failed to lookup personal by phone: %w", err)
	}

	for _, r := range rows {
		candidates := extractNormalizedUAPhones(ptrToString(r.Phones1))
		if _, ok := candidates[normalized]; !ok {
			continue
		}
		it := AdminObjectPersonal{
			ID:          r.ID,
			SourceObjN:  ptrToInt64(r.ObjN),
			Number:      int64(ptrToInt16(r.Order1)),
			Surname:     ptrToString(r.Surname1),
			Name:        ptrToString(r.Name1),
			SecName:     ptrToString(r.SecName1),
			Address:     ptrToString(r.Address1),
			Phones:      ptrToString(r.Phones1),
			Position:    ptrToString(r.Status1),
			Notes:       ptrToString(r.Notes1),
			Access1:     ptrToInt64(r.Access1),
			IsRang:      ptrToInt64(r.IsRang) != 0,
			ViberID:     ptrToString(r.ViberID),
			TelegramID:  ptrToString(r.TelegramID),
			CreatedAt:   formatDateTimePtr(r.Birthday1),
			IsTRKTester: ptrToInt16(r.TRKTester) != 0,
		}
		return &it, nil
	}

	return nil, nil
}

func (p *DBDataProvider) FindObjectsBySIMPhone(phone string, excludeObjN *int64) ([]AdminSIMPhoneUsage, error) {
	normalized := normalizeUASimPhone(phone)
	if len(normalized) != 10 || !strings.HasPrefix(normalized, "0") {
		return nil, nil
	}

	ctx, cancel := context.WithTimeout(context.Background(), adminQueryTimeout)
	defer cancel()

	const q = `
		SELECT
			OBJN,
			OBJSHORTNAME1,
			GSMPHONE,
			GSMPHONE2
		FROM OBJECTS_INFO
		WHERE OBJN > 0
		  AND (
			COALESCE(TRIM(GSMPHONE), '') <> ''
			OR COALESCE(TRIM(GSMPHONE2), '') <> ''
		  )
		ORDER BY OBJN
	`

	var rows []objectSIMUsageRow
	if err := p.db.SelectContext(ctx, &rows, p.db.Rebind(q)); err != nil {
		return nil, fmt.Errorf("failed to lookup SIM phone usage: %w", err)
	}

	type slotUsage struct {
		objn int64
		name string
		slot string
	}
	found := make([]slotUsage, 0, 8)
	for _, r := range rows {
		objn := ptrToInt64(r.ObjN)
		if objn <= 0 {
			continue
		}
		if excludeObjN != nil && objn == *excludeObjN {
			continue
		}
		name := strings.TrimSpace(ptrToString(r.ObjShortName))
		if normalizeUASimPhone(ptrToString(r.GSMPhone)) == normalized {
			found = append(found, slotUsage{objn: objn, name: name, slot: "SIM 1"})
		}
		if normalizeUASimPhone(ptrToString(r.GSMPhone2)) == normalized {
			found = append(found, slotUsage{objn: objn, name: name, slot: "SIM 2"})
		}
	}

	sort.SliceStable(found, func(i, j int) bool {
		if found[i].objn == found[j].objn {
			return found[i].slot < found[j].slot
		}
		return found[i].objn < found[j].objn
	})

	usages := make([]AdminSIMPhoneUsage, 0, len(found))
	for _, f := range found {
		usages = append(usages, AdminSIMPhoneUsage{
			ObjN: f.objn,
			Name: f.name,
			Slot: f.slot,
		})
	}
	return usages, nil
}

func (p *DBDataProvider) AddObjectPersonal(objn int64, item AdminObjectPersonal) error {
	if objn <= 0 {
		return fmt.Errorf("invalid object number")
	}

	item = applyPersonalObjectNumberFallback(item, objn)
	normalized, err := normalizeObjectPersonal(item)
	if err != nil {
		return err
	}

	ctx, cancel := context.WithTimeout(context.Background(), adminQueryTimeout)
	defer cancel()

	tx, err := p.db.BeginTxx(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to begin object personal insert transaction: %w", err)
	}
	defer func() {
		_ = tx.Rollback()
	}()

	objRef, err := resolveObjectRefByObjNTx(ctx, tx, p.db, objn)
	if err != nil {
		return err
	}

	orderNum := normalized.Number
	if orderNum <= 0 {
		const qNext = `SELECT COALESCE(MAX(ORDER1), 0) + 1 FROM PERSONAL WHERE OBJUIN = ?`
		if err := tx.GetContext(ctx, &orderNum, p.db.Rebind(qNext), objRef.ObjUin); err != nil {
			return fmt.Errorf("failed to resolve next personal number: %w", err)
		}
		if orderNum <= 0 {
			orderNum = 1
		}
	}

	itAl := int16(0)
	if normalized.IsTRKTester {
		itAl = 1
	}
	isRang := int64(1)
	if normalized.IsRang {
		isRang = 1
	} else {
		isRang = 0
	}
	viberID := trimmedOrEmptyString(normalized.ViberID)
	telegramID := trimmedOrEmptyString(normalized.TelegramID)

	const qIns = `
		INSERT INTO PERSONAL (
			OBJUIN, ORDER1, SURNAME1, NAME1, SECNAME1,
			ADDRESS1, PHONES1, STATUS1, NOTES1,
			ACCESS1, BIRTHDAY1, ISRANG, VIBER_ID, TELEGRAM_ID, IT_AL
		)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, CURRENT_TIMESTAMP, ?, ?, ?, ?)
	`
	accessLevel := int64(0)
	if normalized.Access1 > 0 {
		accessLevel = 1
	}
	if _, err := tx.ExecContext(
		ctx,
		p.db.Rebind(qIns),
		objRef.ObjUin,
		orderNum,
		normalized.Surname,
		normalized.Name,
		normalized.SecName,
		normalized.Address,
		normalized.Phones,
		normalized.Position,
		normalized.Notes,
		accessLevel,
		isRang,
		viberID,
		telegramID,
		itAl,
	); err != nil {
		return fmt.Errorf("failed to insert PERSONAL row: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit object personal insert: %w", err)
	}
	return nil
}

func (p *DBDataProvider) UpdateObjectPersonal(objn int64, item AdminObjectPersonal) error {
	if objn <= 0 {
		return fmt.Errorf("invalid object number")
	}
	if item.ID <= 0 {
		return fmt.Errorf("invalid personal id")
	}

	item = applyPersonalObjectNumberFallback(item, objn)
	normalized, err := normalizeObjectPersonal(item)
	if err != nil {
		return err
	}
	if normalized.Number <= 0 {
		return fmt.Errorf("personal number must be > 0")
	}

	ctx, cancel := context.WithTimeout(context.Background(), adminQueryTimeout)
	defer cancel()

	objRef, err := p.resolveObjectRefByObjN(ctx, objn)
	if err != nil {
		return err
	}

	itAl := int16(0)
	if normalized.IsTRKTester {
		itAl = 1
	}
	isRang := int64(0)
	if normalized.IsRang {
		isRang = 1
	}
	viberID := trimmedOrEmptyString(normalized.ViberID)
	telegramID := trimmedOrEmptyString(normalized.TelegramID)
	accessLevel := int64(0)
	if normalized.Access1 > 0 {
		accessLevel = 1
	}

	const qUpd = `
		UPDATE PERSONAL
		SET
			ORDER1 = ?,
			SURNAME1 = ?,
			NAME1 = ?,
			SECNAME1 = ?,
			ADDRESS1 = ?,
			PHONES1 = ?,
			STATUS1 = ?,
			NOTES1 = ?,
			ACCESS1 = ?,
			ISRANG = ?,
			VIBER_ID = ?,
			TELEGRAM_ID = ?,
			IT_AL = ?
		WHERE ID = ? AND OBJUIN = ?
	`
	res, err := p.db.ExecContext(
		ctx,
		p.db.Rebind(qUpd),
		normalized.Number,
		normalized.Surname,
		normalized.Name,
		normalized.SecName,
		normalized.Address,
		normalized.Phones,
		normalized.Position,
		normalized.Notes,
		accessLevel,
		isRang,
		viberID,
		telegramID,
		itAl,
		normalized.ID,
		objRef.ObjUin,
	)
	if err != nil {
		return fmt.Errorf("failed to update PERSONAL row: %w", err)
	}
	if affected, _ := res.RowsAffected(); affected == 0 {
		return fmt.Errorf("personal id %d not found for object #%d", normalized.ID, objn)
	}
	return nil
}

func (p *DBDataProvider) DeleteObjectPersonal(objn int64, personalID int64) error {
	if objn <= 0 {
		return fmt.Errorf("invalid object number")
	}
	if personalID <= 0 {
		return fmt.Errorf("invalid personal id")
	}

	ctx, cancel := context.WithTimeout(context.Background(), adminQueryTimeout)
	defer cancel()

	objRef, err := p.resolveObjectRefByObjN(ctx, objn)
	if err != nil {
		return err
	}

	const qDel = `DELETE FROM PERSONAL WHERE ID = ? AND OBJUIN = ?`
	res, err := p.db.ExecContext(ctx, p.db.Rebind(qDel), personalID, objRef.ObjUin)
	if err != nil {
		return fmt.Errorf("failed to delete PERSONAL row: %w", err)
	}
	if affected, _ := res.RowsAffected(); affected == 0 {
		return fmt.Errorf("personal id %d not found for object #%d", personalID, objn)
	}
	return nil
}

func (p *DBDataProvider) ListObjectZones(objn int64) ([]AdminObjectZone, error) {
	if objn <= 0 {
		return nil, fmt.Errorf("invalid object number")
	}

	ctx, cancel := context.WithTimeout(context.Background(), adminQueryTimeout)
	defer cancel()

	objRef, err := p.resolveObjectRefByObjN(ctx, objn)
	if err != nil {
		return nil, err
	}

	const q = `
		SELECT
			ID,
			ZONEN,
			COALESCE(ZONETYPE1, 1) AS ZONETYPE1,
			ZONEDESCR1,
			RESBIGINT1
		FROM ZONES
		WHERE OBJUIN = ?
		ORDER BY COALESCE(ZONEN, 32767), ID
	`
	var rows []objectZoneRow
	if err := p.db.SelectContext(ctx, &rows, p.db.Rebind(q), objRef.ObjUin); err != nil {
		return nil, fmt.Errorf("failed to list object zones: %w", err)
	}

	items := make([]AdminObjectZone, 0, len(rows))
	for _, r := range rows {
		items = append(items, AdminObjectZone{
			ID:            r.ID,
			ZoneNumber:    ptrToInt64(r.ZoneNumber),
			ZoneType:      ptrToInt64(r.ZoneType),
			Description:   ptrToString(r.ZoneDescr),
			EntryDelaySec: ptrToInt64(r.EntryDelay),
		})
	}
	return items, nil
}

func (p *DBDataProvider) AddObjectZone(objn int64, zone AdminObjectZone) error {
	if objn <= 0 {
		return fmt.Errorf("invalid object number")
	}

	normalized, err := normalizeObjectZone(zone)
	if err != nil {
		return err
	}

	ctx, cancel := context.WithTimeout(context.Background(), adminQueryTimeout)
	defer cancel()

	tx, err := p.db.BeginTxx(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to begin object zone insert transaction: %w", err)
	}
	defer func() {
		_ = tx.Rollback()
	}()

	objRef, err := resolveObjectRefByObjNTx(ctx, tx, p.db, objn)
	if err != nil {
		return err
	}

	dup, err := hasDuplicateZoneNumber(ctx, tx, p.db, objRef.ObjUin, normalized.ZoneNumber, 0)
	if err != nil {
		return err
	}
	if dup {
		return fmt.Errorf("zone #%d already exists for object #%d", normalized.ZoneNumber, objn)
	}

	const qIns = `
		INSERT INTO ZONES (
			OBJUIN, OBJN, GRPN, ZONEN, ZONESTATE1, ZONETYPE1, ZONEDESCR1,
			ALARMSTATE1, RESBIGINT1, ZONEON, TECHALARMSTATE1, ZNGRP
		)
		VALUES (?, ?, ?, ?, 0, ?, ?, 0, NULL, 1, 0, 0)
	`
	if _, err := tx.ExecContext(
		ctx,
		p.db.Rebind(qIns),
		objRef.ObjUin,
		objn,
		ptrToInt64(objRef.GrpN),
		normalized.ZoneNumber,
		normalized.ZoneType,
		normalized.Description,
	); err != nil {
		return fmt.Errorf("failed to insert ZONES row: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit object zone insert: %w", err)
	}
	return nil
}

func (p *DBDataProvider) UpdateObjectZone(objn int64, zone AdminObjectZone) error {
	if objn <= 0 {
		return fmt.Errorf("invalid object number")
	}
	if zone.ID <= 0 {
		return fmt.Errorf("invalid zone id")
	}

	normalized, err := normalizeObjectZone(zone)
	if err != nil {
		return err
	}

	ctx, cancel := context.WithTimeout(context.Background(), adminQueryTimeout)
	defer cancel()

	tx, err := p.db.BeginTxx(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to begin object zone update transaction: %w", err)
	}
	defer func() {
		_ = tx.Rollback()
	}()

	objRef, err := resolveObjectRefByObjNTx(ctx, tx, p.db, objn)
	if err != nil {
		return err
	}

	dup, err := hasDuplicateZoneNumber(ctx, tx, p.db, objRef.ObjUin, normalized.ZoneNumber, normalized.ID)
	if err != nil {
		return err
	}
	if dup {
		return fmt.Errorf("zone #%d already exists for object #%d", normalized.ZoneNumber, objn)
	}

	const qUpd = `
		UPDATE ZONES
		SET
			ZONEN = ?,
			ZONETYPE1 = ?,
			ZONEDESCR1 = ?,
			RESBIGINT1 = NULL
		WHERE ID = ? AND OBJUIN = ?
	`
	res, err := tx.ExecContext(
		ctx,
		p.db.Rebind(qUpd),
		normalized.ZoneNumber,
		normalized.ZoneType,
		normalized.Description,
		normalized.ID,
		objRef.ObjUin,
	)
	if err != nil {
		return fmt.Errorf("failed to update ZONES row: %w", err)
	}
	if affected, _ := res.RowsAffected(); affected == 0 {
		return fmt.Errorf("zone id %d not found for object #%d", normalized.ID, objn)
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit object zone update: %w", err)
	}
	return nil
}

func (p *DBDataProvider) DeleteObjectZone(objn int64, zoneID int64) error {
	if objn <= 0 {
		return fmt.Errorf("invalid object number")
	}
	if zoneID <= 0 {
		return fmt.Errorf("invalid zone id")
	}

	ctx, cancel := context.WithTimeout(context.Background(), adminQueryTimeout)
	defer cancel()

	objRef, err := p.resolveObjectRefByObjN(ctx, objn)
	if err != nil {
		return err
	}

	const qDel = `DELETE FROM ZONES WHERE ID = ? AND OBJUIN = ?`
	res, err := p.db.ExecContext(ctx, p.db.Rebind(qDel), zoneID, objRef.ObjUin)
	if err != nil {
		return fmt.Errorf("failed to delete ZONES row: %w", err)
	}
	if affected, _ := res.RowsAffected(); affected == 0 {
		return fmt.Errorf("zone id %d not found for object #%d", zoneID, objn)
	}
	return nil
}

func (p *DBDataProvider) FillObjectZones(objn int64, count int64) error {
	if objn <= 0 {
		return fmt.Errorf("invalid object number")
	}
	if count <= 0 || count > 1024 {
		return fmt.Errorf("zone count must be in range 1..1024")
	}

	ctx, cancel := context.WithTimeout(context.Background(), adminQueryTimeout)
	defer cancel()

	tx, err := p.db.BeginTxx(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to begin zone fill transaction: %w", err)
	}
	defer func() {
		_ = tx.Rollback()
	}()

	objRef, err := resolveObjectRefByObjNTx(ctx, tx, p.db, objn)
	if err != nil {
		return err
	}

	const qExisting = `SELECT ZONEN FROM ZONES WHERE OBJUIN = ?`
	var existing []sql.NullInt64
	if err := tx.SelectContext(ctx, &existing, p.db.Rebind(qExisting), objRef.ObjUin); err != nil {
		return fmt.Errorf("failed to load existing zones: %w", err)
	}

	existingSet := make(map[int64]struct{}, len(existing))
	for _, z := range existing {
		if z.Valid && z.Int64 > 0 {
			existingSet[z.Int64] = struct{}{}
		}
	}

	const qIns = `
		INSERT INTO ZONES (
			OBJUIN, OBJN, GRPN, ZONEN, ZONESTATE1, ZONETYPE1, ZONEDESCR1,
			ALARMSTATE1, RESBIGINT1, ZONEON, TECHALARMSTATE1, ZNGRP
		)
		VALUES (?, ?, ?, ?, 0, 1, ?, 0, NULL, 1, 0, 0)
	`
	for i := int64(1); i <= count; i++ {
		if _, ok := existingSet[i]; ok {
			continue
		}
		desc := fmt.Sprintf("Шлейф %d", i)
		if _, err := tx.ExecContext(
			ctx,
			p.db.Rebind(qIns),
			objRef.ObjUin,
			objn,
			ptrToInt64(objRef.GrpN),
			i,
			desc,
		); err != nil {
			return fmt.Errorf("failed to fill zone #%d: %w", i, err)
		}
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit zone fill: %w", err)
	}
	return nil
}

func (p *DBDataProvider) ClearObjectZones(objn int64) error {
	if objn <= 0 {
		return fmt.Errorf("invalid object number")
	}

	ctx, cancel := context.WithTimeout(context.Background(), adminQueryTimeout)
	defer cancel()

	objRef, err := p.resolveObjectRefByObjN(ctx, objn)
	if err != nil {
		return err
	}

	const qDel = `DELETE FROM ZONES WHERE OBJUIN = ?`
	if _, err := p.db.ExecContext(ctx, p.db.Rebind(qDel), objRef.ObjUin); err != nil {
		return fmt.Errorf("failed to clear zones: %w", err)
	}
	return nil
}

func (p *DBDataProvider) GetObjectCoordinates(objn int64) (AdminObjectCoordinates, error) {
	if objn <= 0 {
		return AdminObjectCoordinates{}, fmt.Errorf("invalid object number")
	}

	ctx, cancel := context.WithTimeout(context.Background(), adminQueryTimeout)
	defer cancel()

	objRef, err := p.resolveObjectRefByObjN(ctx, objn)
	if err != nil {
		return AdminObjectCoordinates{}, err
	}

	const q = `
		SELECT FIRST 1 ID, LATITUDE, LONGITUDE
		FROM OBJECTS_GPS
		WHERE OBJUIN = ?
		ORDER BY ID DESC
	`
	var row objectGPSRow
	if err := p.db.GetContext(ctx, &row, p.db.Rebind(q), objRef.ObjUin); err != nil {
		if err == sql.ErrNoRows {
			return AdminObjectCoordinates{}, nil
		}
		return AdminObjectCoordinates{}, fmt.Errorf("failed to load object coordinates: %w", err)
	}

	return AdminObjectCoordinates{
		Latitude:  strings.TrimSpace(ptrToString(row.Latitude)),
		Longitude: strings.TrimSpace(ptrToString(row.Longitude)),
	}, nil
}

func (p *DBDataProvider) SaveObjectCoordinates(objn int64, coords AdminObjectCoordinates) error {
	if objn <= 0 {
		return fmt.Errorf("invalid object number")
	}

	lat := strings.TrimSpace(coords.Latitude)
	lon := strings.TrimSpace(coords.Longitude)
	if len(lat) > 20 {
		return fmt.Errorf("latitude length must be <= 20")
	}
	if len(lon) > 20 {
		return fmt.Errorf("longitude length must be <= 20")
	}
	lat = coordinateOrZero(lat)
	lon = coordinateOrZero(lon)

	ctx, cancel := context.WithTimeout(context.Background(), adminQueryTimeout)
	defer cancel()

	tx, err := p.db.BeginTxx(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to begin coordinates transaction: %w", err)
	}
	defer func() {
		_ = tx.Rollback()
	}()

	objRef, err := resolveObjectRefByObjNTx(ctx, tx, p.db, objn)
	if err != nil {
		return err
	}

	const qGetID = `SELECT FIRST 1 ID FROM OBJECTS_GPS WHERE OBJUIN = ? ORDER BY ID DESC`
	var gpsID int64
	err = tx.GetContext(ctx, &gpsID, p.db.Rebind(qGetID), objRef.ObjUin)
	if err != nil && err != sql.ErrNoRows {
		return fmt.Errorf("failed to resolve coordinates row id: %w", err)
	}

	if err == sql.ErrNoRows {
		const qIns = `
			INSERT INTO OBJECTS_GPS (OBJUIN, OBJN, GRPN, LATITUDE, LONGITUDE, NAV_ID)
			VALUES (?, ?, NULL, ?, ?, 0)
		`
		if _, err := tx.ExecContext(
			ctx,
			p.db.Rebind(qIns),
			objRef.ObjUin,
			objn,
			lat,
			lon,
		); err != nil {
			return fmt.Errorf("failed to insert object coordinates: %w", err)
		}
	} else {
		const qUpd = `
			UPDATE OBJECTS_GPS
			SET OBJN = ?, GRPN = NULL, LATITUDE = ?, LONGITUDE = ?, NAV_ID = 0
			WHERE ID = ?
		`
		if _, err := tx.ExecContext(
			ctx,
			p.db.Rebind(qUpd),
			objn,
			lat,
			lon,
			gpsID,
		); err != nil {
			return fmt.Errorf("failed to update object coordinates: %w", err)
		}
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit coordinates save: %w", err)
	}
	return nil
}

func normalizeAdminObjectCard(card AdminObjectCard) (AdminObjectCard, error) {
	n := card

	n.ShortName = strings.TrimSpace(n.ShortName)
	n.FullName = strings.TrimSpace(n.FullName)
	n.Address = strings.TrimSpace(n.Address)
	n.Phones = strings.TrimSpace(n.Phones)
	n.Contract = strings.TrimSpace(n.Contract)
	n.StartDate = strings.TrimSpace(n.StartDate)
	n.Location = strings.TrimSpace(n.Location)
	n.Notes = strings.TrimSpace(n.Notes)
	n.GSMPhone1 = normalizeUASimPhone(n.GSMPhone1)
	n.GSMPhone2 = normalizeUASimPhone(n.GSMPhone2)
	n.SubServerA = strings.TrimSpace(n.SubServerA)
	n.SubServerB = strings.TrimSpace(n.SubServerB)

	if n.ObjN < 100 || n.ObjN > 99999 {
		return AdminObjectCard{}, fmt.Errorf("object number must be in range 100..99999")
	}
	if n.ShortName == "" {
		return AdminObjectCard{}, fmt.Errorf("object short name is empty")
	}
	if n.FullName == "" {
		n.FullName = n.ShortName
	}
	if n.ObjTypeID <= 0 {
		return AdminObjectCard{}, fmt.Errorf("object type is required")
	}
	if n.GrpN <= 0 {
		n.GrpN = 1
	}
	if n.ChannelCode < 0 {
		return AdminObjectCard{}, fmt.Errorf("invalid channel code")
	}
	if n.GSMHiddenN < 0 || n.GSMHiddenN > 9999 {
		return AdminObjectCard{}, fmt.Errorf("invalid hidden GPRS number")
	}
	if n.ChannelCode == 5 {
		if n.GSMHiddenN <= 0 {
			return AdminObjectCard{}, fmt.Errorf("hidden GPRS number is required for channel 5")
		}
	} else {
		n.GSMHiddenN = 0
	}
	if n.PPKID < 0 {
		return AdminObjectCard{}, fmt.Errorf("invalid PPK id")
	}

	if n.TestControlEnabled {
		if n.TestIntervalMin <= 0 {
			n.TestIntervalMin = 9
		}
		if n.TestIntervalMin > 1440 {
			n.TestIntervalMin = 1440
		}
	} else {
		n.TestIntervalMin = 0
	}

	return n, nil
}

func normalizeObjectPersonal(item AdminObjectPersonal) (AdminObjectPersonal, error) {
	n := item
	n.Surname = strings.TrimSpace(n.Surname)
	n.Name = strings.TrimSpace(n.Name)
	n.SecName = strings.TrimSpace(n.SecName)
	n.Address = strings.TrimSpace(n.Address)
	n.Phones = normalizeUserPhones(n.Phones)
	n.Position = strings.TrimSpace(n.Position)
	n.Notes = strings.TrimSpace(n.Notes)
	n.ViberID = strings.TrimSpace(n.ViberID)
	n.TelegramID = strings.TrimSpace(n.TelegramID)
	if n.Access1 > 0 {
		n.Access1 = 1
	} else {
		n.Access1 = 0
	}

	if n.Number < 0 || n.Number > 999 {
		return AdminObjectPersonal{}, fmt.Errorf("personal number must be in range 0..999")
	}
	if n.Surname == "" && n.Name == "" && n.SecName == "" {
		return AdminObjectPersonal{}, fmt.Errorf("personal full name is empty")
	}

	return n, nil
}

func applyPersonalObjectNumberFallback(item AdminObjectPersonal, objn int64) AdminObjectPersonal {
	objnText := strings.TrimSpace(strconv.FormatInt(objn, 10))
	if objnText == "" || objn <= 0 {
		return item
	}

	surname := strings.TrimSpace(item.Surname)
	name := strings.TrimSpace(item.Name)
	secName := strings.TrimSpace(item.SecName)

	if surname == "" {
		surname = objnText
	}
	if secName == "" {
		secName = objnText
	}
	// Якщо користувач залишив лише телефон (усі ПІБ порожні), заповнюємо також ім'я.
	if strings.TrimSpace(item.Surname) == "" && strings.TrimSpace(item.SecName) == "" && name == "" {
		name = objnText
	}

	item.Surname = surname
	item.Name = name
	item.SecName = secName
	return item
}

func normalizeObjectZone(zone AdminObjectZone) (AdminObjectZone, error) {
	n := zone
	n.Description = strings.TrimSpace(n.Description)

	if n.ZoneNumber <= 0 || n.ZoneNumber > 9999 {
		return AdminObjectZone{}, fmt.Errorf("zone number must be in range 1..9999")
	}
	// За поточним ТЗ тип зони фіксований: "пож.".
	n.ZoneType = 1
	// За поточним ТЗ поле RESBIGINT1 (затримка) не використовуємо і лишаємо NULL.
	n.EntryDelaySec = 0
	if n.Description == "" {
		n.Description = fmt.Sprintf("Шлейф %d", n.ZoneNumber)
	}

	return n, nil
}

func (p *DBDataProvider) resolveObjectRefByObjN(ctx context.Context, objn int64) (objectRefRow, error) {
	const q = `
		SELECT FIRST 1
			OBJUIN,
			COALESCE(GRPN, 1) AS GRPN
		FROM OBJECTS_INFO
		WHERE OBJN = ?
	`
	var objRef objectRefRow
	if err := p.db.GetContext(ctx, &objRef, p.db.Rebind(q), objn); err != nil {
		if err == sql.ErrNoRows {
			return objectRefRow{}, fmt.Errorf("object #%d not found", objn)
		}
		return objectRefRow{}, fmt.Errorf("failed to resolve object ref: %w", err)
	}
	return objRef, nil
}

func cleanupOrphanObjectMirrorRowsByObjNTx(ctx context.Context, tx *sqlx.Tx, db *sqlx.DB, objn int64) error {
	if objn <= 0 {
		return fmt.Errorf("invalid object number")
	}

	queries := []string{
		`
			DELETE FROM OBJECTS_LA la
			WHERE la.OBJN = ?
			  AND NOT EXISTS (
				SELECT 1 FROM OBJECTS_INFO oi WHERE oi.OBJUIN = la.OBJUIN
			  )
		`,
		`
			DELETE FROM OBJECTS_STATE os
			WHERE os.OBJN = ?
			  AND NOT EXISTS (
				SELECT 1 FROM OBJECTS_INFO oi WHERE oi.OBJUIN = os.OBJUIN
			  )
		`,
	}

	for _, q := range queries {
		if _, err := tx.ExecContext(ctx, db.Rebind(q), objn); err != nil {
			return fmt.Errorf("failed to cleanup orphan mirror rows for object #%d: %w", objn, err)
		}
	}

	return nil
}

func (p *DBDataProvider) notifyObjectCatalogChanged() {
	if p == nil || p.db == nil {
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), adminQueryTimeout)
	defer cancel()

	const q = `EXECUTE PROCEDURE MESS_CHANGE_OBJECT`
	if _, err := p.db.ExecContext(ctx, p.db.Rebind(q)); err != nil {
		log.Warn().Err(err).Msg("failed to notify MESS_CHANGE_OBJECT")
	}
}

func resolveObjectRefByObjNTx(ctx context.Context, tx *sqlx.Tx, db *sqlx.DB, objn int64) (objectRefRow, error) {
	const q = `
		SELECT FIRST 1
			OBJUIN,
			COALESCE(GRPN, 1) AS GRPN
		FROM OBJECTS_INFO
		WHERE OBJN = ?
	`
	var objRef objectRefRow
	if err := tx.GetContext(ctx, &objRef, db.Rebind(q), objn); err != nil {
		if err == sql.ErrNoRows {
			return objectRefRow{}, fmt.Errorf("object #%d not found", objn)
		}
		return objectRefRow{}, fmt.Errorf("failed to resolve object ref: %w", err)
	}
	return objRef, nil
}

func hasDuplicateZoneNumber(ctx context.Context, tx *sqlx.Tx, db *sqlx.DB, objUIN int64, zoneNumber int64, excludeID int64) (bool, error) {
	const q = `
		SELECT COUNT(*)
		FROM ZONES
		WHERE OBJUIN = ?
		  AND ZONEN = ?
		  AND (? = 0 OR ID <> ?)
	`
	var cnt int64
	if err := tx.GetContext(ctx, &cnt, db.Rebind(q), objUIN, zoneNumber, excludeID, excludeID); err != nil {
		return false, fmt.Errorf("failed to check duplicate zone number: %w", err)
	}
	return cnt > 0, nil
}

func ppkCatalogToStoredID(ppkID int64) int64 {
	if ppkID <= 0 {
		return 0
	}
	if ppkID >= 100 {
		return ppkID
	}
	return ppkID + 100
}

func ppkStoredToCatalogID(stored int64) int64 {
	if stored <= 0 {
		return 0
	}
	if stored >= 100 {
		return stored - 100
	}
	return stored
}

func resolveDefaultRegionIDTx(ctx context.Context, tx *sqlx.Tx, db *sqlx.DB, requested int64) (int64, error) {
	if requested > 0 {
		var exists int64
		if err := tx.GetContext(ctx, &exists, db.Rebind(`SELECT COUNT(*) FROM OBJREGS WHERE ID = ?`), requested); err != nil {
			return 0, fmt.Errorf("failed to validate region id: %w", err)
		}
		if exists > 0 {
			return requested, nil
		}
	}

	// За замовчуванням намагаємося використовувати район з ID=1.
	var regionOneExists int64
	if err := tx.GetContext(ctx, &regionOneExists, db.Rebind(`SELECT COUNT(*) FROM OBJREGS WHERE ID = 1`)); err != nil {
		return 0, fmt.Errorf("failed to validate default region id=1: %w", err)
	}
	if regionOneExists > 0 {
		return 1, nil
	}

	// Для сумісності з оригінальним МОСТ (inner join OBJECTS_INFO.OBJREGID = OBJREGS.ID)
	// об'єкт мусить мати валідний регіон. Беремо мінімальний ID з OBJREGS.
	var defaultRegionID int64
	if err := tx.GetContext(ctx, &defaultRegionID, db.Rebind(`SELECT FIRST 1 ID FROM OBJREGS ORDER BY ID`)); err != nil {
		if err == sql.ErrNoRows {
			return 0, fmt.Errorf("no regions found in OBJREGS")
		}
		return 0, fmt.Errorf("failed to resolve default region id: %w", err)
	}
	if defaultRegionID <= 0 {
		return 0, fmt.Errorf("invalid default region id")
	}
	return defaultRegionID, nil
}

func nullableInt64(v int64) any {
	if v <= 0 {
		return nil
	}
	return v
}

func nullableTrimmedString(v string) any {
	v = strings.TrimSpace(v)
	if v == "" {
		return nil
	}
	return v
}

func trimmedOrEmptyString(v string) string {
	return strings.TrimSpace(v)
}

func coordinateOrZero(v string) string {
	v = strings.TrimSpace(v)
	if v == "" {
		return "0"
	}
	return v
}

func normalizeUASimPhone(raw string) string {
	digits := utils.DigitsOnly(raw)
	if digits == "" {
		return ""
	}
	if n, ok := normalizeUAMobileDigits(digits); ok {
		return n
	}
	return digits
}

func normalizeUserPhones(raw string) string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return ""
	}

	parts := strings.FieldsFunc(raw, func(r rune) bool {
		return r == ',' || r == ';' || r == '\n' || r == '\r'
	})
	if len(parts) == 0 {
		parts = []string{raw}
	}

	out := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p == "" {
			continue
		}
		digits := utils.DigitsOnly(p)
		if n, ok := normalizeUAMobileDigits(digits); ok {
			out = append(out, formatUAPhoneDisplay(n))
			continue
		}
		out = append(out, p)
	}

	return strings.Join(out, ", ")
}

func extractNormalizedUAPhones(raw string) map[string]struct{} {
	result := map[string]struct{}{}
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return result
	}

	parts := strings.FieldsFunc(raw, func(r rune) bool {
		return r == ',' || r == ';' || r == '\n' || r == '\r'
	})
	if len(parts) == 0 {
		parts = []string{raw}
	}

	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p == "" {
			continue
		}
		n := normalizeUASimPhone(p)
		if len(n) == 10 && strings.HasPrefix(n, "0") {
			result[n] = struct{}{}
		}
	}

	// Випадок, коли в полі лише один номер, але з пробілами/дужками.
	n := normalizeUASimPhone(raw)
	if len(n) == 10 && strings.HasPrefix(n, "0") {
		result[n] = struct{}{}
	}

	return result
}

func normalizeUAMobileDigits(d string) (string, bool) {
	if d == "" {
		return "", false
	}

	switch {
	case len(d) == 10 && strings.HasPrefix(d, "0"):
		return d, true
	case len(d) == 12 && strings.HasPrefix(d, "380"):
		return "0" + d[3:], true
	case len(d) == 13 && strings.HasPrefix(d, "0380"):
		return "0" + d[4:], true
	case len(d) == 11 && strings.HasPrefix(d, "80"):
		return "0" + d[2:], true
	case len(d) == 9:
		return "0" + d, true
	case len(d) > 12:
		tail := d[len(d)-12:]
		if strings.HasPrefix(tail, "380") {
			return "0" + tail[3:], true
		}
	}

	return "", false
}

func formatUAPhoneDisplay(normalized string) string {
	if len(normalized) != 10 || !strings.HasPrefix(normalized, "0") {
		return normalized
	}
	return fmt.Sprintf("(%s) %s-%s-%s", normalized[0:3], normalized[3:6], normalized[6:8], normalized[8:10])
}

func formatDateTimePtr(t *time.Time) string {
	if t == nil || t.IsZero() {
		return ""
	}
	return t.Format("2006-01-02 15:04:05")
}
