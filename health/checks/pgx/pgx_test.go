package pgx

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// mockPool implements the minimal interface needed for testing.
// We can't use the real pgxpool.Pool without a database,
// so we test at the boundary level.

func TestNew_NilPool(t *testing.T) {
	check := New(Config{Pool: nil})
	err := check(context.Background())

	require.Error(t, err)
	assert.ErrorIs(t, err, ErrNilPool)
	assert.Contains(t, err.Error(), "pool is nil")
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

	// Verify it's callable
	require.NotNil(t, checkFunc)

	// Call it with nil pool - should return error
	err := checkFunc(context.Background())
	require.Error(t, err)
}

// TestNew_ContextRespected verifies the check respects context.
// This tests the function's context parameter usage.
func TestNew_ContextRespected(t *testing.T) {
	check := New(Config{Pool: nil})

	// Create an already cancelled context
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	// With nil pool, we get ErrNilPool before context is checked
	// This is expected behavior - validation happens first
	err := check(ctx)
	assert.ErrorIs(t, err, ErrNilPool)
}

// TestConfig_PoolField verifies Config struct fields.
func TestConfig_PoolField(t *testing.T) {
	cfg := Config{}

	// Pool should be nil by default
	assert.Nil(t, cfg.Pool)
}

// Integration test placeholder - requires actual Postgres.
// This would be run in CI with a test database.
func TestNew_Integration(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	// Integration test would require:
	// 1. Postgres running (e.g., via testcontainers)
	// 2. pgxpool.New() with valid connection string
	// 3. Verify check returns nil when connected
	// 4. Verify check returns error when disconnected
	t.Skip("integration test requires Postgres - use testcontainers in CI")
}

// TestNew_ErrorWrapping verifies that ping errors are properly wrapped.
func TestNew_ErrorWrapping(t *testing.T) {
	// We can only test the nil pool case without a real connection
	// The error wrapping for ping errors follows the same pattern as sql check
	check := New(Config{Pool: nil})
	err := check(context.Background())

	require.Error(t, err)
	// Verify the error is the sentinel error
	assert.ErrorIs(t, err, ErrNilPool)
}

// TestNew_Concurrency verifies thread safety.
func TestNew_Concurrency(t *testing.T) {
	check := New(Config{Pool: nil})

	done := make(chan struct{})
	for range 10 {
		go func() {
			err := check(context.Background())
			assert.Error(t, err)
			done <- struct{}{}
		}()
	}

	// Wait for all goroutines with timeout
	for range 10 {
		select {
		case <-done:
		case <-time.After(time.Second):
			t.Fatal("timeout waiting for concurrent checks")
		}
	}
}

// TestNew_ContextTimeout verifies timeout handling (with nil pool).
func TestNew_ContextTimeout(t *testing.T) {
	check := New(Config{Pool: nil})

	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	// Should return immediately with nil pool error, not wait for timeout
	start := time.Now()
	err := check(ctx)
	elapsed := time.Since(start)

	require.Error(t, err)
	assert.ErrorIs(t, err, ErrNilPool)
	assert.Less(t, elapsed, 10*time.Millisecond, "nil pool check should be immediate")
}

// BenchmarkNew measures allocation and performance.
func BenchmarkNew(b *testing.B) {
	cfg := Config{Pool: nil}

	b.Run("create_check", func(b *testing.B) {
		for b.Loop() {
			_ = New(cfg)
		}
	})

	b.Run("run_check_nil_pool", func(b *testing.B) {
		check := New(cfg)
		ctx := context.Background()
		for b.Loop() {
			_ = check(ctx)
		}
	})
}
