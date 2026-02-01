package cron

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestMockJob(t *testing.T) {
	m := NewMockJob("test-job")

	assert.Equal(t, "test-job", m.Name())
	assert.Equal(t, "@every 1m", m.Schedule())
	assert.Equal(t, 30*time.Second, m.Timeout())
	assert.NoError(t, m.Run(context.Background()))

	m.AssertCalled(t, "Name")
	m.AssertCalled(t, "Schedule")
	m.AssertCalled(t, "Timeout")
	m.AssertCalled(t, "Run", mock.Anything)
}

func TestMockJobCustomExpectations(t *testing.T) {
	m := &MockJob{}
	testErr := errors.New("job failed")

	m.On("Name").Return("failing-job")
	m.On("Schedule").Return("@hourly")
	m.On("Timeout").Return(5 * time.Minute)
	m.On("Run", mock.Anything).Return(testErr)

	assert.Equal(t, "failing-job", m.Name())
	assert.Equal(t, "@hourly", m.Schedule())
	assert.Equal(t, 5*time.Minute, m.Timeout())
	assert.ErrorIs(t, m.Run(context.Background()), testErr)
}

func TestSimpleJob(t *testing.T) {
	j := NewSimpleJob("cleanup", "@daily")

	assert.Equal(t, "cleanup", j.Name())
	assert.Equal(t, "@daily", j.Schedule())
	assert.Equal(t, 30*time.Second, j.Timeout())
	assert.Equal(t, int32(0), j.RunCount.Load())

	require.NoError(t, j.Run(context.Background()))
	assert.Equal(t, int32(1), j.RunCount.Load())

	require.NoError(t, j.Run(context.Background()))
	assert.Equal(t, int32(2), j.RunCount.Load())
}

func TestSimpleJobWithError(t *testing.T) {
	j := NewSimpleJob("error-job", "@hourly")
	jobErr := errors.New("job error")
	j.RunErr = jobErr

	err := j.Run(context.Background())
	assert.ErrorIs(t, err, jobErr)
	assert.Equal(t, int32(1), j.RunCount.Load()) // Still tracks run count
}

func TestSimpleJobSetTimeout(t *testing.T) {
	j := NewSimpleJob("test", "@hourly")
	j.SetTimeout(5 * time.Minute)
	assert.Equal(t, 5*time.Minute, j.Timeout())
}

func TestMockResolver(t *testing.T) {
	r := NewMockResolver()
	job := NewSimpleJob("test", "@hourly")

	r.On("ResolveByName", "*cron.TestJob", []string(nil)).Return(job, nil)

	result, err := r.ResolveByName("*cron.TestJob", nil)
	require.NoError(t, err)
	assert.Equal(t, job, result)

	r.AssertCalled(t, "ResolveByName", "*cron.TestJob", []string(nil))
}

func TestMockResolverWithError(t *testing.T) {
	r := NewMockResolver()
	resolveErr := errors.New("service not found")

	r.On("ResolveByName", "unknown", []string(nil)).Return(nil, resolveErr)

	result, err := r.ResolveByName("unknown", nil)
	assert.Nil(t, result)
	assert.ErrorIs(t, err, resolveErr)
}

func TestTestScheduler(t *testing.T) {
	// Test with nil resolver and nil logger
	s := TestScheduler(nil, nil)
	require.NotNil(t, s)
	assert.False(t, s.IsRunning())
	assert.Equal(t, 0, s.JobCount())
}

func TestTestSchedulerWithCustomResolver(t *testing.T) {
	r := NewMockResolver()
	s := TestScheduler(r, nil)
	require.NotNil(t, s)
}

func TestRequireJobRan(t *testing.T) {
	j := NewSimpleJob("test", "@hourly")
	j.RunCount.Store(1)
	RequireJobRan(t, j) // Should not fail
}

func TestRequireJobRunCount(t *testing.T) {
	j := NewSimpleJob("test", "@hourly")
	j.RunCount.Store(3)
	RequireJobRunCount(t, j, 3) // Should not fail
}

func TestRequireJobNotRan(t *testing.T) {
	j := NewSimpleJob("test", "@hourly")
	RequireJobNotRan(t, j) // Should not fail - job hasn't run
}

func TestRequireSchedulerRunning(t *testing.T) {
	s := TestScheduler(nil, nil)
	require.NoError(t, s.OnStart(context.Background()))
	t.Cleanup(func() { _ = s.OnStop(context.Background()) })

	RequireSchedulerRunning(t, s) // Should not fail
}

func TestRequireSchedulerNotRunning(t *testing.T) {
	s := TestScheduler(nil, nil)
	RequireSchedulerNotRunning(t, s) // Should not fail - scheduler not started
}
