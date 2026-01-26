package gaz

import (
	"context"
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

type AppTestServiceA struct{}
type AppTestServiceB struct{ A *AppTestServiceA }

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
	For[*AppTestServiceA](c).Named("A").Eager().
		OnStart(func(_ context.Context, _ *AppTestServiceA) error {
			recordStart("A")
			return nil
		}).
		OnStop(func(_ context.Context, _ *AppTestServiceA) error {
			recordStop("A")
			return nil
		}).
		Provider(func(_ *Container) (*AppTestServiceA, error) { return &AppTestServiceA{}, nil })

	// Service B depends on A
	For[*AppTestServiceB](c).Named("B").Eager().
		OnStart(func(_ context.Context, _ *AppTestServiceB) error {
			recordStart("B")
			return nil
		}).
		OnStop(func(_ context.Context, _ *AppTestServiceB) error {
			recordStop("B")
			return nil
		}).
		Provider(func(c *Container) (*AppTestServiceB, error) {
			a, err := Resolve[*AppTestServiceA](c, Named("A"))
			if err != nil {
				return nil, err
			}
			return &AppTestServiceB{A: a}, nil
		})

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
	err := app.Stop(context.Background())
	s.NoError(err)

	// Wait for Run to return
	select {
	case err := <-runErr:
		s.NoError(err)
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
	syscall.Kill(syscall.Getpid(), syscall.SIGTERM)

	// Wait for Run to return
	select {
	case err := <-runErr:
		s.NoError(err)
	case <-time.After(1 * time.Second):
		s.Fail("Run did not return after SIGTERM")
	}
}
