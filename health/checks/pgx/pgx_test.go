package pgx

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// mockPinger implements the Pinger interface for testing.
type mockPinger struct {
	pingErr   error
	pingDelay time.Duration
}

func (m *mockPinger) Ping(ctx context.Context) error {
	if m.pingDelay > 0 {
		select {
		case <-ctx.Done():
			return fmt.Errorf("ping cancelled: %w", context.Cause(ctx))
		case <-time.After(m.pingDelay):
		}
	}
	return m.pingErr
}

func TestNew_NilPool(t *testing.T) {
	check := New(Config{Pool: nil})
	err := check(context.Background())

	require.Error(t, err)
	assert.ErrorIs(t, err, ErrNilPool)
	assert.Contains(t, err.Error(), "pool is nil")
}

func TestNew_SuccessfulPing(t *testing.T) {
	mock := &mockPinger{pingErr: nil}
	check := New(Config{Pool: mock})

	err := check(context.Background())

	assert.NoError(t, err)
}

func TestNew_PingFailure(t *testing.T) {
	mock := &mockPinger{pingErr: errors.New("connection refused")}
	check := New(Config{Pool: mock})

	err := check(context.Background())

	require.Error(t, err)
	assert.Contains(t, err.Error(), "pgx: ping failed")
	assert.Contains(t, err.Error(), "connection refused")
}

func TestNew_ContextCancellation(t *testing.T) {
	mock := &mockPinger{pingDelay: 5 * time.Second}
	check := New(Config{Pool: mock})

	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	start := time.Now()
	err := check(ctx)
	elapsed := time.Since(start)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "ping failed")
	// Ensure it returned quickly due to context cancellation
	assert.Less(t, elapsed, time.Second, "should cancel quickly")
}

func TestNew_ContextDeadlineRespected(t *testing.T) {
	mock := &mockPinger{pingDelay: 10 * time.Millisecond}
	check := New(Config{Pool: mock})

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	err := check(ctx)
	assert.NoError(t, err)
}

// Verify return type matches health.CheckFunc signature.
var _ func(context.Context) error = New(Config{})

// TestErrNilPool_ErrorMessage verifies the error message format.
func TestErrNilPool_ErrorMessage(t *testing.T) {
	assert.Equal(t, "pgx: pool is nil", ErrNilPool.Error())
}

// TestNew_ReturnsCheckFunc verifies the function signature.
func TestNew_ReturnsCheckFunc(t *testing.T) {
	cfg := Config{}
	checkFunc := New(cfg)

	require.NotNil(t, checkFunc)

	err := checkFunc(context.Background())
	require.Error(t, err)
}

// TestNew_Concurrency verifies thread safety.
func TestNew_Concurrency(t *testing.T) {
	mock := &mockPinger{pingErr: nil}
	check := New(Config{Pool: mock})

	done := make(chan struct{})
	for range 10 {
		go func() {
			err := check(context.Background())
			assert.NoError(t, err)
			done <- struct{}{}
		}()
	}

	for range 10 {
		select {
		case <-done:
		case <-time.After(time.Second):
			t.Fatal("timeout waiting for concurrent checks")
		}
	}
}

// BenchmarkNew measures allocation and performance.
func BenchmarkNew(b *testing.B) {
	mock := &mockPinger{pingErr: nil}
	cfg := Config{Pool: mock}

	b.Run("create_check", func(b *testing.B) {
		for b.Loop() {
			_ = New(cfg)
		}
	})

	b.Run("run_check", func(b *testing.B) {
		check := New(cfg)
		ctx := context.Background()
		for b.Loop() {
			_ = check(ctx)
		}
	})
}
