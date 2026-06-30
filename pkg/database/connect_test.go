package database

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"errors"
	"testing"
	"time"

	"github.com/jmoiron/sqlx"
)

type healthTestDriver struct {
	pingErr error
}

func (d healthTestDriver) Open(string) (driver.Conn, error) {
	return healthTestConn{pingErr: d.pingErr}, nil
}

type healthTestConn struct {
	pingErr error
}

func (c healthTestConn) Prepare(string) (driver.Stmt, error) {
	return nil, errors.New("not implemented")
}

func (healthTestConn) Close() error {
	return nil
}

func (healthTestConn) Begin() (driver.Tx, error) {
	return nil, errors.New("not implemented")
}

func (c healthTestConn) Ping(_ context.Context) error {
	return c.pingErr
}

func TestInitNamedDB_ReturnsErrorForInvalidDriver(t *testing.T) {
	db, err := InitNamedDB("missing-driver-for-test", "", "Broken")
	if err == nil {
		t.Fatal("expected error for invalid driver")
	}
	if db != nil {
		t.Fatal("expected nil db for invalid driver")
	}
}

func TestStartNamedHealthCheckWithStatusRecordsInitialPing(t *testing.T) {
	const driverName = "connection-health-offline-test"
	sql.Register(driverName, healthTestDriver{pingErr: errors.New("network unavailable")})
	db, err := sql.Open(driverName, "")
	if err != nil {
		t.Fatalf("sql.Open() error = %v", err)
	}
	defer db.Close()

	cancel, health := StartNamedHealthCheckWithStatus(sqlx.NewDb(db, driverName), "test")
	defer cancel()

	deadline := time.Now().Add(time.Second)
	for {
		checked, online := health.Status()
		if checked {
			if online {
				t.Fatal("health status is online, want offline")
			}
			return
		}
		if time.Now().After(deadline) {
			t.Fatal("initial health check did not complete")
		}
		time.Sleep(time.Millisecond)
	}
}
