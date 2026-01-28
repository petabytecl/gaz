package worker

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestDefaultWorkerOptions_ReturnsSensibleDefaults(t *testing.T) {
	opts := DefaultWorkerOptions()

	assert.Equal(t, 1, opts.PoolSize)
	assert.False(t, opts.Critical)
	assert.Equal(t, 30*time.Second, opts.StableRunPeriod)
	assert.Equal(t, 5, opts.MaxRestarts)
	assert.Equal(t, 10*time.Minute, opts.CircuitWindow)
}

func TestWithPoolSize_SetsPoolSize(t *testing.T) {
	opts := DefaultWorkerOptions()
	opts.ApplyOptions(WithPoolSize(4))

	assert.Equal(t, 4, opts.PoolSize)
}

func TestWithPoolSize_IgnoresZero(t *testing.T) {
	opts := DefaultWorkerOptions()
	opts.ApplyOptions(WithPoolSize(0))

	assert.Equal(t, 1, opts.PoolSize) // Default unchanged
}

func TestWithPoolSize_IgnoresNegative(t *testing.T) {
	opts := DefaultWorkerOptions()
	opts.ApplyOptions(WithPoolSize(-1))

	assert.Equal(t, 1, opts.PoolSize) // Default unchanged
}

func TestWithCritical_SetsCriticalTrue(t *testing.T) {
	opts := DefaultWorkerOptions()
	opts.ApplyOptions(WithCritical())

	assert.True(t, opts.Critical)
}

func TestWithStableRunPeriod_SetsPeriod(t *testing.T) {
	opts := DefaultWorkerOptions()
	opts.ApplyOptions(WithStableRunPeriod(time.Minute))

	assert.Equal(t, time.Minute, opts.StableRunPeriod)
}

func TestWithStableRunPeriod_IgnoresZero(t *testing.T) {
	opts := DefaultWorkerOptions()
	opts.ApplyOptions(WithStableRunPeriod(0))

	assert.Equal(t, 30*time.Second, opts.StableRunPeriod) // Default unchanged
}

func TestWithMaxRestarts_SetsMaxRestarts(t *testing.T) {
	opts := DefaultWorkerOptions()
	opts.ApplyOptions(WithMaxRestarts(3))

	assert.Equal(t, 3, opts.MaxRestarts)
}

func TestWithMaxRestarts_IgnoresZero(t *testing.T) {
	opts := DefaultWorkerOptions()
	opts.ApplyOptions(WithMaxRestarts(0))

	assert.Equal(t, 5, opts.MaxRestarts) // Default unchanged
}

func TestWithCircuitWindow_SetsWindow(t *testing.T) {
	opts := DefaultWorkerOptions()
	opts.ApplyOptions(WithCircuitWindow(5 * time.Minute))

	assert.Equal(t, 5*time.Minute, opts.CircuitWindow)
}

func TestWithCircuitWindow_IgnoresZero(t *testing.T) {
	opts := DefaultWorkerOptions()
	opts.ApplyOptions(WithCircuitWindow(0))

	assert.Equal(t, 10*time.Minute, opts.CircuitWindow) // Default unchanged
}

func TestApplyOptions_ChainsMultipleOptions(t *testing.T) {
	opts := DefaultWorkerOptions()
	opts.ApplyOptions(
		WithPoolSize(8),
		WithCritical(),
		WithMaxRestarts(10),
		WithStableRunPeriod(time.Minute),
		WithCircuitWindow(15*time.Minute),
	)

	assert.Equal(t, 8, opts.PoolSize)
	assert.True(t, opts.Critical)
	assert.Equal(t, 10, opts.MaxRestarts)
	assert.Equal(t, time.Minute, opts.StableRunPeriod)
	assert.Equal(t, 15*time.Minute, opts.CircuitWindow)
}
