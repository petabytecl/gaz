package cron

import (
	"context"
	"io"
	"log/slog"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/mock"
)

// MockJob is a testify mock implementing cron.CronJob.
// Use NewMockJob() for a pre-configured instance.
type MockJob struct {
	mock.Mock
}

// Name returns the job's name.
func (m *MockJob) Name() string {
	args := m.Called()
	return args.String(0)
}

// Schedule returns the job's schedule.
func (m *MockJob) Schedule() string {
	args := m.Called()
	return args.String(0)
}

// Timeout returns the job's timeout.
func (m *MockJob) Timeout() time.Duration {
	args := m.Called()
	return args.Get(0).(time.Duration)
}

// Run executes the job and returns the mocked error.
func (m *MockJob) Run(ctx context.Context) error {
	args := m.Called(ctx)
	return args.Error(0)
}

// NewMockJob creates a MockJob with default expectations.
// Schedule returns "@every 1m", Timeout returns 30s, Run returns nil.
func NewMockJob(name string) *MockJob {
	m := &MockJob{}
	m.On("Name").Return(name)
	m.On("Schedule").Return("@every 1m")
	m.On("Timeout").Return(30 * time.Second)
	m.On("Run", mock.Anything).Return(nil)
	return m
}

// SimpleJob is a test job that tracks Run calls.
// Useful for cases where mock complexity isn't needed.
type SimpleJob struct {
	name     string
	schedule string
	timeout  time.Duration
	RunCount atomic.Int32
	RunErr   error
}

// NewSimpleJob creates a SimpleJob with the given name and schedule.
// Timeout defaults to 30 seconds.
func NewSimpleJob(name, schedule string) *SimpleJob {
	return &SimpleJob{
		name:     name,
		schedule: schedule,
		timeout:  30 * time.Second,
	}
}

// Name returns the job's name.
func (j *SimpleJob) Name() string { return j.name }

// Schedule returns the job's schedule.
func (j *SimpleJob) Schedule() string { return j.schedule }

// Timeout returns the job's timeout.
func (j *SimpleJob) Timeout() time.Duration { return j.timeout }

// Run increments the run count and returns RunErr.
func (j *SimpleJob) Run(_ context.Context) error {
	j.RunCount.Add(1)
	return j.RunErr
}

// SetTimeout sets the job's timeout.
func (j *SimpleJob) SetTimeout(d time.Duration) {
	j.timeout = d
}

// MockResolver is a testify mock implementing cron.Resolver.
// Use NewMockResolver() for a pre-configured instance.
type MockResolver struct {
	mock.Mock
}

// ResolveByName returns the mocked service instance.
func (m *MockResolver) ResolveByName(name string, opts []string) (any, error) {
	args := m.Called(name, opts)
	return args.Get(0), args.Error(1)
}

// NewMockResolver creates a MockResolver ready for test setup.
// Configure expectations with On("ResolveByName", ...).
func NewMockResolver() *MockResolver {
	return &MockResolver{}
}

// TestScheduler creates a cron.Scheduler suitable for testing.
// Jobs can be added but scheduler should not be started in most unit tests.
// If resolver is nil, a MockResolver is created.
// If logger is nil, a discard logger is used.
func TestScheduler(resolver Resolver, logger *slog.Logger) *Scheduler {
	if logger == nil {
		logger = slog.New(slog.NewTextHandler(io.Discard, nil))
	}
	if resolver == nil {
		resolver = NewMockResolver()
	}
	return NewScheduler(resolver, context.Background(), logger)
}

// RequireJobRan asserts the job was executed at least once.
func RequireJobRan(tb testing.TB, j *SimpleJob) {
	tb.Helper()
	if j.RunCount.Load() == 0 {
		tb.Fatalf("expected job %q to have run at least once", j.Name())
	}
}

// RequireJobRunCount asserts the job ran exactly n times.
func RequireJobRunCount(tb testing.TB, j *SimpleJob, expected int32) {
	tb.Helper()
	actual := j.RunCount.Load()
	if actual != expected {
		tb.Fatalf("expected job %q to run %d times, got %d", j.Name(), expected, actual)
	}
}

// RequireJobNotRan asserts the job was NOT executed.
func RequireJobNotRan(tb testing.TB, j *SimpleJob) {
	tb.Helper()
	if j.RunCount.Load() != 0 {
		tb.Fatalf("expected job %q to NOT have run, but ran %d times", j.Name(), j.RunCount.Load())
	}
}

// RequireSchedulerRunning asserts the scheduler is running.
func RequireSchedulerRunning(tb testing.TB, s *Scheduler) {
	tb.Helper()
	if !s.IsRunning() {
		tb.Fatal("expected scheduler to be running")
	}
}

// RequireSchedulerNotRunning asserts the scheduler is not running.
func RequireSchedulerNotRunning(tb testing.TB, s *Scheduler) {
	tb.Helper()
	if s.IsRunning() {
		tb.Fatal("expected scheduler to NOT be running")
	}
}
