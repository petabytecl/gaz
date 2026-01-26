package gaz

import (
	"context"
	"errors"
	"sync"
	"syscall"
	"testing"
	"time"

	"github.com/stretchr/testify/suite"
)

type AppTestSuite struct {
	suite.Suite
}

func TestAppTestSuite(t *testing.T) {
	suite.Run(t, new(AppTestSuite))
}

type (
	AppTestServiceA struct{}
	AppTestServiceB struct{ A *AppTestServiceA }
)

func (s *AppTestSuite) TestRunAndStop() {
	c := New()

	var startOrder []string
	var stopOrder []string
	var mu sync.Mutex

	recordStart := func(name string) {
		mu.Lock()
		startOrder = append(startOrder, name)
		mu.Unlock()
	}

	recordStop := func(name string) {
		mu.Lock()
		stopOrder = append(stopOrder, name)
		mu.Unlock()
	}

	// Service A (Leaf dependency)
	err := For[*AppTestServiceA](c).Named("A").Eager().
		OnStart(func(_ context.Context, _ *AppTestServiceA) error {
			recordStart("A")
			return nil
		}).
		OnStop(func(_ context.Context, _ *AppTestServiceA) error {
			recordStop("A")
			return nil
		}).
		Provider(func(_ *Container) (*AppTestServiceA, error) { return &AppTestServiceA{}, nil })
	s.Require().NoError(err)

	// Service B depends on A
	err = For[*AppTestServiceB](c).Named("B").Eager().
		OnStart(func(_ context.Context, _ *AppTestServiceB) error {
			recordStart("B")
			return nil
		}).
		OnStop(func(_ context.Context, _ *AppTestServiceB) error {
			recordStop("B")
			return nil
		}).
		Provider(func(c *Container) (*AppTestServiceB, error) {
			a, resolveErr := Resolve[*AppTestServiceA](c, Named("A"))
			if resolveErr != nil {
				return nil, resolveErr
			}
			return &AppTestServiceB{A: a}, nil
		})
	s.Require().NoError(err)

	app := NewApp(c)

	// Run in goroutine because it blocks
	runErr := make(chan error, 1)
	go func() {
		runErr <- app.Run(context.Background())
	}()

	// Wait a bit for startup
	// Ideally we need a way to know it started.
	// We can check startOrder length?
	// Poll for len(startOrder) == 2
	s.Eventually(func() bool {
		mu.Lock()
		defer mu.Unlock()
		return len(startOrder) == 2
	}, 1*time.Second, 10*time.Millisecond)

	// Stop the app
	err = app.Stop(context.Background())
	s.Require().NoError(err)

	// Wait for Run to return
	select {
	case runResult := <-runErr:
		s.Require().NoError(runResult)
	case <-time.After(1 * time.Second):
		s.Fail("Run did not return after Stop")
	}

	mu.Lock()
	defer mu.Unlock()
	s.Equal([]string{"A", "B"}, startOrder)
	s.Equal([]string{"B", "A"}, stopOrder)
}

func (s *AppTestSuite) TestSignalHandling() {
	c := New()
	app := NewApp(c)

	// Run in goroutine
	runErr := make(chan error, 1)
	go func() {
		runErr <- app.Run(context.Background())
	}()

	// Wait for startup
	time.Sleep(50 * time.Millisecond)

	// Send signal
	err := syscall.Kill(syscall.Getpid(), syscall.SIGTERM)
	s.Require().NoError(err)

	// Wait for Run to return
	select {
	case err := <-runErr:
		s.Require().NoError(err)
	case <-time.After(1 * time.Second):
		s.Fail("Run did not return after SIGTERM")
	}
}

func (s *AppTestSuite) TestWithShutdownTimeout() {
	c := New()
	timeout := 5 * time.Second
	app := NewApp(c, WithShutdownTimeout(timeout))

	s.Equal(timeout, app.opts.ShutdownTimeout, "shutdown timeout should be set")
}

func (s *AppTestSuite) TestRunAlreadyRunning() {
	c := New()
	app := NewApp(c)

	// Start in background
	runErr := make(chan error, 1)
	go func() {
		runErr <- app.Run(context.Background())
	}()

	// Wait for startup
	time.Sleep(50 * time.Millisecond)

	// Try to run again - should error
	err := app.Run(context.Background())
	s.Require().Error(err)
	s.Contains(err.Error(), "already running")

	// Stop the first run
	s.Require().NoError(app.Stop(context.Background()))

	select {
	case runResult := <-runErr:
		s.Require().NoError(runResult)
	case <-time.After(1 * time.Second):
		s.Fail("Run did not return after Stop")
	}
}

func (s *AppTestSuite) TestRunContextCancelled() {
	c := New()
	app := NewApp(c)

	ctx, cancel := context.WithCancel(context.Background())

	// Run in goroutine
	runErr := make(chan error, 1)
	go func() {
		runErr <- app.Run(ctx)
	}()

	// Wait for startup
	time.Sleep(50 * time.Millisecond)

	// Cancel the context
	cancel()

	// Wait for Run to return
	select {
	case err := <-runErr:
		s.Require().NoError(err)
	case <-time.After(1 * time.Second):
		s.Fail("Run did not return after context cancellation")
	}
}

func (s *AppTestSuite) TestStopNotRunning() {
	c := New()
	app := NewApp(c)

	// Stop when not running should be no-op
	err := app.Stop(context.Background())
	s.Require().NoError(err)
}

type FailingStartService struct{}

func (s *AppTestSuite) TestRunStartError() {
	c := New()

	err := For[*FailingStartService](c).Eager().
		OnStart(func(_ context.Context, _ *FailingStartService) error {
			return errors.New("start failed")
		}).
		ProviderFunc(func(_ *Container) *FailingStartService {
			return &FailingStartService{}
		})
	s.Require().NoError(err)

	app := NewApp(c)
	err = app.Run(context.Background())
	s.Require().Error(err)
	s.Contains(err.Error(), "starting service")
}

type FailingStopService struct{}

func (s *AppTestSuite) TestStopError() {
	c := New()

	err := For[*FailingStopService](c).Named("failstop").Eager().
		OnStop(func(_ context.Context, _ *FailingStopService) error {
			return errors.New("stop failed")
		}).
		ProviderFunc(func(_ *Container) *FailingStopService {
			return &FailingStopService{}
		})
	s.Require().NoError(err)

	app := NewApp(c)

	// Run in background
	runErr := make(chan error, 1)
	go func() {
		runErr <- app.Run(context.Background())
	}()

	// Wait for startup
	time.Sleep(50 * time.Millisecond)

	// Stop should collect the error
	err = app.Stop(context.Background())
	s.Require().Error(err)
	s.Contains(err.Error(), "stopping service")

	select {
	case <-runErr:
	case <-time.After(1 * time.Second):
		s.Fail("Run did not return after Stop")
	}
}
