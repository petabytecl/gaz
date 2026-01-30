package cron

import (
	"context"
	"errors"
	"log/slog"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// mockCronJob is a minimal CronJob implementation for scheduler tests.
type mockCronJob struct {
	name     string
	schedule string
	timeout  time.Duration
	runFn    func(ctx context.Context) error
	runCount int
	mu       sync.Mutex
}

func (m *mockCronJob) Name() string           { return m.name }
func (m *mockCronJob) Schedule() string       { return m.schedule }
func (m *mockCronJob) Timeout() time.Duration { return m.timeout }
func (m *mockCronJob) Run(ctx context.Context) error {
	m.mu.Lock()
	m.runCount++
	m.mu.Unlock()
	if m.runFn != nil {
		return m.runFn(ctx)
	}
	return nil
}

func (m *mockCronJob) getRunCount() int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.runCount
}

// mockResolver is a minimal Resolver implementation for scheduler tests.
type mockResolver struct {
	services     map[string]any
	resolveCalls int
	mu           sync.Mutex
}

func newMockResolver() *mockResolver {
	return &mockResolver{
		services: make(map[string]any),
	}
}

func (r *mockResolver) ResolveByName(name string, opts []string) (any, error) {
	r.mu.Lock()
	r.resolveCalls++
	r.mu.Unlock()

	if svc, ok := r.services[name]; ok {
		return svc, nil
	}
	return nil, errors.New("service not found: " + name)
}

func (r *mockResolver) getResolveCalls() int {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.resolveCalls
}

func TestNewScheduler(t *testing.T) {
	resolver := newMockResolver()
	ctx := context.Background()
	logger := slog.Default()

	scheduler := NewScheduler(resolver, ctx, logger)

	require.NotNil(t, scheduler)
	assert.NotNil(t, scheduler.cron)
	assert.NotNil(t, scheduler.logger)
	assert.Equal(t, resolver, scheduler.resolver)
	assert.Equal(t, ctx, scheduler.appCtx)
	assert.False(t, scheduler.running)
	assert.Empty(t, scheduler.jobs)
}

func TestScheduler_Name(t *testing.T) {
	resolver := newMockResolver()
	ctx := context.Background()
	logger := slog.Default()

	scheduler := NewScheduler(resolver, ctx, logger)

	assert.Equal(t, "cron.Scheduler", scheduler.Name())
}

func TestScheduler_RegisterJob_Valid(t *testing.T) {
	resolver := newMockResolver()
	job := &mockCronJob{name: "test-job", schedule: "@every 1h"}
	resolver.services["*cron.mockCronJob"] = job

	ctx := context.Background()
	logger := slog.Default()

	scheduler := NewScheduler(resolver, ctx, logger)

	err := scheduler.RegisterJob("*cron.mockCronJob", "test-job", "@every 1h", 0)

	require.NoError(t, err)
	assert.Equal(t, 1, scheduler.JobCount())
}

func TestScheduler_RegisterJob_EmptySchedule(t *testing.T) {
	resolver := newMockResolver()
	ctx := context.Background()
	logger := slog.Default()

	scheduler := NewScheduler(resolver, ctx, logger)

	// Empty schedule should not register (job disabled) and not return error
	err := scheduler.RegisterJob("*cron.mockCronJob", "disabled-job", "", 0)

	require.NoError(t, err)
	assert.Equal(t, 0, scheduler.JobCount())
}

func TestScheduler_RegisterJob_InvalidSchedule(t *testing.T) {
	resolver := newMockResolver()
	ctx := context.Background()
	logger := slog.Default()

	scheduler := NewScheduler(resolver, ctx, logger)

	// Invalid cron expression should return error
	err := scheduler.RegisterJob("*cron.mockCronJob", "invalid-job", "not-a-cron-expression", 0)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid schedule")
	assert.Equal(t, 0, scheduler.JobCount())
}

func TestScheduler_StartStop(t *testing.T) {
	resolver := newMockResolver()
	ctx := context.Background()
	logger := slog.Default()

	scheduler := NewScheduler(resolver, ctx, logger)

	// Initial state
	assert.False(t, scheduler.IsRunning())

	// Start
	scheduler.Start()
	assert.True(t, scheduler.IsRunning())

	// Double start is idempotent
	scheduler.Start()
	assert.True(t, scheduler.IsRunning())

	// Stop
	scheduler.Stop()
	assert.False(t, scheduler.IsRunning())

	// Double stop is idempotent
	scheduler.Stop()
	assert.False(t, scheduler.IsRunning())
}

func TestScheduler_JobCount(t *testing.T) {
	resolver := newMockResolver()
	ctx := context.Background()
	logger := slog.Default()

	scheduler := NewScheduler(resolver, ctx, logger)

	assert.Equal(t, 0, scheduler.JobCount())

	err := scheduler.RegisterJob("job1", "job-1", "@hourly", 0)
	require.NoError(t, err)
	assert.Equal(t, 1, scheduler.JobCount())

	err = scheduler.RegisterJob("job2", "job-2", "@daily", 0)
	require.NoError(t, err)
	assert.Equal(t, 2, scheduler.JobCount())

	// Empty schedule doesn't add to count
	err = scheduler.RegisterJob("job3", "job-3", "", 0)
	require.NoError(t, err)
	assert.Equal(t, 2, scheduler.JobCount())
}

func TestScheduler_HealthCheck_Running(t *testing.T) {
	resolver := newMockResolver()
	ctx := context.Background()
	logger := slog.Default()

	scheduler := NewScheduler(resolver, ctx, logger)
	scheduler.Start()
	defer scheduler.Stop()

	err := scheduler.HealthCheck(context.Background())
	assert.NoError(t, err)
}

func TestScheduler_HealthCheck_NotRunning(t *testing.T) {
	resolver := newMockResolver()
	ctx := context.Background()
	logger := slog.Default()

	scheduler := NewScheduler(resolver, ctx, logger)

	err := scheduler.HealthCheck(context.Background())
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "scheduler not running")
}

func TestScheduler_Jobs(t *testing.T) {
	resolver := newMockResolver()
	ctx := context.Background()
	logger := slog.Default()

	scheduler := NewScheduler(resolver, ctx, logger)

	err := scheduler.RegisterJob("job1", "job-1", "@hourly", time.Minute)
	require.NoError(t, err)

	err = scheduler.RegisterJob("job2", "job-2", "@daily", 5*time.Minute)
	require.NoError(t, err)

	jobs := scheduler.Jobs()
	assert.Len(t, jobs, 2)

	// Verify it's a copy (modifying returned slice doesn't affect scheduler)
	jobs[0] = nil
	assert.Len(t, scheduler.Jobs(), 2)
	assert.NotNil(t, scheduler.Jobs()[0])
}

func TestScheduler_IsRunning_ThreadSafe(t *testing.T) {
	resolver := newMockResolver()
	ctx := context.Background()
	logger := slog.Default()

	scheduler := NewScheduler(resolver, ctx, logger)

	var wg sync.WaitGroup
	for range 10 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			scheduler.Start()
			_ = scheduler.IsRunning()
			scheduler.Stop()
		}()
	}

	wg.Wait()
	// If we get here without race detector complaints, thread safety is good
}

func TestScheduler_RegisterMultipleJobs(t *testing.T) {
	resolver := newMockResolver()
	ctx := context.Background()
	logger := slog.Default()

	scheduler := NewScheduler(resolver, ctx, logger)

	schedules := []struct {
		name     string
		schedule string
	}{
		{"job1", "@every 1m"},
		{"job2", "@every 5m"},
		{"job3", "@hourly"},
		{"job4", "@daily"},
		{"job5", "0 * * * *"},
	}

	for _, s := range schedules {
		err := scheduler.RegisterJob(s.name, s.name, s.schedule, 0)
		require.NoError(t, err, "failed to register %s", s.name)
	}

	assert.Equal(t, 5, scheduler.JobCount())
}

func TestScheduler_PredefinedSchedules(t *testing.T) {
	resolver := newMockResolver()
	ctx := context.Background()
	logger := slog.Default()

	scheduler := NewScheduler(resolver, ctx, logger)

	// Test all predefined schedules
	predefined := []string{
		"@yearly",
		"@monthly",
		"@weekly",
		"@daily",
		"@hourly",
		"@every 30s",
		"@every 5m",
		"@every 1h30m",
	}

	for i, sched := range predefined {
		err := scheduler.RegisterJob("job", "job-"+sched, sched, 0)
		require.NoError(t, err, "predefined schedule %s should be valid", sched)
		assert.Equal(t, i+1, scheduler.JobCount())
	}
}
