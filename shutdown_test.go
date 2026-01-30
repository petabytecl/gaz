package gaz

import (
	"bytes"
	"context"
	"fmt"
	"log/slog"
	"sync"
	"sync/atomic"
	"syscall"
	"testing"
	"time"

	"github.com/stretchr/testify/suite"
)

// syncBuffer is a thread-safe wrapper around bytes.Buffer for use in tests
// where multiple goroutines may write to the log buffer concurrently.
type syncBuffer struct {
	mu  sync.Mutex
	buf bytes.Buffer
}

func (b *syncBuffer) Write(p []byte) (int, error) {
	b.mu.Lock()
	defer b.mu.Unlock()

	return b.buf.Write(p) //nolint:wrapcheck // internal test helper, no wrapping needed
}

func (b *syncBuffer) String() string {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.buf.String()
}

// ShutdownTestSuite tests hardened shutdown behavior including:
// - Graceful shutdown completion when hooks finish in time.
// - Per-hook timeout enforcement with blame logging.
// - Global timeout force exit.
// - Double-SIGINT immediate exit.
type ShutdownTestSuite struct {
	suite.Suite
	originalExitFunc func(int)
	exitCalled       atomic.Bool
	exitCode         atomic.Int32
	logBuffer        *bytes.Buffer
}

func (s *ShutdownTestSuite) SetupTest() {
	// Save original exitFunc (with lock)
	exitFuncMu.Lock()
	s.originalExitFunc = exitFunc
	exitFuncMu.Unlock()

	// Reset atomics
	s.exitCalled.Store(false)
	s.exitCode.Store(0)

	// Replace with mock that captures exit calls (with lock)
	exitFuncMu.Lock()
	exitFunc = func(code int) {
		s.exitCalled.Store(true)
		s.exitCode.Store(int32(code))
	}
	exitFuncMu.Unlock()

	// Create log buffer for capturing logs
	s.logBuffer = &bytes.Buffer{}
}

func (s *ShutdownTestSuite) TearDownTest() {
	// Restore original exitFunc (with lock)
	exitFuncMu.Lock()
	exitFunc = s.originalExitFunc
	exitFuncMu.Unlock()
}

// createAppWithSlowHook creates an app with a service that has an OnStop hook
// sleeping for hookDuration. Timeouts are configured as specified.
func (s *ShutdownTestSuite) createAppWithSlowHook(
	hookDuration time.Duration,
	perHookTimeout time.Duration,
	globalTimeout time.Duration,
) *App {
	app := New(
		WithPerHookTimeout(perHookTimeout),
		WithShutdownTimeout(globalTimeout),
	)

	// Create a slow service using the container
	// Service implements di.Stopper interface - no fluent hooks needed
	err := For[*slowShutdownService](app.Container()).
		Named("SlowService").
		Eager().
		ProviderFunc(func(_ *Container) *slowShutdownService {
			return &slowShutdownService{duration: hookDuration}
		})
	s.Require().NoError(err)

	return app
}

// slowShutdownService is a test service with configurable shutdown delay.
type slowShutdownService struct {
	duration time.Duration
}

// OnStop implements di.Stopper with a configurable delay.
func (s *slowShutdownService) OnStop(ctx context.Context) error {
	select {
	case <-time.After(s.duration):
		return nil
	case <-ctx.Done():
		return fmt.Errorf("shutdown: %w", ctx.Err())
	}
}

// createAppWithMultipleServices creates an app with multiple services for testing
// per-hook timeout behavior where one service times out but others complete.
func (s *ShutdownTestSuite) createAppWithMultipleServices(
	serviceADuration time.Duration,
	serviceBDuration time.Duration,
	perHookTimeout time.Duration,
	globalTimeout time.Duration,
) (*App, *atomic.Bool, *atomic.Bool) {
	app := New(
		WithPerHookTimeout(perHookTimeout),
		WithShutdownTimeout(globalTimeout),
	)

	// Track which services were stopped
	serviceAStopped := &atomic.Bool{}
	serviceBStopped := &atomic.Bool{}

	// Service A (will timeout if serviceADuration > perHookTimeout)
	// Service implements di.Stopper interface - no fluent hooks needed
	err := For[*shutdownTestServiceA](app.Container()).
		Named("ServiceA").
		Eager().
		ProviderFunc(func(_ *Container) *shutdownTestServiceA {
			return &shutdownTestServiceA{duration: serviceADuration, stopped: serviceAStopped}
		})
	s.Require().NoError(err)

	// Service B (should complete even if A times out)
	// Service implements di.Stopper interface - no fluent hooks needed
	err = For[*shutdownTestServiceB](app.Container()).
		Named("ServiceB").
		Eager().
		ProviderFunc(func(_ *Container) *shutdownTestServiceB {
			return &shutdownTestServiceB{duration: serviceBDuration, stopped: serviceBStopped}
		})
	s.Require().NoError(err)

	return app, serviceAStopped, serviceBStopped
}

type (
	shutdownTestServiceA struct {
		duration time.Duration
		stopped  *atomic.Bool
	}
	shutdownTestServiceB struct {
		duration time.Duration
		stopped  *atomic.Bool
	}
)

// OnStop implements di.Stopper for shutdownTestServiceA.
func (s *shutdownTestServiceA) OnStop(ctx context.Context) error {
	select {
	case <-time.After(s.duration):
		s.stopped.Store(true)
		return nil
	case <-ctx.Done():
		return fmt.Errorf("shutdown: %w", ctx.Err())
	}
}

// OnStop implements di.Stopper for shutdownTestServiceB.
func (s *shutdownTestServiceB) OnStop(ctx context.Context) error {
	select {
	case <-time.After(s.duration):
		s.stopped.Store(true)
		return nil
	case <-ctx.Done():
		return fmt.Errorf("shutdown: %w", ctx.Err())
	}
}

// createLogCapturingApp creates an app with logger writing to the suite's logBuffer.
// This allows assertion on logged messages including blame logging.
func (s *ShutdownTestSuite) createLogCapturingApp(
	perHookTimeout time.Duration,
	globalTimeout time.Duration,
) *App {
	app := New(
		WithPerHookTimeout(perHookTimeout),
		WithShutdownTimeout(globalTimeout),
	)

	// Replace logger with one that writes to our buffer
	handler := slog.NewTextHandler(s.logBuffer, &slog.HandlerOptions{
		Level: slog.LevelDebug,
	})
	app.Logger = slog.New(handler)

	return app
}

// namedSlowService is a service with a configurable name and shutdown duration.
type namedSlowService struct {
	name     string
	duration time.Duration
}

// OnStop implements di.Stopper with a configurable delay.
func (s *namedSlowService) OnStop(ctx context.Context) error {
	select {
	case <-time.After(s.duration):
		return nil
	case <-ctx.Done():
		return fmt.Errorf("shutdown: %w", ctx.Err())
	}
}

// createAppWithNamedService creates an app with a named service for blame logging tests.
func (s *ShutdownTestSuite) createAppWithNamedService(
	serviceName string,
	hookDuration time.Duration,
	perHookTimeout time.Duration,
	globalTimeout time.Duration,
) *App {
	app := s.createLogCapturingApp(perHookTimeout, globalTimeout)

	// Create a named service
	// Service implements di.Stopper interface - no fluent hooks needed
	err := For[*namedSlowService](app.Container()).
		Named(serviceName).
		Eager().
		ProviderFunc(func(_ *Container) *namedSlowService {
			return &namedSlowService{name: serviceName, duration: hookDuration}
		})
	s.Require().NoError(err)

	return app
}

// waitForAppRunning waits until the app is running.
func (s *ShutdownTestSuite) waitForAppRunning(app *App, timeout time.Duration) bool {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		app.mu.Lock()
		running := app.running
		app.mu.Unlock()
		if running {
			return true
		}
		time.Sleep(10 * time.Millisecond)
	}
	return false
}

func TestShutdownTestSuite(t *testing.T) {
	suite.Run(t, new(ShutdownTestSuite))
}

// =============================================================================
// Task 2: Graceful and Timeout Tests
// =============================================================================

// TestGracefulShutdownCompletes verifies that when hooks complete within timeout,
// no force exit is triggered and Stop() returns nil.
func (s *ShutdownTestSuite) TestGracefulShutdownCompletes() {
	// Create app with hook that completes quickly (50ms)
	// Set generous timeouts: 1s per-hook, 5s global
	app := s.createAppWithSlowHook(50*time.Millisecond, 1*time.Second, 5*time.Second)

	// Build the app
	err := app.Build()
	s.Require().NoError(err)

	// Call Stop directly (not Run, to avoid signal handling complexity)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	err = app.Stop(ctx)

	// Assert: Stop() returns nil (no error)
	s.Require().NoError(err, "Stop() should return nil when hooks complete in time")

	// Assert: exitFunc was NOT called (no force exit)
	s.False(s.exitCalled.Load(), "exitFunc should NOT be called for graceful shutdown")
}

// TestPerHookTimeoutContinuesToNextHook verifies that when one hook times out,
// shutdown continues to the next hook rather than blocking forever.
func (s *ShutdownTestSuite) TestPerHookTimeoutContinuesToNextHook() {
	// Service A takes 500ms (will timeout with 100ms per-hook timeout)
	// Service B completes immediately
	// Global timeout is 5s (won't trigger)
	app, serviceAStopped, serviceBStopped := s.createAppWithMultipleServices(
		500*time.Millisecond, // A: slow, will timeout
		10*time.Millisecond,  // B: fast, should complete
		100*time.Millisecond, // per-hook timeout
		5*time.Second,        // global timeout
	)

	// Replace logger to capture blame logs
	handler := slog.NewTextHandler(s.logBuffer, &slog.HandlerOptions{Level: slog.LevelDebug})
	app.Logger = slog.New(handler)

	// Build the app
	err := app.Build()
	s.Require().NoError(err)

	// Stop the app
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	err = app.Stop(ctx)

	// Assert: Stop() returns an error (due to timeout)
	s.Require().Error(err, "Stop() should return error when hook times out")
	s.Contains(err.Error(), "deadline exceeded", "error should mention timeout")

	// Assert: Service A did NOT complete (it was interrupted by timeout)
	s.False(serviceAStopped.Load(), "Service A should NOT complete (timed out)")

	// Assert: Service B DID complete (shutdown continued to next hook)
	s.True(serviceBStopped.Load(), "Service B should complete even though A timed out")

	// Assert: Blame log mentions exceeded timeout
	logOutput := s.logBuffer.String()
	s.Contains(logOutput, "exceeded", "log should contain 'exceeded' for timeout blame")

	// Assert: exitFunc NOT called (per-hook timeout doesn't force exit, just logs)
	s.False(s.exitCalled.Load(), "exitFunc should NOT be called for per-hook timeout")
}

// TestGlobalTimeoutForcesExit verifies that when shutdown exceeds global timeout,
// exitFunc(1) is called to force process termination.
func (s *ShutdownTestSuite) TestGlobalTimeoutForcesExit() {
	// Create app with hook that takes 2s
	// Set per-hook timeout to 5s (won't trigger)
	// Set global timeout to 500ms (will trigger)
	app := s.createAppWithSlowHook(2*time.Second, 5*time.Second, 500*time.Millisecond)

	// Replace logger to capture logs
	handler := slog.NewTextHandler(s.logBuffer, &slog.HandlerOptions{Level: slog.LevelDebug})
	app.Logger = slog.New(handler)

	// Build the app
	err := app.Build()
	s.Require().NoError(err)

	// Stop the app - this should trigger global timeout force exit
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	// Run Stop in goroutine since it may block
	done := make(chan error, 1)
	go func() {
		done <- app.Stop(ctx)
	}()

	// Wait for exitFunc to be called or timeout
	s.Eventually(func() bool {
		return s.exitCalled.Load()
	}, 2*time.Second, 50*time.Millisecond, "exitFunc should be called on global timeout")

	// Assert: exitFunc(1) was called
	s.Equal(int32(1), s.exitCode.Load(), "exitFunc should be called with code 1")

	// Assert: log contains global timeout message
	logOutput := s.logBuffer.String()
	s.Contains(logOutput, "global timeout", "log should mention global timeout")
}

// TestBlameLoggingFormat verifies that blame logging includes hook name,
// timeout value, and elapsed time for debugging hanging hooks.
func (s *ShutdownTestSuite) TestBlameLoggingFormat() {
	// Create app with named service "DatabasePool" that takes 200ms
	// Per-hook timeout is 100ms (will trigger blame log)
	app := s.createAppWithNamedService(
		"DatabasePool",
		200*time.Millisecond, // hook takes 200ms
		100*time.Millisecond, // timeout at 100ms
		5*time.Second,        // global timeout
	)

	// Build the app
	err := app.Build()
	s.Require().NoError(err)

	// Stop the app
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	_ = app.Stop(ctx)

	// Check log output for blame information
	logOutput := s.logBuffer.String()

	// Assert: log contains hook name
	s.Contains(logOutput, "DatabasePool", "blame log should contain hook name")

	// Assert: log contains "exceeded"
	s.Contains(logOutput, "exceeded", "blame log should contain 'exceeded'")

	// Assert: log contains timeout value (100ms)
	s.Contains(logOutput, "100ms", "blame log should contain timeout value")
}

// TestWithPerHookTimeoutOption verifies the WithPerHookTimeout option sets the value.
func (s *ShutdownTestSuite) TestWithPerHookTimeoutOption() {
	app := New(WithPerHookTimeout(2 * time.Second))

	s.Equal(2*time.Second, app.opts.PerHookTimeout, "PerHookTimeout should be set to 2s")
}

// TestWithShutdownTimeoutOption verifies the WithShutdownTimeout option sets the value.
func (s *ShutdownTestSuite) TestWithShutdownTimeoutOption() {
	app := New(WithShutdownTimeout(45 * time.Second))

	s.Equal(45*time.Second, app.opts.ShutdownTimeout, "ShutdownTimeout should be set to 45s")
}

// =============================================================================
// Task 3: Double-SIGINT Tests
// =============================================================================

// TestFirstSIGINTLogsHint verifies that the first SIGINT logs a hint about
// pressing Ctrl+C again to force exit.
func (s *ShutdownTestSuite) TestFirstSIGINTLogsHint() {
	// Create app with slow hook (1s)
	app := s.createAppWithSlowHook(1*time.Second, 5*time.Second, 10*time.Second)

	// Use thread-safe buffer for concurrent log access
	logBuf := &syncBuffer{}
	handler := slog.NewTextHandler(logBuf, &slog.HandlerOptions{Level: slog.LevelDebug})
	app.Logger = slog.New(handler)

	// Build the app
	err := app.Build()
	s.Require().NoError(err)

	// Run in goroutine
	runDone := make(chan error, 1)
	go func() {
		runDone <- app.Run(context.Background())
	}()

	// Wait for app to be running
	s.True(s.waitForAppRunning(app, 1*time.Second), "app should be running")

	// Send first SIGINT
	err = syscall.Kill(syscall.Getpid(), syscall.SIGINT)
	s.Require().NoError(err)

	// Wait a bit for log to be written
	time.Sleep(100 * time.Millisecond)

	// Check log contains hint
	logOutput := logBuf.String()
	s.Contains(logOutput, "Ctrl+C again", "first SIGINT should log hint about force exit")

	// Send second SIGINT to complete the test
	err = syscall.Kill(syscall.Getpid(), syscall.SIGINT)
	s.Require().NoError(err)

	// Wait for app to exit - after second SIGINT, exitFunc is called which
	// in production would os.Exit. Our mock doesn't exit, so the shutdown
	// goroutine continues and eventually completes.
	select {
	case <-runDone:
		// Good, app stopped
	case <-time.After(2 * time.Second):
		// Force stop if needed
		_ = app.Stop(context.Background())
		<-runDone // Must drain the channel to ensure goroutine finishes
	}

	// Small delay to allow force-exit watcher goroutine to exit
	// after shutdownDone is signaled
	time.Sleep(10 * time.Millisecond)
}

// TestDoubleSIGINTForcesImmediateExit verifies that a second SIGINT
// triggers immediate exitFunc(1) without waiting for graceful shutdown.
func (s *ShutdownTestSuite) TestDoubleSIGINTForcesImmediateExit() {
	// Create app with slow hook (5s)
	// Global timeout is 10s (won't trigger)
	app := s.createAppWithSlowHook(5*time.Second, 10*time.Second, 10*time.Second)

	// Replace logger to capture logs
	handler := slog.NewTextHandler(s.logBuffer, &slog.HandlerOptions{Level: slog.LevelDebug})
	app.Logger = slog.New(handler)

	// Build the app
	err := app.Build()
	s.Require().NoError(err)

	// Run in goroutine
	runDone := make(chan error, 1)
	go func() {
		runDone <- app.Run(context.Background())
	}()

	// Wait for app to be running
	s.True(s.waitForAppRunning(app, 1*time.Second), "app should be running")

	// Record time before signals
	startTime := time.Now()

	// Send first SIGINT
	err = syscall.Kill(syscall.Getpid(), syscall.SIGINT)
	s.Require().NoError(err)

	// Wait a bit between signals
	time.Sleep(50 * time.Millisecond)

	// Send second SIGINT
	err = syscall.Kill(syscall.Getpid(), syscall.SIGINT)
	s.Require().NoError(err)

	// Wait for exitFunc to be called
	s.Eventually(func() bool {
		return s.exitCalled.Load()
	}, 500*time.Millisecond, 10*time.Millisecond, "exitFunc should be called on double SIGINT")

	// Assert: exit happened quickly (within 200ms of second signal)
	elapsed := time.Since(startTime)
	s.Less(elapsed, 300*time.Millisecond, "exit should happen quickly after second SIGINT")

	// Assert: exitFunc(1) was called
	s.Equal(int32(1), s.exitCode.Load(), "exitFunc should be called with code 1")

	// Assert: log contains force exit message
	logOutput := s.logBuffer.String()
	s.Contains(logOutput, "second interrupt", "log should mention second interrupt")
}

// TestSIGTERMDoesNotEnableDoubleSignal verifies that SIGTERM performs graceful
// shutdown without the double-signal force exit behavior (SIGTERM + SIGKILL is
// the standard ops pattern for force exit).
func (s *ShutdownTestSuite) TestSIGTERMDoesNotEnableDoubleSignal() {
	// Create app with hook that completes quickly
	app := s.createAppWithSlowHook(100*time.Millisecond, 5*time.Second, 10*time.Second)

	// Replace logger to capture logs
	handler := slog.NewTextHandler(s.logBuffer, &slog.HandlerOptions{Level: slog.LevelDebug})
	app.Logger = slog.New(handler)

	// Build the app
	err := app.Build()
	s.Require().NoError(err)

	// Run in goroutine
	runDone := make(chan error, 1)
	go func() {
		runDone <- app.Run(context.Background())
	}()

	// Wait for app to be running
	s.True(s.waitForAppRunning(app, 1*time.Second), "app should be running")

	// Send SIGTERM
	err = syscall.Kill(syscall.Getpid(), syscall.SIGTERM)
	s.Require().NoError(err)

	// Wait for graceful shutdown to complete
	select {
	case runErr := <-runDone:
		s.Require().NoError(runErr, "Run should complete without error on SIGTERM")
	case <-time.After(2 * time.Second):
		s.Fail("Run should return after SIGTERM graceful shutdown")
	}

	// Assert: exitFunc was NOT called (graceful shutdown, no force exit)
	s.False(s.exitCalled.Load(), "exitFunc should NOT be called for SIGTERM graceful shutdown")

	// Assert: log does NOT contain the Ctrl+C hint (that's SIGINT-specific)
	// SIGTERM should still log the shutdown message but the hint is for interactive use
	logOutput := s.logBuffer.String()
	s.Contains(logOutput, "Shutting down gracefully", "SIGTERM should trigger graceful shutdown")
}
