package sql

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"errors"
	"fmt"
	"io"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// testDriver implements a minimal database/sql/driver for testing.
type testDriver struct {
	mu        sync.Mutex
	pingErr   error
	pingDelay time.Duration
}

func (d *testDriver) Open(name string) (driver.Conn, error) {
	return &testConn{driver: d}, nil
}

func (d *testDriver) setPingError(err error) {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.pingErr = err
}

func (d *testDriver) setPingDelay(delay time.Duration) {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.pingDelay = delay
}

func (d *testDriver) getPingBehavior() (time.Duration, error) {
	d.mu.Lock()
	defer d.mu.Unlock()
	return d.pingDelay, d.pingErr
}

// testConn implements driver.Conn and driver.Pinger.
type testConn struct {
	driver *testDriver
}

func (c *testConn) Prepare(query string) (driver.Stmt, error) {
	return nil, errors.New("not implemented")
}

func (c *testConn) Close() error {
	return nil
}

func (c *testConn) Begin() (driver.Tx, error) {
	return nil, errors.New("not implemented")
}

// Ping implements driver.Pinger for context-aware ping testing.
func (c *testConn) Ping(ctx context.Context) error {
	delay, pingErr := c.driver.getPingBehavior()

	if delay > 0 {
		select {
		case <-ctx.Done():
			return fmt.Errorf("context error: %w", ctx.Err())
		case <-time.After(delay):
		}
	}

	return pingErr
}

// testConnector implements driver.Connector for sql.OpenDB.
type testConnector struct {
	driver *testDriver
}

func (c *testConnector) Connect(ctx context.Context) (driver.Conn, error) {
	return &testConn{driver: c.driver}, nil
}

func (c *testConnector) Driver() driver.Driver {
	return c.driver
}

func newTestDB(d *testDriver) *sql.DB {
	return sql.OpenDB(&testConnector{driver: d})
}

func TestNew_NilDB(t *testing.T) {
	check := New(Config{DB: nil})
	err := check(context.Background())

	require.Error(t, err)
	assert.Contains(t, err.Error(), "database connection is nil")
}

func TestNew_SuccessfulPing(t *testing.T) {
	d := &testDriver{}
	db := newTestDB(d)
	defer db.Close()

	check := New(Config{DB: db})
	err := check(context.Background())

	assert.NoError(t, err)
}

func TestNew_PingFailure(t *testing.T) {
	d := &testDriver{pingErr: errors.New("connection refused")}
	db := newTestDB(d)
	defer db.Close()

	check := New(Config{DB: db})
	err := check(context.Background())

	require.Error(t, err)
	assert.Contains(t, err.Error(), "ping failed")
	assert.Contains(t, err.Error(), "connection refused")
}

func TestNew_ContextCancellation(t *testing.T) {
	d := &testDriver{pingDelay: 5 * time.Second}
	db := newTestDB(d)
	defer db.Close()

	check := New(Config{DB: db})

	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	start := time.Now()
	err := check(ctx)
	elapsed := time.Since(start)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "ping failed")
	// Ensure it returned quickly due to context cancellation, not the full 5s delay
	assert.Less(t, elapsed, time.Second, "should cancel quickly")
}

func TestNew_ContextDeadlineRespected(t *testing.T) {
	d := &testDriver{pingDelay: 10 * time.Millisecond}
	db := newTestDB(d)
	defer db.Close()

	check := New(Config{DB: db})

	// Context with enough time for the ping to complete
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	err := check(ctx)
	assert.NoError(t, err)
}

// Verify return type matches health.CheckFunc signature.
var _ func(context.Context) error = New(Config{})

// stmtRows is a minimal implementation to satisfy driver.Rows interface.
type stmtRows struct{}

func (r *stmtRows) Columns() []string { return nil }
func (r *stmtRows) Close() error      { return nil }
func (r *stmtRows) Next(dest []driver.Value) error {
	return io.EOF
}
