package gaz

import (
	"bytes"
	"context"
	"log/slog"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/suite"
)

// ShutdownTestSuite tests hardened shutdown behavior including:
// - Graceful shutdown completion when hooks finish in time
// - Per-hook timeout enforcement with blame logging
// - Global timeout force exit
// - Double-SIGINT immediate exit
type ShutdownTestSuite struct {
	suite.Suite
	originalExitFunc func(int)
	exitCalled       atomic.Bool
	exitCode         atomic.Int32
	logBuffer        *bytes.Buffer
}

func (s *ShutdownTestSuite) SetupTest() {
	// Save original exitFunc
	s.originalExitFunc = exitFunc

	// Reset atomics
	s.exitCalled.Store(false)
	s.exitCode.Store(0)

	// Replace with mock that captures exit calls
	exitFunc = func(code int) {
		s.exitCalled.Store(true)
		s.exitCode.Store(int32(code))
	}

	// Create log buffer for capturing logs
	s.logBuffer = &bytes.Buffer{}
}

func (s *ShutdownTestSuite) TearDownTest() {
	// Restore original exitFunc
	exitFunc = s.originalExitFunc
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
	err := For[*slowShutdownService](app.Container()).
		Named("SlowService").
		Eager().
		OnStop(func(ctx context.Context, svc *slowShutdownService) error {
			select {
			case <-time.After(hookDuration):
				return nil
			case <-ctx.Done():
				return ctx.Err()
			}
		}).
		ProviderFunc(func(c *Container) *slowShutdownService {
			return &slowShutdownService{}
		})
	s.Require().NoError(err)

	return app
}

// slowShutdownService is a test service with configurable shutdown delay.
type slowShutdownService struct{}

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
	err := For[*shutdownTestServiceA](app.Container()).
		Named("ServiceA").
		Eager().
		OnStop(func(ctx context.Context, svc *shutdownTestServiceA) error {
			select {
			case <-time.After(serviceADuration):
				serviceAStopped.Store(true)
				return nil
			case <-ctx.Done():
				return ctx.Err()
			}
		}).
		ProviderFunc(func(c *Container) *shutdownTestServiceA {
			return &shutdownTestServiceA{}
		})
	s.Require().NoError(err)

	// Service B (should complete even if A times out)
	err = For[*shutdownTestServiceB](app.Container()).
		Named("ServiceB").
		Eager().
		OnStop(func(ctx context.Context, svc *shutdownTestServiceB) error {
			select {
			case <-time.After(serviceBDuration):
				serviceBStopped.Store(true)
				return nil
			case <-ctx.Done():
				return ctx.Err()
			}
		}).
		ProviderFunc(func(c *Container) *shutdownTestServiceB {
			return &shutdownTestServiceB{}
		})
	s.Require().NoError(err)

	return app, serviceAStopped, serviceBStopped
}

type shutdownTestServiceA struct{}
type shutdownTestServiceB struct{}

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
	stopped  *atomic.Bool
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
	err := For[*namedSlowService](app.Container()).
		Named(serviceName).
		Eager().
		OnStop(func(ctx context.Context, svc *namedSlowService) error {
			select {
			case <-time.After(hookDuration):
				return nil
			case <-ctx.Done():
				return ctx.Err()
			}
		}).
		ProviderFunc(func(c *Container) *namedSlowService {
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
