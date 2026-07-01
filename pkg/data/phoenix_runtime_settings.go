package data

import (
	"context"
	"crypto/md5"
	"crypto/subtle"
	"encoding/hex"
	"fmt"
	"sort"
	"strings"

	"github.com/jmoiron/sqlx"

	"obj_catalog_fyne_v3/pkg/config"
	"obj_catalog_fyne_v3/pkg/database"
)

// PhoenixOperator is an existing Phoenix operator available for alarm actions.
type PhoenixOperator struct {
	ID    int64
	Name  string
	Login string
}

// DisplayName returns a stable user-facing operator label.
func (o PhoenixOperator) DisplayName() string {
	name := strings.TrimSpace(o.Name)
	login := strings.TrimSpace(o.Login)
	switch {
	case name != "" && login != "" && !strings.EqualFold(name, login):
		return name + " (" + login + ")"
	case name != "":
		return name
	case login != "":
		return login
	default:
		return fmt.Sprintf("Користувач %d", o.ID)
	}
}

// PhoenixRuntimeMetadata contains Control Center ports and selectable operators.
type PhoenixRuntimeMetadata struct {
	ControlPort int
	ClientPort  int
	AdminPort   int
	GPSPort     int
	Operators   []PhoenixOperator
}

// ConfigureRuntime loads database-owned ports and validates the configured operator.
func (p *PhoenixDataProvider) ConfigureRuntime(ctx context.Context, cfg config.DBConfig) error {
	if p == nil || p.db == nil {
		return fmt.Errorf("phoenix runtime: база не ініціалізована")
	}
	metadata, err := QueryPhoenixRuntimeMetadata(ctx, p.db)
	if err != nil {
		return err
	}

	operatorID := cfg.PhoenixOperatorID
	operatorName := strings.TrimSpace(cfg.PhoenixOperatorName)
	if operatorID > 0 && cfg.PhoenixOperatorPassword != "" {
		if err := ValidatePhoenixOperatorPassword(
			ctx,
			p.db,
			operatorID,
			cfg.PhoenixOperatorPassword,
		); err != nil {
			return err
		}
		for _, operator := range metadata.Operators {
			if operator.ID == operatorID {
				operatorName = strings.TrimSpace(operator.Login)
				if operatorName == "" {
					operatorName = strings.TrimSpace(operator.Name)
				}
				break
			}
		}
	} else {
		operatorID = 0
		operatorName = ""
	}
	p.ConfigureAlarmOperator(
		operatorID,
		operatorName,
		cfg.PhoenixControlHost,
		metadata,
		config.NormalizePhoenixClientRole(cfg.PhoenixClientRole),
	)
	return nil
}

// StartControlCenterSession starts the configured Duty Operator UDP session.
func (p *PhoenixDataProvider) StartControlCenterSession() error {
	if p == nil {
		return fmt.Errorf("phoenix UDP: провайдер не ініціалізований")
	}
	p.operatorMu.RLock()
	host := strings.TrimSpace(p.controlCenterHost)
	p.operatorMu.RUnlock()
	if host == "" {
		return nil
	}
	if err := p.startControlCenterSession(); err != nil {
		return err
	}
	go p.announceControlCenterOperator()
	return nil
}

// LoadPhoenixRuntimeMetadata opens the configured Phoenix database and reads
// ports and operators used by the desktop clients.
func LoadPhoenixRuntimeMetadata(ctx context.Context, cfg config.DBConfig) (PhoenixRuntimeMetadata, error) {
	db, err := database.InitNamedDB("sqlserver", cfg.PhoenixDSN(), "Phoenix settings")
	if err != nil {
		return PhoenixRuntimeMetadata{}, err
	}
	defer db.Close()
	if err := db.PingContext(ctx); err != nil {
		return PhoenixRuntimeMetadata{}, fmt.Errorf("phoenix settings ping: %w", err)
	}
	return QueryPhoenixRuntimeMetadata(ctx, db)
}

// QueryPhoenixRuntimeMetadata reads Control Center ports and operators.
func QueryPhoenixRuntimeMetadata(ctx context.Context, db *sqlx.DB) (PhoenixRuntimeMetadata, error) {
	if db == nil {
		return PhoenixRuntimeMetadata{}, fmt.Errorf("phoenix metadata: база не ініціалізована")
	}

	var metadata PhoenixRuntimeMetadata
	var portRows []struct {
		Field string `db:"field"`
		Value int    `db:"value"`
	}
	if err := db.SelectContext(ctx, &portRows, `
SELECT
	LTRIM(RTRIM(ISNULL(Field, ''))) AS field,
	ISNULL(Value, 0) AS value
FROM PortSettings WITH (NOLOCK)`); err != nil {
		return PhoenixRuntimeMetadata{}, fmt.Errorf("phoenix port settings: %w", err)
	}
	for _, row := range portRows {
		switch strings.ToLower(strings.TrimSpace(row.Field)) {
		case "control center udp port":
			metadata.ControlPort = row.Value
		case "duty operator udp port":
			metadata.ClientPort = row.Value
		case "administrator udp port":
			metadata.AdminPort = row.Value
		case "gps udp port":
			metadata.GPSPort = row.Value
		}
	}

	operators, err := queryPhoenixOperators(ctx, db)
	if err != nil {
		return PhoenixRuntimeMetadata{}, err
	}
	metadata.Operators = operators
	return metadata, nil
}

func queryPhoenixOperators(ctx context.Context, db *sqlx.DB) ([]PhoenixOperator, error) {
	queries := []string{
		`SELECT ISNULL(Person_Id, 0) AS person_id, ISNULL(person_name, '') AS person_name,
		        ISNULL(Person_Login, '') AS person_login
		   FROM vwPersonal WITH (NOLOCK)`,
		`SELECT ISNULL(Person_Id, 0) AS person_id, ISNULL(Person_Name, '') AS person_name,
		        ISNULL(Person_Login, '') AS person_login
		   FROM Personal WITH (NOLOCK)`,
	}

	var lastErr error
	for _, query := range queries {
		var rows []struct {
			ID    int64  `db:"person_id"`
			Name  string `db:"person_name"`
			Login string `db:"person_login"`
		}
		if err := db.SelectContext(ctx, &rows, query); err != nil {
			lastErr = err
			continue
		}
		operators := make([]PhoenixOperator, 0, len(rows))
		for _, row := range rows {
			if row.ID <= 0 {
				continue
			}
			name := strings.TrimSpace(row.Name)
			if separator := strings.Index(name, "|"); separator >= 0 {
				name = strings.TrimSpace(name[:separator])
			}
			operators = append(operators, PhoenixOperator{
				ID:    row.ID,
				Name:  name,
				Login: strings.TrimSpace(row.Login),
			})
		}
		sort.SliceStable(operators, func(i, j int) bool {
			return strings.ToLower(operators[i].DisplayName()) < strings.ToLower(operators[j].DisplayName())
		})
		return operators, nil
	}
	return nil, fmt.Errorf("phoenix operators query: %w", lastErr)
}

// ValidatePhoenixOperatorCredentials verifies the selected Phoenix user's password.
func ValidatePhoenixOperatorCredentials(ctx context.Context, cfg config.DBConfig) error {
	if cfg.PhoenixOperatorID <= 0 {
		return fmt.Errorf("не вибрано користувача Phoenix")
	}
	if cfg.PhoenixOperatorPassword == "" {
		return fmt.Errorf("не введено пароль користувача Phoenix")
	}

	db, err := database.InitNamedDB("sqlserver", cfg.PhoenixDSN(), "Phoenix operator auth")
	if err != nil {
		return err
	}
	defer db.Close()
	if err := db.PingContext(ctx); err != nil {
		return fmt.Errorf("phoenix operator auth ping: %w", err)
	}
	return ValidatePhoenixOperatorPassword(ctx, db, cfg.PhoenixOperatorID, cfg.PhoenixOperatorPassword)
}

// ValidatePhoenixOperatorPassword verifies one operator against the connected Phoenix database.
func ValidatePhoenixOperatorPassword(ctx context.Context, db *sqlx.DB, operatorID int64, password string) error {
	var columns []struct {
		Table  string `db:"table_name"`
		Column string `db:"column_name"`
	}
	if err := db.SelectContext(ctx, &columns, `
SELECT TABLE_NAME AS table_name, COLUMN_NAME AS column_name
FROM INFORMATION_SCHEMA.COLUMNS
WHERE LOWER(TABLE_NAME) IN ('personal', 'vwpersonal')`); err != nil {
		return fmt.Errorf("phoenix operator password schema: %w", err)
	}

	passwordFieldFound := false
	for _, column := range columns {
		name := strings.ToLower(strings.TrimSpace(column.Column))
		if !strings.Contains(name, "pass") &&
			!strings.Contains(name, "pwd") &&
			!strings.Contains(name, "psw") {
			continue
		}
		passwordFieldFound = true
		table := quoteSQLServerIdentifier(column.Table)
		passwordColumn := quoteSQLServerIdentifier(column.Column)
		query := fmt.Sprintf(
			"SELECT TOP (1) CONVERT(varchar(255), %s) FROM %s WITH (NOLOCK) WHERE Person_Id = @p1",
			passwordColumn,
			table,
		)
		var storedPassword string
		if err := db.GetContext(ctx, &storedPassword, query, operatorID); err != nil {
			continue
		}
		if phoenixPasswordMatches(storedPassword, password) {
			return nil
		}
	}
	if passwordFieldFound {
		return fmt.Errorf("невірний пароль користувача Phoenix")
	}
	return fmt.Errorf("Phoenix БД не надає поле для перевірки пароля користувача")
}

func phoenixPasswordMatches(stored, password string) bool {
	stored = strings.ToUpper(strings.TrimSpace(stored))
	plain := strings.ToUpper(strings.TrimSpace(password))
	sum := md5.Sum([]byte(password))
	hashed := strings.ToUpper(hex.EncodeToString(sum[:]))
	return subtle.ConstantTimeCompare([]byte(stored), []byte(plain)) == 1 ||
		subtle.ConstantTimeCompare([]byte(stored), []byte(hashed)) == 1
}

func quoteSQLServerIdentifier(value string) string {
	return "[" + strings.ReplaceAll(strings.TrimSpace(value), "]", "]]") + "]"
}
