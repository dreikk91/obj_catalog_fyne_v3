package data

import (
	"context"
	"database/sql"
	"fmt"
	"strconv"
	"strings"

	"github.com/jmoiron/sqlx"
	"github.com/rs/zerolog/log"

	"obj_catalog_fyne_v3/pkg/utils"
)

func (p *DBDataProvider) ListMessageProtocols() ([]int64, error) {
	ctx, cancel := context.WithTimeout(context.Background(), adminQueryTimeout)
	defer cancel()

	const q = `
		SELECT DISTINCT PROTID
		FROM MESSLIST
		WHERE PROTID IN (18, 3, 4)
		  AND OBJN = 0
		  AND SC1 < 12
		ORDER BY
			CASE PROTID
				WHEN 18 THEN 1
				WHEN 3 THEN 2
				WHEN 4 THEN 3
				ELSE 99
			END,
			PROTID
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
		WHERE PROTID IN (18, 3, 4)
		  AND OBJN = 0
		  AND SC1 < 12
	`
	args := make([]any, 0, 6)

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

func (p *DBDataProvider) List220VMessageBuckets(protocolIDs []int64, filter string) (Admin220VMessageBuckets, error) {
	result := Admin220VMessageBuckets{
		Free:    []AdminMessage{},
		Alarm:   []AdminMessage{},
		Restore: []AdminMessage{},
	}

	if len(protocolIDs) == 0 {
		return result, nil
	}

	ctx, cancel := context.WithTimeout(context.Background(), adminQueryTimeout)
	defer cancel()

	filter = strings.TrimSpace(filter)

	baseQuery := `
		SELECT UIN, PROTID, MESSID, MESSIDHEX, UKR1, SC1, FORADMINONLY, MC220V
		FROM MESSLIST
		WHERE PROTID IN (?)
		  AND OBJN = 0
		  AND SC1 < 12
	`
	args := []any{protocolIDs}

	if filter != "" {
		baseQuery += `
		  AND (
			UKR1 CONTAINING ?
			OR MESSIDHEX CONTAINING ?
			OR CAST(MESSID AS VARCHAR(20)) CONTAINING ?
		  )
		`
		args = append(args, filter, filter, filter)
	}

	baseQuery += ` ORDER BY COALESCE(MESSID, UIN), UIN`

	q, inArgs, err := sqlx.In(baseQuery, args...)
	if err != nil {
		return result, fmt.Errorf("failed to build 220v messages query: %w", err)
	}
	q = p.db.Rebind(q)

	var rows []message220Row
	if err := p.db.SelectContext(ctx, &rows, q, inArgs...); err != nil {
		return result, fmt.Errorf("failed to list 220v messages: %w", err)
	}

	for _, r := range rows {
		msg := AdminMessage{
			UIN:          r.UIN,
			ProtocolID:   r.ProtID,
			MessageID:    r.MessID,
			MessageHex:   ptrToString(r.MessIDHex),
			Text:         ptrToString(r.Ukr1),
			SC1:          r.Sc1,
			ForAdminOnly: r.ForAdminOnly != nil && *r.ForAdminOnly != 0,
		}

		mode := int16(0)
		if r.Mc220v != nil {
			mode = *r.Mc220v
		}

		switch mode {
		case int16(Admin220VAlarm):
			result.Alarm = append(result.Alarm, msg)
		case int16(Admin220VRestore):
			result.Restore = append(result.Restore, msg)
		default:
			result.Free = append(result.Free, msg)
		}
	}

	return result, nil
}

func (p *DBDataProvider) SetMessage220VMode(uin int64, mode Admin220VMode) error {
	if uin <= 0 {
		return fmt.Errorf("invalid message uin")
	}

	if mode != Admin220VNone && mode != Admin220VAlarm && mode != Admin220VRestore {
		return fmt.Errorf("invalid 220v mode")
	}

	ctx, cancel := context.WithTimeout(context.Background(), adminQueryTimeout)
	defer cancel()

	const q = `UPDATE MESSLIST SET MC220V = ? WHERE UIN = ?`
	if _, err := p.db.ExecContext(ctx, p.db.Rebind(q), int16(mode), uin); err != nil {
		return fmt.Errorf("failed to update MC220V mode: %w", err)
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

	args := make([]any, 0, 3)
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
		full := utils.JoinTrimmedNonEmpty(surname, name, secName)
		variants := []string{
			normalizeComparableName(surname),
			normalizeComparableName(name),
			normalizeComparableName(secName),
			normalizeComparableName(utils.JoinTrimmedNonEmpty(surname, name)),
			normalizeComparableName(utils.JoinTrimmedNonEmpty(name, surname)),
			normalizeComparableName(utils.JoinTrimmedNonEmpty(surname, name, secName)),
			normalizeComparableName(utils.JoinTrimmedNonEmpty(name, secName, surname)),
			normalizeComparableName(utils.JoinTrimmedNonEmpty(surname, secName)),
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
			WHERE oi.OBJN > 36
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
			WHERE oi.OBJN > 36
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
			WHERE oi.OBJN > 36
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
		PpkName           *string `db:"PPK_NAME"`
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
			COALESCE(TRIM(ppk.PANELMARK1), '') AS PPK_NAME,
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
		LEFT JOIN PPK ppk ON ppk.ID = oi.PPKID - 100
		WHERE oi.OBJN > 36
	`, limit)

	conditions := make([]string, 0, 8)
	args := make([]any, 0, 10)

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
			PPKName:        trimString(r.PpkName),
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
