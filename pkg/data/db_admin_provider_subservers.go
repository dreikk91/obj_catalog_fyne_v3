package data

import (
	"context"
	"fmt"
	"strings"
)

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
		WHERE OBJN > 36
		
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
