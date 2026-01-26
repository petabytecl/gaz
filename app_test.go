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
	c := NewContainer()

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
	c := NewContainer()
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
	case runResult := <-runErr:
		s.Require().NoError(runResult)
	case <-time.After(1 * time.Second):
		s.Fail("Run did not return after SIGTERM")
	}
}

func (s *AppTestSuite) TestWithShutdownTimeout() {
	c := NewContainer()
	timeout := 5 * time.Second
	app := NewApp(c, withShutdownTimeoutLegacy(timeout))

	s.Equal(timeout, app.opts.ShutdownTimeout, "shutdown timeout should be set")
}

func (s *AppTestSuite) TestRunAlreadyRunning() {
	c := NewContainer()
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
	c := NewContainer()
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
	c := NewContainer()
	app := NewApp(c)

	// Stop when not running should be no-op
	err := app.Stop(context.Background())
	s.Require().NoError(err)
}

type FailingStartService struct{}

func (s *AppTestSuite) TestRunStartError() {
	c := NewContainer()

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
	c := NewContainer()

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

// =============================================================================
// Tests for new fluent API (gaz.New())
// =============================================================================

func (s *AppTestSuite) TestNewCreatesAppWithDefaults() {
	app := New()

	s.NotNil(app.container, "container should be created")
	s.Equal(
		defaultShutdownTimeout,
		app.opts.ShutdownTimeout,
		"default shutdown timeout should be 30s",
	)
	s.False(app.built, "app should not be built initially")
}

func (s *AppTestSuite) TestNewWithShutdownTimeout() {
	timeout := 10 * time.Second
	app := New(WithShutdownTimeout(timeout))

	s.Equal(timeout, app.opts.ShutdownTimeout, "shutdown timeout should be set")
}

type FluentTestDB struct{ connected bool }

type FluentTestCache struct{ db *FluentTestDB }

type FluentTestRequest struct{ id int }

func (s *AppTestSuite) TestProvideSingletonRegistersService() {
	app := New()
	app.ProvideSingleton(func(_ *Container) (*FluentTestDB, error) {
		return &FluentTestDB{connected: true}, nil
	})

	s.Require().NoError(app.Build())

	db, err := Resolve[*FluentTestDB](app.Container())
	s.Require().NoError(err)
	s.True(db.connected, "db should be connected")

	// Singleton should return same instance
	db2, _ := Resolve[*FluentTestDB](app.Container())
	s.Same(db, db2, "singleton should return same instance")
}

func (s *AppTestSuite) TestProvideTransientReturnsNewInstances() {
	app := New()
	counter := 0
	app.ProvideTransient(func(_ *Container) (*FluentTestRequest, error) {
		counter++
		return &FluentTestRequest{id: counter}, nil
	})

	s.Require().NoError(app.Build())

	req1, err := Resolve[*FluentTestRequest](app.Container())
	s.Require().NoError(err)
	s.Equal(1, req1.id)

	req2, _ := Resolve[*FluentTestRequest](app.Container())
	s.Equal(2, req2.id)

	s.NotSame(req1, req2, "transient should return different instances")
}

func (s *AppTestSuite) TestProvideEagerInstantiatesAtBuild() {
	app := New()
	instantiated := false
	app.ProvideEager(func(_ *Container) (*FluentTestDB, error) {
		instantiated = true
		return &FluentTestDB{}, nil
	})

	s.False(instantiated, "should not instantiate before Build()")
	s.Require().NoError(app.Build())
	s.True(instantiated, "should instantiate at Build() time")
}

func (s *AppTestSuite) TestProvideInstance() {
	app := New()
	db := &FluentTestDB{connected: true}
	app.ProvideInstance(db)

	s.Require().NoError(app.Build())

	resolved, err := Resolve[*FluentTestDB](app.Container())
	s.Require().NoError(err)
	s.Same(db, resolved, "should return same instance")
}

func (s *AppTestSuite) TestFluentChaining() {
	app := New()
	app.ProvideSingleton(func(_ *Container) (*FluentTestDB, error) {
		return &FluentTestDB{connected: true}, nil
	}).
		ProvideSingleton(func(c *Container) (*FluentTestCache, error) {
			db, err := Resolve[*FluentTestDB](c)
			if err != nil {
				return nil, err
			}
			return &FluentTestCache{db: db}, nil
		})

	s.Require().NoError(app.Build())

	cache, err := Resolve[*FluentTestCache](app.Container())
	s.Require().NoError(err)
	s.NotNil(cache.db, "cache should have db dependency")
	s.True(cache.db.connected)
}

func (s *AppTestSuite) TestLateRegistrationPanics() {
	app := New()
	s.Require().NoError(app.Build())

	s.Panics(func() {
		app.ProvideSingleton(func(_ *Container) (*FluentTestDB, error) {
			return &FluentTestDB{}, nil
		})
	}, "should panic on late registration")

	s.Panics(func() {
		app.ProvideTransient(func(_ *Container) (*FluentTestDB, error) {
			return &FluentTestDB{}, nil
		})
	}, "should panic on late transient registration")

	s.Panics(func() {
		app.ProvideEager(func(_ *Container) (*FluentTestDB, error) {
			return &FluentTestDB{}, nil
		})
	}, "should panic on late eager registration")

	s.Panics(func() {
		app.ProvideInstance(&FluentTestDB{})
	}, "should panic on late instance registration")
}

func (s *AppTestSuite) TestBuildAggregatesErrors() {
	app := New()

	// Register same type twice - should collect errors
	app.ProvideSingleton(func(_ *Container) (*FluentTestDB, error) {
		return &FluentTestDB{}, nil
	})
	app.ProvideSingleton(func(_ *Container) (*FluentTestDB, error) {
		return &FluentTestDB{}, nil
	})

	err := app.Build()
	s.Require().Error(err, "should have error for duplicate registration")
	s.Contains(err.Error(), "FluentTestDB")
	s.Require().ErrorIs(err, ErrDuplicate)
}

func (s *AppTestSuite) TestBuildIsIdempotent() {
	app := New()
	app.ProvideSingleton(func(_ *Container) (*FluentTestDB, error) {
		return &FluentTestDB{connected: true}, nil
	})

	s.Require().NoError(app.Build())
	s.Require().NoError(app.Build()) // Should return nil on second call
	s.Require().NoError(app.Build()) // And third
}

func (s *AppTestSuite) TestInvalidProviderSignature() {
	app := New()

	// No arguments
	app.ProvideSingleton(func() (*FluentTestDB, error) {
		return &FluentTestDB{}, nil
	})

	err := app.Build()
	s.Require().Error(err)
	s.Require().ErrorIs(err, ErrInvalidProvider)

	// Wrong argument type
	app2 := New()
	app2.ProvideSingleton(func(_ int) (*FluentTestDB, error) {
		return &FluentTestDB{}, nil
	})

	err = app2.Build()
	s.Require().Error(err)
	s.Require().ErrorIs(err, ErrInvalidProvider)

	// Too many return values - test with a valid case instead
	app3 := New()
	app3.ProvideSingleton("not a function") // Invalid: not a function

	err = app3.Build()
	s.Require().Error(err)
	s.Require().ErrorIs(err, ErrInvalidProvider)
}

func (s *AppTestSuite) TestProviderWithoutError() {
	app := New()
	app.ProvideSingleton(func(_ *Container) *FluentTestDB {
		return &FluentTestDB{connected: true}
	})

	s.Require().NoError(app.Build())

	db, err := Resolve[*FluentTestDB](app.Container())
	s.Require().NoError(err)
	s.True(db.connected)
}

func (s *AppTestSuite) TestContainerAccessor() {
	app := New()
	container := app.Container()

	s.NotNil(container)
	s.Same(app.container, container)
}
