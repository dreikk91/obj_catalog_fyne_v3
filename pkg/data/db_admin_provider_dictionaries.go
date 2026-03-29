package data

import (
	"context"
	"fmt"
	"strings"
)

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
