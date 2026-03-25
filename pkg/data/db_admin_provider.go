package data

import (
	"context"
	"database/sql"
	"fmt"
	"sort"
	"strconv"
	"strings"
	"time"
	"unicode"

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

func (p *DBDataProvider) ListObjectTypes() ([]DictionaryItem, error) {
	ctx, cancel := context.WithTimeout(context.Background(), adminQueryTimeout)
	defer cancel()

	const q = `
		SELECT ID, OBJTYPE1
		FROM OBJTYPES
		ORDER BY OBJTYPE1
	`

	var rows []objTypeRow
	if err := p.db.SelectContext(ctx, &rows, p.db.Rebind(q)); err != nil {
		return nil, fmt.Errorf("failed to list object types: %w", err)
	}

	items := make([]DictionaryItem, 0, len(rows))
	for _, r := range rows {
		items = append(items, DictionaryItem{
			ID:   r.ID,
			Name: ptrToString(r.ObjType1),
		})
	}
	return items, nil
}

func (p *DBDataProvider) AddObjectType(name string) error {
	name = strings.TrimSpace(name)
	if name == "" {
		return fmt.Errorf("name is empty")
	}

	ctx, cancel := context.WithTimeout(context.Background(), adminQueryTimeout)
	defer cancel()

	const q = `INSERT INTO OBJTYPES (OBJTYPE1) VALUES (?)`
	if _, err := p.db.ExecContext(ctx, p.db.Rebind(q), name); err != nil {
		return fmt.Errorf("failed to add object type: %w", err)
	}
	return nil
}

func (p *DBDataProvider) UpdateObjectType(id int64, name string) error {
	name = strings.TrimSpace(name)
	if id <= 0 {
		return fmt.Errorf("invalid object type id")
	}
	if name == "" {
		return fmt.Errorf("name is empty")
	}

	ctx, cancel := context.WithTimeout(context.Background(), adminQueryTimeout)
	defer cancel()

	const q = `UPDATE OBJTYPES SET OBJTYPE1 = ? WHERE ID = ?`
	if _, err := p.db.ExecContext(ctx, p.db.Rebind(q), name, id); err != nil {
		return fmt.Errorf("failed to update object type: %w", err)
	}
	return nil
}

func (p *DBDataProvider) DeleteObjectType(id int64) error {
	if id <= 0 {
		return fmt.Errorf("invalid object type id")
	}

	ctx, cancel := context.WithTimeout(context.Background(), adminQueryTimeout)
	defer cancel()

	const q = `DELETE FROM OBJTYPES WHERE ID = ?`
	if _, err := p.db.ExecContext(ctx, p.db.Rebind(q), id); err != nil {
		return fmt.Errorf("failed to delete object type: %w", err)
	}
	return nil
}

func (p *DBDataProvider) ListRegions() ([]DictionaryItem, error) {
	ctx, cancel := context.WithTimeout(context.Background(), adminQueryTimeout)
	defer cancel()

	const q = `
		SELECT ID, RG_IND, RG_NAME
		FROM DCT_REGION
		ORDER BY RG_NAME
	`

	var rows []regionRow
	if err := p.db.SelectContext(ctx, &rows, p.db.Rebind(q)); err != nil {
		return nil, fmt.Errorf("failed to list regions: %w", err)
	}

	items := make([]DictionaryItem, 0, len(rows))
	for _, r := range rows {
		items = append(items, DictionaryItem{
			ID:   r.ID,
			Name: ptrToString(r.RgName),
			Code: r.RgInd,
		})
	}
	return items, nil
}

func (p *DBDataProvider) AddRegion(name string, regionCode *int64) error {
	name = strings.TrimSpace(name)
	if name == "" {
		return fmt.Errorf("region name is empty")
	}

	ctx, cancel := context.WithTimeout(context.Background(), adminQueryTimeout)
	defer cancel()

	if regionCode == nil {
		const q = `INSERT INTO DCT_REGION (RG_NAME) VALUES (?)`
		if _, err := p.db.ExecContext(ctx, p.db.Rebind(q), name); err != nil {
			return fmt.Errorf("failed to add region: %w", err)
		}
		return nil
	}

	const q = `INSERT INTO DCT_REGION (RG_IND, RG_NAME) VALUES (?, ?)`
	if _, err := p.db.ExecContext(ctx, p.db.Rebind(q), *regionCode, name); err != nil {
		return fmt.Errorf("failed to add region: %w", err)
	}
	return nil
}

func (p *DBDataProvider) UpdateRegion(id int64, name string, regionCode *int64) error {
	name = strings.TrimSpace(name)
	if id <= 0 {
		return fmt.Errorf("invalid region id")
	}
	if name == "" {
		return fmt.Errorf("region name is empty")
	}

	ctx, cancel := context.WithTimeout(context.Background(), adminQueryTimeout)
	defer cancel()

	if regionCode == nil {
		const q = `UPDATE DCT_REGION SET RG_NAME = ?, RG_IND = NULL WHERE ID = ?`
		if _, err := p.db.ExecContext(ctx, p.db.Rebind(q), name, id); err != nil {
			return fmt.Errorf("failed to update region: %w", err)
		}
		return nil
	}

	const q = `UPDATE DCT_REGION SET RG_NAME = ?, RG_IND = ? WHERE ID = ?`
	if _, err := p.db.ExecContext(ctx, p.db.Rebind(q), name, *regionCode, id); err != nil {
		return fmt.Errorf("failed to update region: %w", err)
	}
	return nil
}

func (p *DBDataProvider) DeleteRegion(id int64) error {
	if id <= 0 {
		return fmt.Errorf("invalid region id")
	}

	ctx, cancel := context.WithTimeout(context.Background(), adminQueryTimeout)
	defer cancel()

	const q = `DELETE FROM DCT_REGION WHERE ID = ?`
	if _, err := p.db.ExecContext(ctx, p.db.Rebind(q), id); err != nil {
		return fmt.Errorf("failed to delete region: %w", err)
	}
	return nil
}

func (p *DBDataProvider) ListObjectDistricts() ([]DictionaryItem, error) {
	ctx, cancel := context.WithTimeout(context.Background(), adminQueryTimeout)
	defer cancel()

	const q = `
		SELECT ID, REG1
		FROM OBJREGS
		ORDER BY REG1
	`

	var rows []objectDistrictRow
	if err := p.db.SelectContext(ctx, &rows, p.db.Rebind(q)); err != nil {
		return nil, fmt.Errorf("failed to list object districts: %w", err)
	}

	items := make([]DictionaryItem, 0, len(rows))
	for _, r := range rows {
		items = append(items, DictionaryItem{
			ID:   r.ID,
			Name: ptrToString(r.Reg1),
		})
	}
	return items, nil
}

func (p *DBDataProvider) ListAlarmReasons() ([]DictionaryItem, error) {
	ctx, cancel := context.WithTimeout(context.Background(), adminQueryTimeout)
	defer cancel()

	const q = `
		SELECT ID, REASON1
		FROM ALARMREAS
		ORDER BY ID
	`

	var rows []alarmReasonRow
	if err := p.db.SelectContext(ctx, &rows, p.db.Rebind(q)); err != nil {
		return nil, fmt.Errorf("failed to list alarm reasons: %w", err)
	}

	items := make([]DictionaryItem, 0, len(rows))
	for _, r := range rows {
		items = append(items, DictionaryItem{
			ID:   r.ID,
			Name: ptrToString(r.Reason1),
		})
	}
	return items, nil
}

func (p *DBDataProvider) AddAlarmReason(name string) error {
	name = strings.TrimSpace(name)
	if name == "" {
		return fmt.Errorf("reason is empty")
	}

	ctx, cancel := context.WithTimeout(context.Background(), adminQueryTimeout)
	defer cancel()

	const q = `INSERT INTO ALARMREAS (REASON1) VALUES (?)`
	if _, err := p.db.ExecContext(ctx, p.db.Rebind(q), name); err != nil {
		return fmt.Errorf("failed to add alarm reason: %w", err)
	}
	return nil
}

func (p *DBDataProvider) UpdateAlarmReason(id int64, name string) error {
	name = strings.TrimSpace(name)
	if id <= 0 {
		return fmt.Errorf("invalid reason id")
	}
	if name == "" {
		return fmt.Errorf("reason is empty")
	}

	ctx, cancel := context.WithTimeout(context.Background(), adminQueryTimeout)
	defer cancel()

	const q = `UPDATE ALARMREAS SET REASON1 = ? WHERE ID = ?`
	if _, err := p.db.ExecContext(ctx, p.db.Rebind(q), name, id); err != nil {
		return fmt.Errorf("failed to update alarm reason: %w", err)
	}
	return nil
}

func (p *DBDataProvider) DeleteAlarmReason(id int64) error {
	if id <= 0 {
		return fmt.Errorf("invalid reason id")
	}

	ctx, cancel := context.WithTimeout(context.Background(), adminQueryTimeout)
	defer cancel()

	const q = `DELETE FROM ALARMREAS WHERE ID = ?`
	if _, err := p.db.ExecContext(ctx, p.db.Rebind(q), id); err != nil {
		return fmt.Errorf("failed to delete alarm reason: %w", err)
	}
	return nil
}

func (p *DBDataProvider) MoveAlarmReason(id int64, direction int) error {
	if direction == 0 {
		return nil
	}

	items, err := p.ListAlarmReasons()
	if err != nil {
		return err
	}
	if len(items) < 2 {
		return nil
	}

	idx := -1
	for i := range items {
		if items[i].ID == id {
			idx = i
			break
		}
	}
	if idx == -1 {
		return fmt.Errorf("reason id %d not found", id)
	}

	targetIdx := idx + direction
	if targetIdx < 0 || targetIdx >= len(items) {
		return nil
	}

	idA := items[idx].ID
	idB := items[targetIdx].ID
	if idA == idB {
		return nil
	}

	ctx, cancel := context.WithTimeout(context.Background(), adminQueryTimeout)
	defer cancel()

	tx, err := p.db.BeginTxx(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to begin transaction for move: %w", err)
	}
	defer func() {
		_ = tx.Rollback()
	}()

	tempID := int64(-1)
	for {
		var exists int
		if err := tx.GetContext(ctx, &exists, p.db.Rebind(`SELECT COUNT(*) FROM ALARMREAS WHERE ID = ?`), tempID); err != nil {
			return fmt.Errorf("failed to check temporary id: %w", err)
		}
		if exists == 0 {
			break
		}
		tempID--
	}

	if _, err := tx.ExecContext(ctx, p.db.Rebind(`UPDATE ALARMREAS SET ID = ? WHERE ID = ?`), tempID, idA); err != nil {
		return fmt.Errorf("failed to move reason step 1: %w", err)
	}
	if _, err := tx.ExecContext(ctx, p.db.Rebind(`UPDATE ALARMREAS SET ID = ? WHERE ID = ?`), idA, idB); err != nil {
		return fmt.Errorf("failed to move reason step 2: %w", err)
	}
	if _, err := tx.ExecContext(ctx, p.db.Rebind(`UPDATE ALARMREAS SET ID = ? WHERE ID = ?`), idB, tempID); err != nil {
		return fmt.Errorf("failed to move reason step 3: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit reason move: %w", err)
	}
	return nil
}

func (p *DBDataProvider) ListPPKConstructor() ([]PPKConstructorItem, error) {
	ctx, cancel := context.WithTimeout(context.Background(), adminQueryTimeout)
	defer cancel()

	const q = `
		SELECT
			ID,
			PANELMARK1,
			COALESCE(ZONESCOUNT1, 0) AS ZONESCOUNT1,
			COALESCE(RESBYTE1, 0) AS RESBYTE1
		FROM PPK
		ORDER BY ID
	`

	var rows []ppkConstructorRow
	if err := p.db.SelectContext(ctx, &rows, p.db.Rebind(q)); err != nil {
		return nil, fmt.Errorf("failed to list PPK constructor items: %w", err)
	}

	items := make([]PPKConstructorItem, 0, len(rows))
	for _, r := range rows {
		items = append(items, PPKConstructorItem{
			ID:        r.ID,
			Name:      ptrToString(r.PanelMark1),
			Channel:   ptrToInt64(r.ChannelCode),
			ZoneCount: ptrToInt64(r.ZoneCount1),
		})
	}
	return items, nil
}

func (p *DBDataProvider) AddPPKConstructor(name string, channel int64, zoneCount int64) error {
	name = strings.TrimSpace(name)
	if name == "" {
		return fmt.Errorf("PPK name is empty")
	}
	if channel < 0 {
		return fmt.Errorf("invalid channel code")
	}
	if zoneCount <= 0 {
		return fmt.Errorf("zone count must be > 0")
	}

	ctx, cancel := context.WithTimeout(context.Background(), adminQueryTimeout)
	defer cancel()

	const q = `
		INSERT INTO PPK (
			PANELMARK1, PANELTYPE0, ZONESCOUNT1, RELAYCOUNT1, PGMCOUNT1, INCOUNT1,
			ASPTOUTCOUNT1, OPOVOUTCOUNT1, VENTOUTCOUNT1, CONTROLOUTCOUNT1, RESBYTE1
		)
		VALUES (?, 0, ?, 0, 0, 0, 0, 0, 0, 0, ?)
	`
	if _, err := p.db.ExecContext(ctx, p.db.Rebind(q), name, zoneCount, channel); err != nil {
		return fmt.Errorf("failed to add PPK constructor item: %w", err)
	}
	return nil
}

func (p *DBDataProvider) UpdatePPKConstructor(id int64, name string, channel int64, zoneCount int64) error {
	name = strings.TrimSpace(name)
	if id <= 0 {
		return fmt.Errorf("invalid PPK id")
	}
	if name == "" {
		return fmt.Errorf("PPK name is empty")
	}
	if channel < 0 {
		return fmt.Errorf("invalid channel code")
	}
	if zoneCount <= 0 {
		return fmt.Errorf("zone count must be > 0")
	}

	ctx, cancel := context.WithTimeout(context.Background(), adminQueryTimeout)
	defer cancel()

	const q = `
		UPDATE PPK
		SET PANELMARK1 = ?, ZONESCOUNT1 = ?, RESBYTE1 = ?
		WHERE ID = ?
	`
	if _, err := p.db.ExecContext(ctx, p.db.Rebind(q), name, zoneCount, channel, id); err != nil {
		return fmt.Errorf("failed to update PPK constructor item: %w", err)
	}
	return nil
}

func (p *DBDataProvider) DeletePPKConstructor(id int64) error {
	if id <= 0 {
		return fmt.Errorf("invalid PPK id")
	}

	ctx, cancel := context.WithTimeout(context.Background(), adminQueryTimeout)
	defer cancel()

	const q = `DELETE FROM PPK WHERE ID = ?`
	if _, err := p.db.ExecContext(ctx, p.db.Rebind(q), id); err != nil {
		return fmt.Errorf("failed to delete PPK constructor item: %w", err)
	}
	return nil
}

func (p *DBDataProvider) ListSubServers() ([]AdminSubServer, error) {
	ctx, cancel := context.WithTimeout(context.Background(), adminQueryTimeout)
	defer cancel()

	const q = `
		SELECT
			ID,
			SBINFO,
			SBIND,
			SBHOST,
			COALESCE(SBTYPE, 0) AS SBTYPE,
			SBHOST2
		FROM SBS
		ORDER BY ID
	`

	var rows []subServerRow
	if err := p.db.SelectContext(ctx, &rows, p.db.Rebind(q)); err != nil {
		return nil, fmt.Errorf("failed to list subservers: %w", err)
	}

	items := make([]AdminSubServer, 0, len(rows))
	for _, r := range rows {
		items = append(items, AdminSubServer{
			ID:    r.ID,
			Info:  strings.TrimSpace(ptrToString(r.SBInfo)),
			Bind:  strings.TrimSpace(ptrToString(r.SBInd)),
			Host:  strings.TrimSpace(ptrToString(r.SBHost)),
			Type:  ptrToInt64(r.SBType),
			Host2: strings.TrimSpace(ptrToString(r.SBHost2)),
		})
	}

	return items, nil
}

func (p *DBDataProvider) ListSubServerObjects(filter string) ([]AdminSubServerObject, error) {
	ctx, cancel := context.WithTimeout(context.Background(), adminQueryTimeout)
	defer cancel()

	filter = strings.TrimSpace(filter)
	args := make([]interface{}, 0, 4)
	query := `
		SELECT
			OBJN,
			OBJSHORTNAME1,
			ADDRESS1,
			SBSA,
			SBSB
		FROM OBJECTS_INFO
		WHERE OBJN > 0
	`
	if filter != "" {
		query += `
			AND (
				CAST(OBJN AS VARCHAR(20)) CONTAINING ?
				OR COALESCE(OBJSHORTNAME1, '') CONTAINING ?
				OR COALESCE(ADDRESS1, '') CONTAINING ?
			)
		`
		args = append(args, filter, filter, filter)
	}
	query += `
		ORDER BY
			substring(OBJN FROM 1 FOR 4),
			OBJN
	`

	var rows []subServerObjectRow
	if err := p.db.SelectContext(ctx, &rows, p.db.Rebind(query), args...); err != nil {
		return nil, fmt.Errorf("failed to list subserver objects: %w", err)
	}

	items := make([]AdminSubServerObject, 0, len(rows))
	for _, r := range rows {
		items = append(items, AdminSubServerObject{
			ObjN:       ptrToInt64(r.ObjN),
			Name:       strings.TrimSpace(ptrToString(r.ObjShortName)),
			Address:    strings.TrimSpace(ptrToString(r.Address1)),
			SubServerA: strings.TrimSpace(ptrToString(r.SBSA)),
			SubServerB: strings.TrimSpace(ptrToString(r.SBSB)),
		})
	}
	return items, nil
}

func (p *DBDataProvider) SetObjectSubServer(objn int64, channel int, bind string) error {
	if objn <= 0 {
		return fmt.Errorf("invalid object number")
	}
	if channel != 1 && channel != 2 {
		return fmt.Errorf("invalid subserver channel")
	}

	bind = strings.TrimSpace(bind)
	if bind == "" {
		return fmt.Errorf("subserver bind is empty")
	}

	ctx, cancel := context.WithTimeout(context.Background(), adminQueryTimeout)
	defer cancel()

	tx, err := p.db.BeginTxx(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to begin subserver bind transaction: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	objRef, err := resolveObjectRefByObjNTx(ctx, tx, p.db, objn)
	if err != nil {
		return err
	}

	var exists int64
	if err := tx.GetContext(ctx, &exists, p.db.Rebind(`SELECT COUNT(*) FROM SBS WHERE TRIM(SBIND) = ?`), bind); err != nil {
		return fmt.Errorf("failed to validate subserver bind: %w", err)
	}
	if exists <= 0 {
		return fmt.Errorf("subserver bind %q not found", bind)
	}

	col := "SBSA"
	if channel == 2 {
		col = "SBSB"
	}
	q := fmt.Sprintf("UPDATE OBJECTS_INFO SET %s = ? WHERE OBJUIN = ?", col)
	if _, err := tx.ExecContext(ctx, p.db.Rebind(q), bind, objRef.ObjUin); err != nil {
		return fmt.Errorf("failed to set object subserver: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit object subserver change: %w", err)
	}
	p.notifyObjectCatalogChanged()
	return nil
}

func (p *DBDataProvider) ClearObjectSubServer(objn int64, channel int) error {
	if objn <= 0 {
		return fmt.Errorf("invalid object number")
	}
	if channel != 1 && channel != 2 {
		return fmt.Errorf("invalid subserver channel")
	}

	ctx, cancel := context.WithTimeout(context.Background(), adminQueryTimeout)
	defer cancel()

	tx, err := p.db.BeginTxx(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to begin subserver clear transaction: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	objRef, err := resolveObjectRefByObjNTx(ctx, tx, p.db, objn)
	if err != nil {
		return err
	}

	col := "SBSA"
	if channel == 2 {
		col = "SBSB"
	}
	q := fmt.Sprintf("UPDATE OBJECTS_INFO SET %s = NULL WHERE OBJUIN = ?", col)
	if _, err := tx.ExecContext(ctx, p.db.Rebind(q), objRef.ObjUin); err != nil {
		return fmt.Errorf("failed to clear object subserver: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit object subserver clear: %w", err)
	}
	p.notifyObjectCatalogChanged()
	return nil
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
			oi.OBJTYPEID,
			oi.OBJREGID,
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

	const qUpdObj = `
		UPDATE OBJECTS
		SET
			OBJN = ?, GRPN = ?, OBJFULLNAME1 = ?, OBJSHORTNAME1 = ?, OBJTYPEID = ?,
			ADDRESS1 = ?, PHONES1 = ?, OBJREGID = ?, CONTRACT1 = ?, LOCATION1 = ?, NOTES1 = ?,
			TESTCONTROL1 = ?, TESTTIME1 = ?, PPKID = ?, RESERVTEXT = ?, GSMPHONE = ?, GSMPHONE2 = ?, GSMHIDENINT = ?, OBJCHAN = ?
		WHERE OBJUIN = ?
	`
	_, err = tx.ExecContext(
		ctx,
		p.db.Rebind(qUpdObj),
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
		testControl,
		testInterval,
		storedPPKID,
		normalized.StartDate,
		normalized.GSMPhone1,
		normalized.GSMPhone2,
		nullableInt64(normalized.GSMHiddenN),
		normalized.ChannelCode,
		objUIN,
	)
	if err != nil {
		return fmt.Errorf("failed to update OBJECTS mirror row: %w", err)
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
		`DELETE FROM OBJECTS WHERE OBJUIN = ?`,
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
		`
			DELETE FROM OBJECTS o
			WHERE o.OBJN = ?
			  AND NOT EXISTS (
				SELECT 1 FROM OBJECTS_INFO oi WHERE oi.OBJUIN = o.OBJUIN
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

func nullableInt64(v int64) interface{} {
	if v <= 0 {
		return nil
	}
	return v
}

func nullableTrimmedString(v string) interface{} {
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
	digits := digitsOnly(raw)
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
		digits := digitsOnly(p)
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

func digitsOnly(s string) string {
	var b strings.Builder
	b.Grow(len(s))
	for _, r := range s {
		if unicode.IsDigit(r) {
			b.WriteRune(r)
		}
	}
	return b.String()
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

func (p *DBDataProvider) ListMessageProtocols() ([]int64, error) {
	ctx, cancel := context.WithTimeout(context.Background(), adminQueryTimeout)
	defer cancel()

	const q = `
		SELECT DISTINCT PROTID
		FROM MESSLIST
		WHERE PROTID IS NOT NULL
		ORDER BY PROTID
	`

	var raw []sql.NullInt64
	if err := p.db.SelectContext(ctx, &raw, p.db.Rebind(q)); err != nil {
		return nil, fmt.Errorf("failed to list protocols: %w", err)
	}

	protocols := make([]int64, 0, len(raw))
	for _, v := range raw {
		if v.Valid {
			protocols = append(protocols, v.Int64)
		}
	}
	return protocols, nil
}

func (p *DBDataProvider) ListMessages(protocolID *int64, filter string) ([]AdminMessage, error) {
	ctx, cancel := context.WithTimeout(context.Background(), adminQueryTimeout)
	defer cancel()

	filter = strings.TrimSpace(filter)

	query := `
		SELECT UIN, PROTID, MESSID, MESSIDHEX, UKR1, SC1, FORADMINONLY
		FROM MESSLIST
		WHERE 1 = 1
	`
	args := make([]interface{}, 0, 6)

	if protocolID != nil {
		query += ` AND PROTID = ?`
		args = append(args, *protocolID)
	}

	if filter != "" {
		query += ` AND (
			UKR1 CONTAINING ?
			OR MESSIDHEX CONTAINING ?
			OR CAST(MESSID AS VARCHAR(20)) CONTAINING ?
		)`
		args = append(args, filter, filter, filter)
	}

	// Сортуємо коди подій як числа (за MESSID, а якщо він порожній - за UIN),
	// щоб 2, 10, 100 не перемішувались як текст.
	query += ` ORDER BY COALESCE(MESSID, UIN), UIN`

	var rows []messageRow
	if err := p.db.SelectContext(ctx, &rows, p.db.Rebind(query), args...); err != nil {
		return nil, fmt.Errorf("failed to list messages: %w", err)
	}

	messages := make([]AdminMessage, 0, len(rows))
	for _, r := range rows {
		messages = append(messages, AdminMessage{
			UIN:          r.UIN,
			ProtocolID:   r.ProtID,
			MessageID:    r.MessID,
			MessageHex:   ptrToString(r.MessIDHex),
			Text:         ptrToString(r.Ukr1),
			SC1:          r.Sc1,
			ForAdminOnly: r.ForAdminOnly != nil && *r.ForAdminOnly != 0,
		})
	}
	return messages, nil
}

func (p *DBDataProvider) SetMessageAdminOnly(uin int64, adminOnly bool) error {
	if uin <= 0 {
		return fmt.Errorf("invalid message uin")
	}

	ctx, cancel := context.WithTimeout(context.Background(), adminQueryTimeout)
	defer cancel()

	flag := int16(0)
	if adminOnly {
		flag = 1
	}

	const q = `UPDATE MESSLIST SET FORADMINONLY = ? WHERE UIN = ?`
	if _, err := p.db.ExecContext(ctx, p.db.Rebind(q), flag, uin); err != nil {
		return fmt.Errorf("failed to update FORADMINONLY flag: %w", err)
	}
	return nil
}

func (p *DBDataProvider) SetMessageCategory(uin int64, sc1 *int64) error {
	if uin <= 0 {
		return fmt.Errorf("invalid message uin")
	}

	ctx, cancel := context.WithTimeout(context.Background(), adminQueryTimeout)
	defer cancel()

	if sc1 == nil {
		const q = `UPDATE MESSLIST SET SC1 = NULL WHERE UIN = ?`
		if _, err := p.db.ExecContext(ctx, p.db.Rebind(q), uin); err != nil {
			return fmt.Errorf("failed to clear SC1: %w", err)
		}
		return nil
	}

	const q = `UPDATE MESSLIST SET SC1 = ? WHERE UIN = ?`
	if _, err := p.db.ExecContext(ctx, p.db.Rebind(q), *sc1, uin); err != nil {
		return fmt.Errorf("failed to update SC1: %w", err)
	}
	return nil
}

func (p *DBDataProvider) ListDisplayBlockObjects(filter string) ([]DisplayBlockObject, error) {
	ctx, cancel := context.WithTimeout(context.Background(), adminQueryTimeout)
	defer cancel()

	filter = strings.TrimSpace(filter)

	// Основа запиту така ж, як у GetObjectsList (OBJECTS_INFO + OBJECTS_LA + OBJECTS_STATE).
	// Режим налагодження визначаємо через OBJECTS_INFO.ENG1.
	// BLOCKEDARMED_ON_OFF лишаємо як fallback для сумісності зі старими схемами.
	query := `
		SELECT
			oi.OBJN,
			oi.OBJSHORTNAME1,
			COALESCE(os.ALARMSTATE1, 0) AS ALARMSTATE1,
			COALESCE(os.GUARDSTATE1, 0) AS GUARDSTATE1,
			COALESCE(os.TECHALARMSTATE1, 0) AS TECHALARMSTATE1,
			COALESCE(ol.ISCONNSTATE1, 0) AS ISCONNSTATE1,
			COALESCE(oi.ENG1, 0) AS ENG1,
			COALESCE(os.BLOCKEDARMED_ON_OFF, 0) AS BLOCKEDARMED_ON_OFF
		FROM OBJECTS_INFO oi
		JOIN OBJECTS_LA ol ON ol.OBJUIN = oi.OBJUIN
		JOIN OBJECTS_STATE os ON os.OBJUIN = oi.OBJUIN
		WHERE oi.OBJTYPEID <> 1
		  AND oi.OBJN > 1000
	`

	args := make([]interface{}, 0, 3)
	if filter != "" {
		query += ` AND (
			CAST(oi.OBJN AS VARCHAR(20)) CONTAINING ?
			OR oi.OBJSHORTNAME1 CONTAINING ?
		)`
		args = append(args, filter, filter)
	}
	query += ` ORDER BY oi.OBJN`

	var rows []displayBlockRow
	if err := p.db.SelectContext(ctx, &rows, p.db.Rebind(query), args...); err != nil {
		return nil, fmt.Errorf("failed to list display-block objects: %w", err)
	}

	items := make([]DisplayBlockObject, 0, len(rows))
	for _, r := range rows {
		objn := ptrToInt64(r.ObjN)
		mode := resolveDisplayBlockMode(r.GuardState1, r.Eng1, r.BlockedArmedOnOff)
		items = append(items, DisplayBlockObject{
			ObjN:           objn,
			Name:           ptrToString(r.ObjShortName1),
			BlockMode:      mode,
			AlarmState:     ptrToInt64(r.AlarmState1),
			GuardState:     ptrToInt64(r.GuardState1),
			TechAlarmState: ptrToInt64(r.TechAlarmState1),
			IsConnState:    ptrToInt64(r.IsConnState1),
		})
	}
	return items, nil
}

func resolveDisplayBlockMode(guardState *int64, eng1 *int64, blockedArmed *int16) DisplayBlockMode {
	// За ТЗ: GUARDSTATE1 = 0 означає "знято зі спостереження".
	if guardState != nil && *guardState == 0 {
		return DisplayBlockTemporaryOff
	}

	// Режим налагодження визначаємо через OBJECTS_INFO.ENG1 (ненульове).
	if eng1 != nil && *eng1 != 0 {
		return DisplayBlockDebug
	}

	// Fallback для сумісності зі старими базами.
	if blockedArmed != nil {
		mode := DisplayBlockMode(*blockedArmed)
		switch mode {
		case DisplayBlockTemporaryOff, DisplayBlockDebug:
			return mode
		}
		if *blockedArmed != 0 {
			return DisplayBlockTemporaryOff
		}
	}

	return DisplayBlockNone
}

func (p *DBDataProvider) SetDisplayBlockMode(objn int64, mode DisplayBlockMode) error {
	if objn <= 0 {
		return fmt.Errorf("invalid object number")
	}
	if objn <= 1000 {
		return fmt.Errorf("object number must be > 1000 for display blocking")
	}
	if mode < DisplayBlockNone || mode > DisplayBlockDebug {
		return fmt.Errorf("invalid display block mode: %d", mode)
	}

	ctx, cancel := context.WithTimeout(context.Background(), adminQueryTimeout)
	defer cancel()

	tx, err := p.db.BeginTxx(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to begin transaction for display-block update: %w", err)
	}
	defer func() {
		_ = tx.Rollback()
	}()

	objRef, err := resolveObjectRefByObjNTx(ctx, tx, p.db, objn)
	if err != nil {
		return err
	}

	var current displayBlockRow
	const qCurrent = `
		SELECT FIRST 1
			oi.OBJN,
			oi.OBJSHORTNAME1,
			COALESCE(os.GUARDSTATE1, 0) AS GUARDSTATE1,
			COALESCE(oi.ENG1, 0) AS ENG1,
			COALESCE(os.BLOCKEDARMED_ON_OFF, 0) AS BLOCKEDARMED_ON_OFF
		FROM OBJECTS_INFO oi
		LEFT JOIN OBJECTS_STATE os ON os.OBJUIN = oi.OBJUIN
		WHERE oi.OBJN = ?
	`
	if err := tx.GetContext(ctx, &current, p.db.Rebind(qCurrent), objn); err != nil {
		if err == sql.ErrNoRows {
			return fmt.Errorf("object #%d not found", objn)
		}
		return fmt.Errorf("failed to load current display-block mode: %w", err)
	}
	prevMode := resolveDisplayBlockMode(current.GuardState1, current.Eng1, current.BlockedArmedOnOff)

	guardState := int64(1)
	debugFlag := int64(0)
	blockedArmed := int64(0)
	switch mode {
	case DisplayBlockNone:
		guardState = 1
		debugFlag = 0
		blockedArmed = 0
	case DisplayBlockTemporaryOff:
		guardState = 0
		debugFlag = 0
		blockedArmed = 1
	case DisplayBlockDebug:
		guardState = 1
		debugFlag = 1
		blockedArmed = 2
	}

	totalAffected := int64(0)

	// 1) OBJECTS_STATE.GUARDSTATE1 + BLOCKEDARMED_ON_OFF.
	const qState = `
		UPDATE OBJECTS_STATE os
		SET
			os.GUARDSTATE1 = ?,
			os.BLOCKEDARMED_ON_OFF = ?
		WHERE os.OBJUIN IN (
			SELECT oi.OBJUIN
			FROM OBJECTS_INFO oi
			WHERE oi.OBJN = ?
		)
	`
	if res, execErr := tx.ExecContext(ctx, p.db.Rebind(qState), guardState, blockedArmed, objn); execErr != nil {
		return fmt.Errorf("failed to update OBJECTS_STATE guard/block mode: %w", execErr)
	} else if n, nErr := res.RowsAffected(); nErr == nil {
		totalAffected += n
	}

	// 2) OBJECTS_INFO.ENG1: прапорець режиму налагодження (1/0).
	const qInfo = `
		UPDATE OBJECTS_INFO oi
		SET oi.ENG1 = ?
		WHERE oi.OBJN = ?
	`
	if res, execErr := tx.ExecContext(ctx, p.db.Rebind(qInfo), debugFlag, objn); execErr != nil {
		return fmt.Errorf("failed to update OBJECTS_INFO.ENG1 debug flag: %w", execErr)
	} else if n, nErr := res.RowsAffected(); nErr == nil {
		totalAffected += n
	}

	// 3) Дзеркалимо стан в OBJECTS для сумісності з legacy-частинами системи.
	qObj := `
		UPDATE OBJECTS o
		SET
			o.GUARDSTATE1 = ?,
			o.ENG1 = ?
		WHERE o.OBJUIN IN (
			SELECT oi.OBJUIN
			FROM OBJECTS_INFO oi
			WHERE oi.OBJN = ?
		)
	`
	qObjArgs := []interface{}{guardState, debugFlag, objn}
	if res, execErr := tx.ExecContext(ctx, p.db.Rebind(qObj), qObjArgs...); execErr != nil {
		return fmt.Errorf("failed to update OBJECTS legacy guard/debug state: %w", execErr)
	} else if n, nErr := res.RowsAffected(); nErr == nil {
		totalAffected += n
	}

	if prevMode != mode {
		if err := insertDisplayBlockNotificationEventTx(ctx, tx, p.db, objRef, prevMode, mode); err != nil {
			return fmt.Errorf("failed to write display-block notification event: %w", err)
		}
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit display-block update: %w", err)
	}

	if totalAffected == 0 {
		log.Warn().Int64("objn", objn).Msg("SetDisplayBlockMode: object not found in OBJECTS_INFO/target tables")
	}
	return nil
}

func insertDisplayBlockNotificationEventTx(
	ctx context.Context,
	tx *sqlx.Tx,
	db *sqlx.DB,
	objRef objectRefRow,
	prevMode DisplayBlockMode,
	newMode DisplayBlockMode,
) error {
	// Коди повідомлень з MESSLIST (PROTID=1), які потрібні для сумісності з MOST:
	// 6  - Тимчасове зняття з пожежного спостереження
	// 7  - Повторна постановка під пожежне спостереження
	// 59 - Ввімкнення режиму налагодження
	// 60 - Вимкнення режиму налагодження
	var messID int64
	switch newMode {
	case DisplayBlockTemporaryOff:
		messID = 6
	case DisplayBlockDebug:
		messID = 59
	case DisplayBlockNone:
		switch prevMode {
		case DisplayBlockTemporaryOff:
			messID = 7
		case DisplayBlockDebug:
			messID = 60
		default:
			return nil
		}
	default:
		return nil
	}

	evUIN, err := resolveMessUINByProtAndMessIDTx(ctx, tx, db, 1, messID)
	if err != nil {
		return err
	}

	operator := resolveCurrentOperatorLabelTx(ctx, tx, db)
	group := int64(1)
	if objRef.GrpN != nil && *objRef.GrpN > 0 {
		group = *objRef.GrpN
	}

	const qIns = `
		INSERT INTO EVLOG (EVTIME1, OBJUIN, GRPN, ZONEN, EVUIN, INFO1)
		VALUES (CURRENT_TIMESTAMP, ?, ?, 0, ?, ?)
	`
	if _, err := tx.ExecContext(
		ctx,
		db.Rebind(qIns),
		objRef.ObjUin,
		group,
		evUIN,
		operator,
	); err != nil {
		return fmt.Errorf("failed to insert EVLOG notification (MESSID=%d, UIN=%d): %w", messID, evUIN, err)
	}

	return nil
}

func insertObjectCrudNotificationEventTx(
	ctx context.Context,
	tx *sqlx.Tx,
	db *sqlx.DB,
	objRef objectRefRow,
	messID int64,
) error {
	evUIN, err := resolveMessUINByProtAndMessIDTx(ctx, tx, db, 1, messID)
	if err != nil {
		return err
	}

	operator := resolveCurrentOperatorLabelTx(ctx, tx, db)
	group := int64(1)
	if objRef.GrpN != nil && *objRef.GrpN > 0 {
		group = *objRef.GrpN
	}

	const qIns = `
		INSERT INTO EVLOG (EVTIME1, OBJUIN, GRPN, ZONEN, EVUIN, INFO1)
		VALUES (CURRENT_TIMESTAMP, ?, ?, 0, ?, ?)
	`
	if _, err := tx.ExecContext(
		ctx,
		db.Rebind(qIns),
		objRef.ObjUin,
		group,
		evUIN,
		operator,
	); err != nil {
		return fmt.Errorf("failed to insert EVLOG CRUD notification (MESSID=%d, UIN=%d): %w", messID, evUIN, err)
	}

	return nil
}

func resolveMessUINByProtAndMessIDTx(ctx context.Context, tx *sqlx.Tx, db *sqlx.DB, protID int64, messID int64) (int64, error) {
	var uin int64
	const qByProt = `
		SELECT FIRST 1 UIN
		FROM MESSLIST
		WHERE PROTID = ?
		  AND MESSID = ?
		ORDER BY UIN
	`
	if err := tx.GetContext(ctx, &uin, db.Rebind(qByProt), protID, messID); err == nil {
		return uin, nil
	} else if err != sql.ErrNoRows {
		return 0, fmt.Errorf("failed to resolve message UIN by PROTID=%d/MESSID=%d: %w", protID, messID, err)
	}

	const qFallback = `
		SELECT FIRST 1 UIN
		FROM MESSLIST
		WHERE MESSID = ?
		ORDER BY UIN
	`
	if err := tx.GetContext(ctx, &uin, db.Rebind(qFallback), messID); err != nil {
		if err == sql.ErrNoRows {
			return 0, fmt.Errorf("message UIN not found for MESSID=%d", messID)
		}
		return 0, fmt.Errorf("failed to resolve message UIN fallback for MESSID=%d: %w", messID, err)
	}
	return uin, nil
}

func resolveCurrentOperatorLabelTx(ctx context.Context, tx *sqlx.Tx, db *sqlx.DB) string {
	type operatorRow struct {
		UserName *string `db:"USERNAME"`
		UserType *int64  `db:"USERTYPE"`
	}

	selectRole := func(userType int64) string {
		switch userType {
		case 1:
			return "Администратор"
		case 2:
			return "Старший оператор"
		default:
			return "Оператор"
		}
	}

	buildLabel := func(r operatorRow) string {
		name := strings.TrimSpace(ptrToString(r.UserName))
		role := selectRole(ptrToInt64(r.UserType))
		if name == "" {
			return role
		}
		return fmt.Sprintf("%s - %s", name, role)
	}

	// 1) Активний користувач (USERSTATE <> 0).
	var active operatorRow
	const qActive = `
		SELECT FIRST 1
			USERNAME,
			COALESCE(USERTYPE, 1) AS USERTYPE
		FROM USERSTATUS
		WHERE COALESCE(USERSTATE, 0) <> 0
		ORDER BY USERNAME
	`
	if err := tx.GetContext(ctx, &active, db.Rebind(qActive)); err == nil {
		return buildLabel(active)
	}

	// 2) Якщо активного немає - беремо персональний логін (не "Зміна X ...").
	var named operatorRow
	const qNamed = `
		SELECT FIRST 1
			USERNAME,
			COALESCE(USERTYPE, 1) AS USERTYPE
		FROM USERSTATUS
		WHERE TRIM(COALESCE(USERNAME, '')) <> ''
		  AND NOT (
			TRIM(USERNAME) STARTING WITH 'Зміна'
			OR TRIM(USERNAME) STARTING WITH 'ЗМІНА'
			OR TRIM(USERNAME) STARTING WITH 'Смена'
			OR TRIM(USERNAME) STARTING WITH 'СМЕНА'
		  )
		ORDER BY USERNAME
	`
	if err := tx.GetContext(ctx, &named, db.Rebind(qNamed)); err == nil {
		return buildLabel(named)
	}

	// 3) Fallback: перший доступний користувач.
	var any operatorRow
	const qAny = `
		SELECT FIRST 1
			USERNAME,
			COALESCE(USERTYPE, 1) AS USERTYPE
		FROM USERSTATUS
		ORDER BY USERNAME
	`
	if err := tx.GetContext(ctx, &any, db.Rebind(qAny)); err == nil {
		return buildLabel(any)
	}

	return "Администратор"
}

func (p *DBDataProvider) GetFireMonitoringSettings() (FireMonitoringSettings, error) {
	ctx, cancel := context.WithTimeout(context.Background(), adminQueryTimeout)
	defer cancel()

	const q = `
		SELECT ID, PARAM_NAME, PARAM_VALUE
		FROM PULT_INFO
		WHERE PARAM_NAME STARTING WITH 'FMON_'
		   OR PARAM_NAME IN ('DEHost', 'DEService', 'OH_IPPP')
		ORDER BY ID
	`

	var rows []pultInfoRow
	if err := p.db.SelectContext(ctx, &rows, p.db.Rebind(q)); err != nil {
		return FireMonitoringSettings{}, fmt.Errorf("failed to load fire monitoring settings: %w", err)
	}

	settings := FireMonitoringSettings{
		Enabled:       false,
		ObjectID:      "",
		AckWaitSec:    5,
		UseStdDateFmt: true,
		Servers:       []FireMonitoringServer{},
	}

	values := map[string]string{}
	for _, r := range rows {
		name := strings.ToUpper(strings.TrimSpace(ptrToString(r.ParamName)))
		if name == "" {
			continue
		}
		values[name] = strings.TrimSpace(ptrToString(r.ParamVal))
	}

	if v, ok := values["FMON_ENABLED"]; ok {
		settings.Enabled = parseSmallBool(v)
	}
	if v, ok := values["FMON_OBJID"]; ok {
		settings.ObjectID = v
	}
	if v, ok := values["FMON_ACKSEC"]; ok {
		if n, err := strconv.ParseInt(v, 10, 64); err == nil && n > 0 {
			settings.AckWaitSec = n
		}
	}
	if v, ok := values["FMON_DATESTD"]; ok {
		settings.UseStdDateFmt = parseSmallBool(v)
	}

	for i := 1; i <= 8; i++ {
		host := strings.TrimSpace(values[fireMonServerHostKey(i)])
		portRaw := strings.TrimSpace(values[fireMonServerPortKey(i)])
		info := strings.TrimSpace(values[fireMonServerInfoKey(i)])
		enabled := parseSmallBool(values[fireMonServerEnabledKey(i)])

		port := int64(0)
		if portRaw != "" {
			if n, err := strconv.ParseInt(portRaw, 10, 64); err == nil {
				port = n
			}
		}
		if host == "" && port == 0 && info == "" {
			continue
		}

		settings.Servers = append(settings.Servers, FireMonitoringServer{
			Host:    host,
			Port:    port,
			Info:    info,
			Enabled: enabled,
		})
	}

	// Fallback: у частині інсталяцій зберігаються тільки DEHost/DEService.
	if len(settings.Servers) == 0 {
		deHost := strings.TrimSpace(values["DEHOST"])
		dePort := parseInt64OrZero(values["DESERVICE"])
		if deHost != "" || dePort > 0 {
			settings.Servers = append(settings.Servers, FireMonitoringServer{
				Host:    deHost,
				Port:    dePort,
				Info:    "Основний",
				Enabled: true,
			})
		}

		// OH_IPPP часто зберігається як host:port.
		if host, port := parseHostPort(values["OH_IPPP"]); host != "" || port > 0 {
			if len(settings.Servers) == 0 || !sameServer(settings.Servers[0], host, port) {
				settings.Servers = append(settings.Servers, FireMonitoringServer{
					Host:    host,
					Port:    port,
					Info:    "Резервний",
					Enabled: true,
				})
			}
		}
	}

	if len(settings.Servers) == 0 {
		settings.Servers = []FireMonitoringServer{
			{Enabled: true},
		}
	}

	return settings, nil
}

func (p *DBDataProvider) SaveFireMonitoringSettings(settings FireMonitoringSettings) error {
	ctx, cancel := context.WithTimeout(context.Background(), adminQueryTimeout)
	defer cancel()

	normalized := normalizeFireMonitoringSettings(settings)

	tx, err := p.db.BeginTxx(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to begin fire-monitoring transaction: %w", err)
	}
	defer func() {
		_ = tx.Rollback()
	}()

	upsert := func(name string, value string) error {
		var id int64
		const qGet = `
			SELECT FIRST 1 ID
			FROM PULT_INFO
			WHERE PARAM_NAME = ?
			ORDER BY ID DESC
		`
		if err := tx.GetContext(ctx, &id, p.db.Rebind(qGet), name); err != nil {
			if err == sql.ErrNoRows {
				const qIns = `INSERT INTO PULT_INFO (PARAM_NAME, PARAM_VALUE) VALUES (?, ?)`
				if _, execErr := tx.ExecContext(ctx, p.db.Rebind(qIns), name, value); execErr != nil {
					return fmt.Errorf("insert %s: %w", name, execErr)
				}
				return nil
			}
			return fmt.Errorf("get %s: %w", name, err)
		}

		const qUpd = `UPDATE PULT_INFO SET PARAM_VALUE = ? WHERE ID = ?`
		if _, err := tx.ExecContext(ctx, p.db.Rebind(qUpd), value, id); err != nil {
			return fmt.Errorf("update %s: %w", name, err)
		}
		return nil
	}

	if err := upsert("FMON_ENABLED", boolToSmallIntText(normalized.Enabled)); err != nil {
		return err
	}
	if err := upsert("FMON_OBJID", strings.TrimSpace(normalized.ObjectID)); err != nil {
		return err
	}
	if err := upsert("FMON_ACKSEC", strconv.FormatInt(normalized.AckWaitSec, 10)); err != nil {
		return err
	}
	if err := upsert("FMON_DATESTD", boolToSmallIntText(normalized.UseStdDateFmt)); err != nil {
		return err
	}
	if err := upsert("FMON_SRVCNT", strconv.FormatInt(int64(len(normalized.Servers)), 10)); err != nil {
		return err
	}

	for i := 1; i <= 8; i++ {
		if i <= len(normalized.Servers) {
			s := normalized.Servers[i-1]
			if err := upsert(fireMonServerHostKey(i), s.Host); err != nil {
				return err
			}
			if err := upsert(fireMonServerPortKey(i), strconv.FormatInt(s.Port, 10)); err != nil {
				return err
			}
			if err := upsert(fireMonServerInfoKey(i), s.Info); err != nil {
				return err
			}
			if err := upsert(fireMonServerEnabledKey(i), boolToSmallIntText(s.Enabled)); err != nil {
				return err
			}
			continue
		}

		if err := upsert(fireMonServerHostKey(i), ""); err != nil {
			return err
		}
		if err := upsert(fireMonServerPortKey(i), "0"); err != nil {
			return err
		}
		if err := upsert(fireMonServerInfoKey(i), ""); err != nil {
			return err
		}
		if err := upsert(fireMonServerEnabledKey(i), "0"); err != nil {
			return err
		}
	}

	// Legacy-сумісність: дублюємо перший сервер в DEHost/DEService/OH_IPPP.
	if len(normalized.Servers) > 0 {
		mainSrv := normalized.Servers[0]
		if err := upsert("DEHost", mainSrv.Host); err != nil {
			return err
		}
		if err := upsert("DEService", strconv.FormatInt(mainSrv.Port, 10)); err != nil {
			return err
		}
		if err := upsert("OH_IPPP", fmt.Sprintf("%s:%d", mainSrv.Host, mainSrv.Port)); err != nil {
			return err
		}
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit fire-monitoring settings: %w", err)
	}
	return nil
}

func fireMonServerHostKey(i int) string {
	return fmt.Sprintf("FMON_S%d_HOST", i)
}

func fireMonServerPortKey(i int) string {
	return fmt.Sprintf("FMON_S%d_PORT", i)
}

func fireMonServerInfoKey(i int) string {
	return fmt.Sprintf("FMON_S%d_INFO", i)
}

func fireMonServerEnabledKey(i int) string {
	return fmt.Sprintf("FMON_S%d_EN", i)
}

func parseSmallBool(v string) bool {
	switch strings.TrimSpace(strings.ToLower(v)) {
	case "1", "true", "yes", "y", "on":
		return true
	default:
		return false
	}
}

func boolToSmallIntText(v bool) string {
	if v {
		return "1"
	}
	return "0"
}

func parseInt64OrZero(v string) int64 {
	v = strings.TrimSpace(v)
	if v == "" {
		return 0
	}
	n, err := strconv.ParseInt(v, 10, 64)
	if err != nil {
		return 0
	}
	return n
}

func parseHostPort(v string) (host string, port int64) {
	v = strings.TrimSpace(v)
	if v == "" {
		return "", 0
	}
	if idx := strings.LastIndex(v, ":"); idx > 0 && idx < len(v)-1 {
		host = strings.TrimSpace(v[:idx])
		port = parseInt64OrZero(v[idx+1:])
		return host, port
	}
	return v, 0
}

func sameServer(s FireMonitoringServer, host string, port int64) bool {
	return strings.EqualFold(strings.TrimSpace(s.Host), strings.TrimSpace(host)) && s.Port == port
}

func normalizeFireMonitoringSettings(settings FireMonitoringSettings) FireMonitoringSettings {
	n := FireMonitoringSettings{
		Enabled:       settings.Enabled,
		ObjectID:      strings.TrimSpace(settings.ObjectID),
		AckWaitSec:    settings.AckWaitSec,
		UseStdDateFmt: settings.UseStdDateFmt,
		Servers:       make([]FireMonitoringServer, 0, len(settings.Servers)),
	}

	if n.AckWaitSec <= 0 {
		n.AckWaitSec = 5
	}
	if n.AckWaitSec > 3600 {
		n.AckWaitSec = 3600
	}

	for _, s := range settings.Servers {
		host := strings.TrimSpace(s.Host)
		info := strings.TrimSpace(s.Info)
		port := s.Port
		if port < 0 {
			port = 0
		}
		if port > 65535 {
			port = 65535
		}
		if host == "" && port == 0 && info == "" {
			continue
		}
		n.Servers = append(n.Servers, FireMonitoringServer{
			Host:    host,
			Port:    port,
			Info:    info,
			Enabled: s.Enabled,
		})
		if len(n.Servers) >= 8 {
			break
		}
	}

	if len(n.Servers) == 0 {
		n.Servers = []FireMonitoringServer{
			{Enabled: true},
		}
	}

	return n
}

func (p *DBDataProvider) GetAdminAccessStatus() (AdminAccessStatus, error) {
	ctx, cancel := context.WithTimeout(context.Background(), adminQueryTimeout)
	defer cancel()

	status := AdminAccessStatus{}

	const qAdminCount = `
		SELECT COUNT(*)
		FROM PERSONAL
		WHERE COALESCE(ACCESS1, 0) = 1
	`
	if err := p.db.GetContext(ctx, &status.AdminUsersCount, p.db.Rebind(qAdminCount)); err != nil {
		return AdminAccessStatus{}, fmt.Errorf("failed to load admin users count: %w", err)
	}

	currentUser, err := resolveCurrentUserName(ctx, p.db)
	if err != nil {
		return AdminAccessStatus{}, err
	}
	status.CurrentUser = currentUser
	if strings.TrimSpace(currentUser) == "" {
		status.HasFullAccess = false
		status.MatchDescription = "active user is not resolved"
		return status, nil
	}

	type adminPersonalRow struct {
		Surname1 *string `db:"SURNAME1"`
		Name1    *string `db:"NAME1"`
		SecName1 *string `db:"SECNAME1"`
	}
	const qAdminUsers = `
		SELECT
			SURNAME1,
			NAME1,
			SECNAME1
		FROM PERSONAL
		WHERE COALESCE(ACCESS1, 0) = 1
		ORDER BY ID
	`
	var users []adminPersonalRow
	if err := p.db.SelectContext(ctx, &users, p.db.Rebind(qAdminUsers)); err != nil {
		return AdminAccessStatus{}, fmt.Errorf("failed to load admin users list: %w", err)
	}

	currNorm := normalizeComparableName(currentUser)
	for _, u := range users {
		surname := strings.TrimSpace(ptrToString(u.Surname1))
		name := strings.TrimSpace(ptrToString(u.Name1))
		secName := strings.TrimSpace(ptrToString(u.SecName1))
		full := strings.TrimSpace(strings.Join(compactStrings(surname, name, secName), " "))
		variants := []string{
			normalizeComparableName(surname),
			normalizeComparableName(name),
			normalizeComparableName(secName),
			normalizeComparableName(strings.Join(compactStrings(surname, name), " ")),
			normalizeComparableName(strings.Join(compactStrings(name, surname), " ")),
			normalizeComparableName(strings.Join(compactStrings(surname, name, secName), " ")),
			normalizeComparableName(strings.Join(compactStrings(name, secName, surname), " ")),
			normalizeComparableName(strings.Join(compactStrings(surname, secName), " ")),
		}
		for _, v := range variants {
			if v == "" {
				continue
			}
			if v == currNorm {
				status.HasFullAccess = true
				status.MatchedPersonal = full
				status.MatchDescription = "exact personal-name match"
				return status, nil
			}
		}
		fullNorm := normalizeComparableName(full)
		if fullNorm != "" && currNorm != "" && (strings.Contains(fullNorm, currNorm) || strings.Contains(currNorm, fullNorm)) {
			status.HasFullAccess = true
			status.MatchedPersonal = full
			status.MatchDescription = "fuzzy personal-name match"
			return status, nil
		}
	}

	status.HasFullAccess = false
	status.MatchDescription = "no PERSONAL.ACCESS1=1 match for active USERSTATUS user"
	return status, nil
}

func (p *DBDataProvider) RunDataIntegrityChecks(limit int) ([]AdminDataCheckIssue, error) {
	if limit <= 0 {
		limit = 200
	}
	if limit > 2000 {
		limit = 2000
	}

	ctx, cancel := context.WithTimeout(context.Background(), adminQueryTimeout)
	defer cancel()

	issues := make([]AdminDataCheckIssue, 0, limit)
	appendIssue := func(severity, code string, objn int64, details string) {
		if len(issues) >= limit {
			return
		}
		issues = append(issues, AdminDataCheckIssue{
			Severity: severity,
			Code:     code,
			ObjN:     objn,
			Details:  details,
		})
	}

	type dupObjNRow struct {
		ObjN *int64 `db:"OBJN"`
		Cnt  *int64 `db:"CNT"`
	}
	{
		const q = `
			SELECT FIRST 200
				OBJN,
				COUNT(*) AS CNT
			FROM OBJECTS_INFO
			WHERE OBJN IS NOT NULL
			  AND OBJN > 0
			GROUP BY OBJN
			HAVING COUNT(*) > 1
			ORDER BY OBJN
		`
		var rows []dupObjNRow
		if err := p.db.SelectContext(ctx, &rows, p.db.Rebind(q)); err != nil {
			return nil, fmt.Errorf("failed to run duplicate OBJN check: %w", err)
		}
		for _, r := range rows {
			appendIssue(
				"error",
				"DUP_OBJN",
				ptrToInt64(r.ObjN),
				fmt.Sprintf("в OBJECTS_INFO знайдено дублікати OBJN, кількість=%d", ptrToInt64(r.Cnt)),
			)
		}
	}

	type objOnlyRow struct {
		ObjN *int64 `db:"OBJN"`
	}
	{
		const q = `
			SELECT FIRST 400
				oi.OBJN
			FROM OBJECTS_INFO oi
			LEFT JOIN OBJECTS_STATE os ON os.OBJUIN = oi.OBJUIN
			WHERE oi.OBJN > 0
			  AND os.OBJUIN IS NULL
			ORDER BY oi.OBJN
		`
		var rows []objOnlyRow
		if err := p.db.SelectContext(ctx, &rows, p.db.Rebind(q)); err != nil {
			return nil, fmt.Errorf("failed to run OBJECTS_STATE check: %w", err)
		}
		for _, r := range rows {
			appendIssue("error", "NO_STATE", ptrToInt64(r.ObjN), "для об'єкта відсутній рядок в OBJECTS_STATE")
		}
	}
	{
		const q = `
			SELECT FIRST 400
				oi.OBJN
			FROM OBJECTS_INFO oi
			LEFT JOIN OBJECTS_LA la ON la.OBJUIN = oi.OBJUIN
			WHERE oi.OBJN > 0
			  AND la.OBJUIN IS NULL
			ORDER BY oi.OBJN
		`
		var rows []objOnlyRow
		if err := p.db.SelectContext(ctx, &rows, p.db.Rebind(q)); err != nil {
			return nil, fmt.Errorf("failed to run OBJECTS_LA check: %w", err)
		}
		for _, r := range rows {
			appendIssue("error", "NO_LA", ptrToInt64(r.ObjN), "для об'єкта відсутній рядок в OBJECTS_LA")
		}
	}
	{
		const q = `
			SELECT FIRST 400
				oi.OBJN
			FROM OBJECTS_INFO oi
			WHERE oi.OBJN > 0
			  AND CHAR_LENGTH(TRIM(COALESCE(oi.OBJSHORTNAME1, ''))) = 0
			ORDER BY oi.OBJN
		`
		var rows []objOnlyRow
		if err := p.db.SelectContext(ctx, &rows, p.db.Rebind(q)); err != nil {
			return nil, fmt.Errorf("failed to run empty short-name check: %w", err)
		}
		for _, r := range rows {
			appendIssue("warn", "EMPTY_SHORTNAME", ptrToInt64(r.ObjN), "порожня коротка назва об'єкта")
		}
	}

	type dupZoneRow struct {
		ObjN *int64 `db:"OBJN"`
		Zone *int64 `db:"ZONEN"`
		Cnt  *int64 `db:"CNT"`
	}
	{
		const q = `
			SELECT FIRST 400
				OBJN,
				ZONEN,
				COUNT(*) AS CNT
			FROM ZONES
			WHERE OBJN IS NOT NULL
			  AND ZONEN IS NOT NULL
			GROUP BY OBJN, ZONEN
			HAVING COUNT(*) > 1
			ORDER BY OBJN, ZONEN
		`
		var rows []dupZoneRow
		if err := p.db.SelectContext(ctx, &rows, p.db.Rebind(q)); err != nil {
			return nil, fmt.Errorf("failed to run duplicate ZONEN check: %w", err)
		}
		for _, r := range rows {
			appendIssue(
				"error",
				"DUP_ZONEN",
				ptrToInt64(r.ObjN),
				fmt.Sprintf("дубль зони ZONEN=%d, кількість=%d", ptrToInt64(r.Zone), ptrToInt64(r.Cnt)),
			)
		}
	}

	type gpsMismatchRow struct {
		ObjN  *int64 `db:"OBJN"`
		NavID *int64 `db:"NAV_ID"`
		GrpN  *int64 `db:"GRPN"`
	}
	{
		const q = `
			SELECT FIRST 400
				OBJN,
				NAV_ID,
				GRPN
			FROM OBJECTS_GPS
			WHERE COALESCE(NAV_ID, 0) <> 0
			   OR GRPN IS NOT NULL
			ORDER BY OBJN
		`
		var rows []gpsMismatchRow
		if err := p.db.SelectContext(ctx, &rows, p.db.Rebind(q)); err != nil {
			return nil, fmt.Errorf("failed to run OBJECTS_GPS NAV/GRPN check: %w", err)
		}
		for _, r := range rows {
			navID := ptrToInt64(r.NavID)
			grpN := ptrToInt64(r.GrpN)
			if navID != 0 {
				appendIssue(
					"warn",
					"GPS_NAV_NOT_ZERO",
					ptrToInt64(r.ObjN),
					fmt.Sprintf("в OBJECTS_GPS бажаний стан: NAV_ID=0 (зараз NAV_ID=%d, GRPN=%d)", navID, grpN),
				)
				continue
			}
			appendIssue(
				"info",
				"GPS_GRPN_LEGACY",
				ptrToInt64(r.ObjN),
				fmt.Sprintf("в OBJECTS_GPS знайдено легасі-стан: NAV_ID=0, GRPN=%d (допускається, але рекомендовано GRPN=NULL)", grpN),
			)
		}
	}

	return issues, nil
}

func (p *DBDataProvider) CollectObjectStatistics(filter AdminStatisticsFilter, limit int) ([]AdminStatisticsRow, error) {
	if limit <= 0 {
		limit = 5000
	}
	if limit > 50000 {
		limit = 50000
	}

	ctx, cancel := context.WithTimeout(context.Background(), adminQueryTimeout)
	defer cancel()

	type statsRow struct {
		ObjUin            *int64  `db:"OBJUIN"`
		ObjN              *int64  `db:"OBJN"`
		GrpN              *int64  `db:"GRPN"`
		ObjShortName      *string `db:"OBJSHORTNAME1"`
		ObjFullName       *string `db:"OBJFULLNAME1"`
		Address1          *string `db:"ADDRESS1"`
		Phones1           *string `db:"PHONES1"`
		Contract1         *string `db:"CONTRACT1"`
		ReservText        *string `db:"RESERVTEXT"`
		Location1         *string `db:"LOCATION1"`
		Notes1            *string `db:"NOTES1"`
		ObjChan           *int64  `db:"OBJCHAN"`
		PpkID             *int64  `db:"PPKID"`
		GsmPhone1         *string `db:"GSMPHONE"`
		GsmPhone2         *string `db:"GSMPHONE2"`
		GsmHiddenN        *int64  `db:"GSMHIDENINT"`
		SubServerA        *string `db:"SBSA"`
		SubServerB        *string `db:"SBSB"`
		TestControl1      *int64  `db:"TESTCONTROL1"`
		TestTime1         *int64  `db:"TESTTIME1"`
		GuardState1       *int64  `db:"GUARDSTATE1"`
		IsConnState1      *int64  `db:"ISCONNSTATE1"`
		AlarmState1       *int64  `db:"ALARMSTATE1"`
		TechAlarmState1   *int64  `db:"TECHALARMSTATE1"`
		ObjTypeID         *int64  `db:"OBJTYPEID"`
		ObjTypeName       *string `db:"OBJTYPE1"`
		ObjRegID          *int64  `db:"OBJREGID"`
		RegionName        *string `db:"RG_NAME"`
		Eng1              *int64  `db:"ENG1"`
		BlockedArmedOnOff *int16  `db:"BLOCKEDARMED_ON_OFF"`
	}

	baseQuery := fmt.Sprintf(`
		SELECT FIRST %d
			oi.OBJUIN,
			oi.OBJN,
			COALESCE(oi.GRPN, 1) AS GRPN,
			COALESCE(oi.OBJSHORTNAME1, '') AS OBJSHORTNAME1,
			COALESCE(oi.OBJFULLNAME1, '') AS OBJFULLNAME1,
			COALESCE(oi.ADDRESS1, '') AS ADDRESS1,
			COALESCE(oi.PHONES1, '') AS PHONES1,
			COALESCE(oi.CONTRACT1, '') AS CONTRACT1,
			COALESCE(oi.RESERVTEXT, '') AS RESERVTEXT,
			COALESCE(oi.LOCATION1, '') AS LOCATION1,
			COALESCE(oi.NOTES1, '') AS NOTES1,
			COALESCE(os.OBJCHAN, oi.OBJCHAN, 0) AS OBJCHAN,
			COALESCE(oi.PPKID, 0) AS PPKID,
			COALESCE(oi.GSMPHONE, '') AS GSMPHONE,
			COALESCE(oi.GSMPHONE2, '') AS GSMPHONE2,
			COALESCE(oi.GSMHIDENINT, 0) AS GSMHIDENINT,
			COALESCE(oi.SBSA, '') AS SBSA,
			COALESCE(oi.SBSB, '') AS SBSB,
			COALESCE(os.TESTCONTROL1, 0) AS TESTCONTROL1,
			COALESCE(os.TESTTIME1, 0) AS TESTTIME1,
			COALESCE(os.GUARDSTATE1, 0) AS GUARDSTATE1,
			COALESCE(os.ISCONNSTATE1, 0) AS ISCONNSTATE1,
			COALESCE(os.ALARMSTATE1, 0) AS ALARMSTATE1,
			COALESCE(os.TECHALARMSTATE1, 0) AS TECHALARMSTATE1,
			COALESCE(oi.OBJTYPEID, 0) AS OBJTYPEID,
			COALESCE(ot.OBJTYPE1, '') AS OBJTYPE1,
			COALESCE(oi.OBJREGID, 0) AS OBJREGID,
			COALESCE(rg.REG1, '') AS RG_NAME,
			COALESCE(oi.ENG1, 0) AS ENG1,
			COALESCE(os.BLOCKEDARMED_ON_OFF, 0) AS BLOCKEDARMED_ON_OFF
		FROM OBJECTS_INFO oi
		LEFT JOIN OBJECTS_STATE os ON os.OBJUIN = oi.OBJUIN
		LEFT JOIN OBJTYPES ot ON ot.ID = oi.OBJTYPEID
		LEFT JOIN OBJREGS rg ON rg.ID = oi.OBJREGID
		WHERE oi.OBJN > 0
	`, limit)

	conditions := make([]string, 0, 8)
	args := make([]interface{}, 0, 10)

	switch filter.ConnectionMode {
	case StatsConnectionOnline:
		conditions = append(conditions, "COALESCE(os.ISCONNSTATE1, 0) = 1")
	case StatsConnectionOffline:
		conditions = append(conditions, "COALESCE(os.ISCONNSTATE1, 0) = 0")
	}

	switch filter.ProtocolFilter {
	case StatsProtocolAutodial:
		conditions = append(conditions, "COALESCE(os.OBJCHAN, oi.OBJCHAN, 0) IN (1, 4)")
	case StatsProtocolMost:
		conditions = append(conditions, "COALESCE(os.OBJCHAN, oi.OBJCHAN, 0) IN (3, 5, 6, 7, 8, 9, 10, 11, 12, 13)")
	case StatsProtocolNova:
		conditions = append(conditions, "COALESCE(os.OBJCHAN, oi.OBJCHAN, 0) = 14")
	}

	if filter.ChannelCode != nil && *filter.ChannelCode >= 0 {
		conditions = append(conditions, "COALESCE(os.OBJCHAN, oi.OBJCHAN, 0) = ?")
		args = append(args, *filter.ChannelCode)
	}
	if filter.GuardState != nil && *filter.GuardState >= 0 {
		conditions = append(conditions, "COALESCE(os.GUARDSTATE1, 0) = ?")
		args = append(args, *filter.GuardState)
	}
	if filter.ObjTypeID != nil && *filter.ObjTypeID > 0 {
		conditions = append(conditions, "COALESCE(oi.OBJTYPEID, 0) = ?")
		args = append(args, *filter.ObjTypeID)
	}
	if filter.RegionID != nil && *filter.RegionID > 0 {
		conditions = append(conditions, "COALESCE(oi.OBJREGID, 0) = ?")
		args = append(args, *filter.RegionID)
	}
	if filter.BlockMode != nil {
		switch *filter.BlockMode {
		case DisplayBlockNone:
			conditions = append(conditions, "COALESCE(os.BLOCKEDARMED_ON_OFF, 0) = 0")
			conditions = append(conditions, "COALESCE(oi.ENG1, 0) = 0")
		case DisplayBlockTemporaryOff:
			conditions = append(conditions, "COALESCE(os.BLOCKEDARMED_ON_OFF, 0) = 1")
			conditions = append(conditions, "COALESCE(oi.ENG1, 0) = 0")
		case DisplayBlockDebug:
			conditions = append(conditions, "COALESCE(oi.ENG1, 0) <> 0")
		}
	}

	search := strings.TrimSpace(filter.Search)
	if search != "" {
		conditions = append(conditions, "(CAST(oi.OBJN AS VARCHAR(20)) CONTAINING ? OR COALESCE(oi.OBJSHORTNAME1, '') CONTAINING ?)")
		args = append(args, search, search)
	}

	query := baseQuery
	if len(conditions) > 0 {
		query += "\n  AND " + strings.Join(conditions, "\n  AND ")
	}
	query += "\nORDER BY oi.OBJN"

	rows := make([]statsRow, 0, limit)
	if err := p.db.SelectContext(ctx, &rows, p.db.Rebind(query), args...); err != nil {
		return nil, fmt.Errorf("failed to collect object statistics: %w", err)
	}

	items := make([]AdminStatisticsRow, 0, len(rows))
	for _, r := range rows {
		guard := ptrToInt64(r.GuardState1)
		eng1 := ptrToInt64(r.Eng1)
		blocked := int16(0)
		if r.BlockedArmedOnOff != nil {
			blocked = *r.BlockedArmedOnOff
		}
		mode := resolveDisplayBlockMode(&guard, &eng1, &blocked)
		trimString := func(v *string) string {
			if v == nil {
				return ""
			}
			return strings.TrimSpace(*v)
		}
		items = append(items, AdminStatisticsRow{
			ObjUIN:         ptrToInt64(r.ObjUin),
			ObjN:           ptrToInt64(r.ObjN),
			GrpN:           ptrToInt64(r.GrpN),
			ShortName:      trimString(r.ObjShortName),
			FullName:       trimString(r.ObjFullName),
			Address:        trimString(r.Address1),
			Phones:         trimString(r.Phones1),
			Contract:       trimString(r.Contract1),
			StartDate:      trimString(r.ReservText),
			Location:       trimString(r.Location1),
			Notes:          trimString(r.Notes1),
			ChannelCode:    ptrToInt64(r.ObjChan),
			PPKID:          ptrToInt64(r.PpkID),
			GSMPhone1:      trimString(r.GsmPhone1),
			GSMPhone2:      trimString(r.GsmPhone2),
			GSMHiddenN:     ptrToInt64(r.GsmHiddenN),
			SubServerA:     trimString(r.SubServerA),
			SubServerB:     trimString(r.SubServerB),
			TestControl:    ptrToInt64(r.TestControl1),
			TestTime:       ptrToInt64(r.TestTime1),
			GuardState:     guard,
			IsConnState:    ptrToInt64(r.IsConnState1),
			AlarmState:     ptrToInt64(r.AlarmState1),
			TechAlarmState: ptrToInt64(r.TechAlarmState1),
			ObjTypeID:      ptrToInt64(r.ObjTypeID),
			ObjTypeName:    trimString(r.ObjTypeName),
			RegionID:       ptrToInt64(r.ObjRegID),
			RegionName:     trimString(r.RegionName),
			BlockMode:      mode,
		})
	}

	return items, nil
}

func (p *DBDataProvider) EmulateEvent(objn int64, zone int64, messageUIN int64) error {
	if objn <= 0 {
		return fmt.Errorf("invalid object number")
	}
	if zone < 0 {
		return fmt.Errorf("invalid zone number")
	}
	if messageUIN <= 0 {
		return fmt.Errorf("invalid message UIN")
	}

	ctx, cancel := context.WithTimeout(context.Background(), adminQueryTimeout)
	defer cancel()

	tx, err := p.db.BeginTxx(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to begin emulation transaction: %w", err)
	}
	defer func() {
		_ = tx.Rollback()
	}()

	var objectRef objectRefRow
	const qObj = `
		SELECT FIRST 1 OBJUIN, GRPN
		FROM OBJECTS_INFO
		WHERE OBJN = ?
	`
	if err := tx.GetContext(ctx, &objectRef, p.db.Rebind(qObj), objn); err != nil {
		if err == sql.ErrNoRows {
			return fmt.Errorf("object #%d not found", objn)
		}
		return fmt.Errorf("failed to load object for emulation: %w", err)
	}

	group := int64(1)
	if objectRef.GrpN != nil && *objectRef.GrpN > 0 {
		group = *objectRef.GrpN
	}

	// За документацією перед емульованою подією повинно йти інформаційне
	// повідомлення "Емуляція події". Беремо відповідний код з MESSLIST, якщо є.
	var emulationInfoUIN int64
	const qInfo = `
		SELECT FIRST 1 UIN
		FROM MESSLIST
		WHERE UKR1 CONTAINING 'Емуляц'
		ORDER BY UIN
	`
	if err := tx.GetContext(ctx, &emulationInfoUIN, p.db.Rebind(qInfo)); err == nil {
		const qInsInfo = `
			INSERT INTO EVLOG (EVTIME1, OBJUIN, GRPN, ZONEN, EVUIN, INFO1)
			VALUES (CURRENT_TIMESTAMP, ?, ?, ?, ?, ?)
		`
		if _, err := tx.ExecContext(
			ctx,
			p.db.Rebind(qInsInfo),
			objectRef.ObjUin,
			group,
			zone,
			emulationInfoUIN,
			"Емуляція події",
		); err != nil {
			return fmt.Errorf("failed to insert emulation-info event: %w", err)
		}
	}

	const qInsEvent = `
		INSERT INTO EVLOG (EVTIME1, OBJUIN, GRPN, ZONEN, EVUIN, INFO1)
		VALUES (CURRENT_TIMESTAMP, ?, ?, ?, ?, ?)
	`
	if _, err := tx.ExecContext(
		ctx,
		p.db.Rebind(qInsEvent),
		objectRef.ObjUin,
		group,
		zone,
		messageUIN,
		"",
	); err != nil {
		return fmt.Errorf("failed to insert emulated event: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit emulation transaction: %w", err)
	}
	return nil
}

func resolveCurrentUserName(ctx context.Context, db *sqlx.DB) (string, error) {
	const qActive = `
		SELECT FIRST 1 TRIM(COALESCE(USERNAME, ''))
		FROM USERSTATUS
		WHERE COALESCE(USERSTATE, 0) <> 0
		ORDER BY USERNAME
	`
	var active sql.NullString
	if err := db.GetContext(ctx, &active, db.Rebind(qActive)); err == nil {
		return strings.TrimSpace(active.String), nil
	} else if err != sql.ErrNoRows {
		return "", fmt.Errorf("failed to load active USERSTATUS user: %w", err)
	}

	const qAny = `
		SELECT FIRST 1 TRIM(COALESCE(USERNAME, ''))
		FROM USERSTATUS
		WHERE CHAR_LENGTH(TRIM(COALESCE(USERNAME, ''))) > 0
		ORDER BY USERNAME
	`
	var any sql.NullString
	if err := db.GetContext(ctx, &any, db.Rebind(qAny)); err == nil {
		return strings.TrimSpace(any.String), nil
	} else if err != sql.ErrNoRows {
		return "", fmt.Errorf("failed to load USERSTATUS user fallback: %w", err)
	}
	return "", nil
}

func normalizeComparableName(v string) string {
	v = strings.TrimSpace(strings.ToUpper(v))
	if v == "" {
		return ""
	}
	return strings.Join(strings.Fields(v), " ")
}

func compactStrings(values ...string) []string {
	out := make([]string, 0, len(values))
	for _, v := range values {
		v = strings.TrimSpace(v)
		if v == "" {
			continue
		}
		out = append(out, v)
	}
	return out
}
