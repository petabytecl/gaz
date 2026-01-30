package cron

import (
	"bytes"
	"context"
	"errors"
	"log/slog"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// wrapperMockJob is a CronJob implementation for wrapper tests.
type wrapperMockJob struct {
	name     string
	schedule string
	timeout  time.Duration
	runFn    func(ctx context.Context) error
}

func (m *wrapperMockJob) Name() string           { return m.name }
func (m *wrapperMockJob) Schedule() string       { return m.schedule }
func (m *wrapperMockJob) Timeout() time.Duration { return m.timeout }
func (m *wrapperMockJob) Run(ctx context.Context) error {
	if m.runFn != nil {
		return m.runFn(ctx)
	}
	return nil
}

// countingResolver tracks resolve calls and returns fresh instances.
type countingResolver struct {
	services     map[string]func() any
	resolveCalls int
	resolveErr   error
	mu           sync.Mutex
}

func newCountingResolver() *countingResolver {
	return &countingResolver{
		services: make(map[string]func() any),
	}
}

func (r *countingResolver) ResolveByName(name string, opts []string) (any, error) {
	r.mu.Lock()
	r.resolveCalls++
	r.mu.Unlock()

	if r.resolveErr != nil {
		return nil, r.resolveErr
	}

	if factory, ok := r.services[name]; ok {
		return factory(), nil
	}
	return nil, errors.New("service not found: " + name)
}

func (r *countingResolver) getResolveCalls() int {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.resolveCalls
}

func TestNewJobWrapper(t *testing.T) {
	resolver := newCountingResolver()
	ctx := context.Background()
	logger := slog.Default()

	wrapper := NewJobWrapper(
		resolver,
		"*cron.TestJob",
		"test-job",
		"@hourly",
		5*time.Minute,
		ctx,
		logger,
	)

	require.NotNil(t, wrapper)
	assert.Equal(t, "test-job", wrapper.Name())
	assert.Equal(t, "@hourly", wrapper.Schedule())
	assert.False(t, wrapper.IsRunning())
	assert.True(t, wrapper.LastRun().IsZero())
	assert.NoError(t, wrapper.LastError())
}

func TestJobWrapper_Run_Success(t *testing.T) {
	var buf bytes.Buffer
	logger := slog.New(slog.NewTextHandler(&buf, nil))

	resolver := newCountingResolver()
	executed := false
	resolver.services["*cron.TestJob"] = func() any {
		return &wrapperMockJob{
			name:     "test-job",
			schedule: "@hourly",
			runFn: func(ctx context.Context) error {
				executed = true
				return nil
			},
		}
	}

	ctx := context.Background()
	wrapper := NewJobWrapper(resolver, "*cron.TestJob", "test-job", "@hourly", 0, ctx, logger)

	wrapper.Run()

	assert.True(t, executed)
	assert.False(t, wrapper.IsRunning())
	assert.False(t, wrapper.LastRun().IsZero())
	assert.NoError(t, wrapper.LastError())

	output := buf.String()
	assert.Contains(t, output, "job started")
	assert.Contains(t, output, "job finished")
	assert.Contains(t, output, "job=test-job")
}

func TestJobWrapper_Run_Error(t *testing.T) {
	var buf bytes.Buffer
	logger := slog.New(slog.NewTextHandler(&buf, nil))

	resolver := newCountingResolver()
	testErr := errors.New("job execution failed")
	resolver.services["*cron.TestJob"] = func() any {
		return &wrapperMockJob{
			name:     "error-job",
			schedule: "@hourly",
			runFn: func(ctx context.Context) error {
				return testErr
			},
		}
	}

	ctx := context.Background()
	wrapper := NewJobWrapper(resolver, "*cron.TestJob", "error-job", "@hourly", 0, ctx, logger)

	wrapper.Run()

	assert.False(t, wrapper.IsRunning())
	assert.False(t, wrapper.LastRun().IsZero())
	require.Error(t, wrapper.LastError())
	assert.Equal(t, testErr, wrapper.LastError())

	output := buf.String()
	assert.Contains(t, output, "job started")
	assert.Contains(t, output, "job failed")
	assert.Contains(t, output, "job execution failed")
}

func TestJobWrapper_Run_Panic(t *testing.T) {
	var buf bytes.Buffer
	logger := slog.New(slog.NewTextHandler(&buf, nil))

	resolver := newCountingResolver()
	resolver.services["*cron.PanicJob"] = func() any {
		return &wrapperMockJob{
			name:     "panic-job",
			schedule: "@every 1s",
			runFn: func(ctx context.Context) error {
				panic("test panic")
			},
		}
	}

	ctx := context.Background()
	wrapper := NewJobWrapper(resolver, "*cron.PanicJob", "panic-job", "@every 1s", 0, ctx, logger)

	// Run should NOT panic - it should recover
	require.NotPanics(t, func() {
		wrapper.Run()
	})

	assert.False(t, wrapper.IsRunning())
	assert.False(t, wrapper.LastRun().IsZero())
	require.Error(t, wrapper.LastError())
	assert.Contains(t, wrapper.LastError().Error(), "panic")

	output := buf.String()
	assert.Contains(t, output, "job panicked")
	assert.Contains(t, output, "test panic")
	assert.Contains(t, output, "stack") // Stack trace should be logged
}

func TestJobWrapper_Run_Timeout(t *testing.T) {
	var buf bytes.Buffer
	logger := slog.New(slog.NewTextHandler(&buf, nil))

	resolver := newCountingResolver()
	var ctxErr error
	resolver.services["*cron.TimeoutJob"] = func() any {
		return &wrapperMockJob{
			name:     "timeout-job",
			schedule: "@hourly",
			timeout:  50 * time.Millisecond,
			runFn: func(ctx context.Context) error {
				// Wait longer than timeout
				select {
				case <-ctx.Done():
					ctxErr = ctx.Err()
					return ctx.Err()
				case <-time.After(500 * time.Millisecond):
					return nil
				}
			},
		}
	}

	ctx := context.Background()
	wrapper := NewJobWrapper(resolver, "*cron.TimeoutJob", "timeout-job", "@hourly", 50*time.Millisecond, ctx, logger)

	wrapper.Run()

	// Context should have been cancelled due to timeout
	assert.Equal(t, context.DeadlineExceeded, ctxErr)
	assert.False(t, wrapper.IsRunning())
}

func TestJobWrapper_Run_ContextCancelled(t *testing.T) {
	var buf bytes.Buffer
	logger := slog.New(slog.NewTextHandler(&buf, nil))

	resolver := newCountingResolver()
	var ctxErr error
	resolver.services["*cron.CancelJob"] = func() any {
		return &wrapperMockJob{
			name:     "cancel-job",
			schedule: "@hourly",
			runFn: func(ctx context.Context) error {
				<-ctx.Done()
				ctxErr = ctx.Err()
				return ctx.Err()
			},
		}
	}

	ctx, cancel := context.WithCancel(context.Background())
	wrapper := NewJobWrapper(resolver, "*cron.CancelJob", "cancel-job", "@hourly", 0, ctx, logger)

	// Start job in background
	done := make(chan struct{})
	go func() {
		wrapper.Run()
		close(done)
	}()

	// Give the job time to start
	time.Sleep(50 * time.Millisecond)

	// Cancel app context
	cancel()

	// Wait for job to complete
	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("job did not complete after context cancellation")
	}

	assert.Equal(t, context.Canceled, ctxErr)
}

func TestJobWrapper_StatusTracking(t *testing.T) {
	resolver := newCountingResolver()
	started := make(chan struct{})
	proceed := make(chan struct{})

	resolver.services["*cron.StatusJob"] = func() any {
		return &wrapperMockJob{
			name:     "status-job",
			schedule: "@hourly",
			runFn: func(ctx context.Context) error {
				close(started)
				<-proceed
				return nil
			},
		}
	}

	ctx := context.Background()
	logger := slog.Default()
	wrapper := NewJobWrapper(resolver, "*cron.StatusJob", "status-job", "@hourly", 0, ctx, logger)

	// Initial state
	assert.False(t, wrapper.IsRunning())
	assert.True(t, wrapper.LastRun().IsZero())
	assert.NoError(t, wrapper.LastError())

	// Start job in background
	done := make(chan struct{})
	go func() {
		wrapper.Run()
		close(done)
	}()

	// Wait for job to start
	<-started

	// While running
	assert.True(t, wrapper.IsRunning())

	// Let job complete
	close(proceed)
	<-done

	// After completion
	assert.False(t, wrapper.IsRunning())
	assert.False(t, wrapper.LastRun().IsZero())
	assert.NoError(t, wrapper.LastError())
}

func TestJobWrapper_TransientResolution(t *testing.T) {
	resolver := newCountingResolver()
	instanceCount := 0
	var mu sync.Mutex

	resolver.services["*cron.TransientJob"] = func() any {
		mu.Lock()
		instanceCount++
		mu.Unlock()
		return &wrapperMockJob{
			name:     "transient-job",
			schedule: "@hourly",
			runFn: func(ctx context.Context) error {
				return nil
			},
		}
	}

	ctx := context.Background()
	logger := slog.Default()
	wrapper := NewJobWrapper(resolver, "*cron.TransientJob", "transient-job", "@hourly", 0, ctx, logger)

	// Run multiple times
	wrapper.Run()
	wrapper.Run()
	wrapper.Run()

	// Each run should resolve a fresh instance
	mu.Lock()
	assert.Equal(t, 3, instanceCount)
	mu.Unlock()
	assert.Equal(t, 3, resolver.getResolveCalls())
}

func TestJobWrapper_ResolutionError(t *testing.T) {
	var buf bytes.Buffer
	logger := slog.New(slog.NewTextHandler(&buf, nil))

	resolver := newCountingResolver()
	resolver.resolveErr = errors.New("container error")

	ctx := context.Background()
	wrapper := NewJobWrapper(resolver, "*cron.MissingJob", "missing-job", "@hourly", 0, ctx, logger)

	// Should not panic
	require.NotPanics(t, func() {
		wrapper.Run()
	})

	assert.Error(t, wrapper.LastError())
	assert.Contains(t, wrapper.LastError().Error(), "resolve failed")

	output := buf.String()
	assert.Contains(t, output, "failed to resolve job")
}

func TestJobWrapper_TypeAssertionError(t *testing.T) {
	var buf bytes.Buffer
	logger := slog.New(slog.NewTextHandler(&buf, nil))

	resolver := newCountingResolver()
	// Return something that's not a CronJob
	resolver.services["*cron.WrongType"] = func() any {
		return "not a cron job"
	}

	ctx := context.Background()
	wrapper := NewJobWrapper(resolver, "*cron.WrongType", "wrong-type-job", "@hourly", 0, ctx, logger)

	wrapper.Run()

	assert.Error(t, wrapper.LastError())
	assert.Contains(t, wrapper.LastError().Error(), "type assertion failed")

	output := buf.String()
	assert.Contains(t, output, "resolved instance is not CronJob")
}

func TestJobWrapper_NoTimeout(t *testing.T) {
	var buf bytes.Buffer
	logger := slog.New(slog.NewTextHandler(&buf, nil))

	resolver := newCountingResolver()
	var receivedCtx context.Context
	resolver.services["*cron.NoTimeoutJob"] = func() any {
		return &wrapperMockJob{
			name:     "no-timeout-job",
			schedule: "@hourly",
			timeout:  0, // No timeout
			runFn: func(ctx context.Context) error {
				receivedCtx = ctx
				return nil
			},
		}
	}

	appCtx := context.Background()
	wrapper := NewJobWrapper(resolver, "*cron.NoTimeoutJob", "no-timeout-job", "@hourly", 0, appCtx, logger)

	wrapper.Run()

	// Context should not have deadline when timeout is 0
	_, hasDeadline := receivedCtx.Deadline()
	assert.False(t, hasDeadline, "context should not have deadline when timeout is 0")
}

func TestJobWrapper_WithTimeout(t *testing.T) {
	var buf bytes.Buffer
	logger := slog.New(slog.NewTextHandler(&buf, nil))

	resolver := newCountingResolver()
	var receivedCtx context.Context
	resolver.services["*cron.WithTimeoutJob"] = func() any {
		return &wrapperMockJob{
			name:     "with-timeout-job",
			schedule: "@hourly",
			runFn: func(ctx context.Context) error {
				receivedCtx = ctx
				return nil
			},
		}
	}

	appCtx := context.Background()
	wrapper := NewJobWrapper(resolver, "*cron.WithTimeoutJob", "with-timeout-job", "@hourly", 5*time.Minute, appCtx, logger)

	wrapper.Run()

	// Context should have deadline when timeout is set
	deadline, hasDeadline := receivedCtx.Deadline()
	assert.True(t, hasDeadline, "context should have deadline when timeout > 0")
	assert.True(t, deadline.After(time.Now().Add(4*time.Minute)), "deadline should be ~5 minutes from now")
}

func TestJobWrapper_ConcurrentAccess(t *testing.T) {
	resolver := newCountingResolver()
	resolver.services["*cron.ConcurrentJob"] = func() any {
		return &wrapperMockJob{
			name:     "concurrent-job",
			schedule: "@hourly",
			runFn: func(ctx context.Context) error {
				time.Sleep(10 * time.Millisecond)
				return nil
			},
		}
	}

	ctx := context.Background()
	logger := slog.Default()
	wrapper := NewJobWrapper(resolver, "*cron.ConcurrentJob", "concurrent-job", "@hourly", 0, ctx, logger)

	var wg sync.WaitGroup
	for range 10 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			wrapper.Run()
			_ = wrapper.IsRunning()
			_ = wrapper.LastRun()
			_ = wrapper.LastError()
		}()
	}

	wg.Wait()
	// If we get here without race detector complaints, thread safety is good
}

func TestJobWrapper_PanicWithValue(t *testing.T) {
	var buf bytes.Buffer
	logger := slog.New(slog.NewTextHandler(&buf, nil))

	resolver := newCountingResolver()
	resolver.services["*cron.PanicValueJob"] = func() any {
		return &wrapperMockJob{
			name:     "panic-value-job",
			schedule: "@hourly",
			runFn: func(ctx context.Context) error {
				panic(42) // Panic with non-string value
			},
		}
	}

	ctx := context.Background()
	wrapper := NewJobWrapper(resolver, "*cron.PanicValueJob", "panic-value-job", "@hourly", 0, ctx, logger)

	require.NotPanics(t, func() {
		wrapper.Run()
	})

	assert.Error(t, wrapper.LastError())
	assert.Contains(t, wrapper.LastError().Error(), "panic: 42")
}

func TestJobWrapper_PanicWithError(t *testing.T) {
	var buf bytes.Buffer
	logger := slog.New(slog.NewTextHandler(&buf, nil))

	resolver := newCountingResolver()
	panicErr := errors.New("panic error")
	resolver.services["*cron.PanicErrorJob"] = func() any {
		return &wrapperMockJob{
			name:     "panic-error-job",
			schedule: "@hourly",
			runFn: func(ctx context.Context) error {
				panic(panicErr)
			},
		}
	}

	ctx := context.Background()
	wrapper := NewJobWrapper(resolver, "*cron.PanicErrorJob", "panic-error-job", "@hourly", 0, ctx, logger)

	require.NotPanics(t, func() {
		wrapper.Run()
	})

	assert.Error(t, wrapper.LastError())
	output := buf.String()
	assert.Contains(t, output, "panic error")
}

func TestJobWrapper_DurationLogging(t *testing.T) {
	var buf bytes.Buffer
	logger := slog.New(slog.NewTextHandler(&buf, nil))

	resolver := newCountingResolver()
	resolver.services["*cron.DurationJob"] = func() any {
		return &wrapperMockJob{
			name:     "duration-job",
			schedule: "@hourly",
			runFn: func(ctx context.Context) error {
				time.Sleep(50 * time.Millisecond)
				return nil
			},
		}
	}

	ctx := context.Background()
	wrapper := NewJobWrapper(resolver, "*cron.DurationJob", "duration-job", "@hourly", 0, ctx, logger)

	wrapper.Run()

	output := buf.String()
	assert.Contains(t, output, "duration=")
	// Duration should be logged as part of "job finished" message
	assert.True(t, strings.Contains(output, "duration=") && strings.Contains(output, "job finished"))
}
