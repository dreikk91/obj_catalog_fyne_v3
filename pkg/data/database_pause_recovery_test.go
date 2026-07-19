package data

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"errors"
	"sync/atomic"
	"testing"
	"time"

	"github.com/jmoiron/sqlx"

	"obj_catalog_fyne_v3/pkg/models"
)

type pausedQueryDriver struct {
	opened  atomic.Int32
	started chan struct{}
	release chan struct{}
}

func (d *pausedQueryDriver) Open(string) (driver.Conn, error) {
	return &pausedQueryConn{
		id:      d.opened.Add(1),
		started: d.started,
		release: d.release,
	}, nil
}

type pausedQueryConn struct {
	id      int32
	started chan struct{}
	release chan struct{}
}

func (c *pausedQueryConn) Prepare(string) (driver.Stmt, error) {
	return nil, errors.New("not implemented")
}

func (c *pausedQueryConn) Close() error {
	return nil
}

func (c *pausedQueryConn) Begin() (driver.Tx, error) {
	return nil, errors.New("not implemented")
}

func (c *pausedQueryConn) QueryContext(context.Context, string, []driver.NamedValue) (driver.Rows, error) {
	if c.id == 1 {
		select {
		case c.started <- struct{}{}:
		default:
		}
		<-c.release
		return nil, errors.New("paused query released")
	}
	return nil, errors.New("replacement connection reached")
}

func TestDBDataProviderTriggerReconnectDetachesPausedEventQuery(t *testing.T) {
	db, paused := newPausedQueryDB(t, "bridge-pause-recovery-test")
	provider := NewDBDataProvider(db, "")

	assertEventRefreshRecovers(t, paused, provider.GetEventsContext, provider.TriggerReconnect)
}

func TestPhoenixDataProviderTriggerReconnectDetachesPausedEventQuery(t *testing.T) {
	db, paused := newPausedQueryDB(t, "phoenix-pause-recovery-test")
	provider := NewPhoenixDataProvider(db, "")

	assertEventRefreshRecovers(t, paused, provider.GetEventsContext, provider.TriggerReconnect)
}

func newPausedQueryDB(t *testing.T, driverName string) (*sqlx.DB, *pausedQueryDriver) {
	t.Helper()

	paused := &pausedQueryDriver{
		started: make(chan struct{}, 1),
		release: make(chan struct{}),
	}
	sql.Register(driverName, paused)
	rawDB, err := sql.Open(driverName, "")
	if err != nil {
		t.Fatalf("sql.Open() error = %v", err)
	}
	rawDB.SetMaxOpenConns(2)
	t.Cleanup(func() {
		_ = rawDB.Close()
	})
	return sqlx.NewDb(rawDB, driverName), paused
}

func assertEventRefreshRecovers(
	t *testing.T,
	paused *pausedQueryDriver,
	refresh func(context.Context) []models.Event,
	recover func(string),
) {
	t.Helper()

	firstDone := make(chan struct{})
	go func() {
		defer close(firstDone)
		refresh(context.Background())
	}()

	select {
	case <-paused.started:
	case <-time.After(time.Second):
		t.Fatal("first event query did not start")
	}

	recover("test virtual machine pause")

	secondDone := make(chan struct{})
	go func() {
		defer close(secondDone)
		refresh(context.Background())
	}()

	select {
	case <-secondDone:
	case <-time.After(250 * time.Millisecond):
		t.Fatal("replacement event query remained blocked behind the paused query")
	}

	close(paused.release)
	select {
	case <-firstDone:
	case <-time.After(time.Second):
		t.Fatal("paused query did not stop after test release")
	}
}
