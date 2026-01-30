package gaz_test

import (
	"context"
	"sync/atomic"
	"testing"
	"time"

	"github.com/spf13/cobra"
	"github.com/stretchr/testify/suite"

	"github.com/petabytecl/gaz"
)

// IntegrationSuite tests the complete Phase 3 feature set working together.
type IntegrationSuite struct {
	suite.Suite
}

func TestIntegrationSuite(t *testing.T) {
	suite.Run(t, new(IntegrationSuite))
}

// =============================================================================
// Complete Workflow Tests
// =============================================================================

func (s *IntegrationSuite) TestCompleteFluentWorkflow() {
	// Test: gaz.New() -> providers -> Build() -> Resolve
	// This exercises the complete workflow from Plan 01

	app := gaz.New(gaz.WithShutdownTimeout(5 * time.Second))

	// Register database service
	err := gaz.For[*testDatabase](app.Container()).
		Provider(func(_ *gaz.Container) (*testDatabase, error) {
			return &testDatabase{dsn: "postgres://localhost"}, nil
		})
	s.Require().NoError(err)

	// Register service that depends on database
	err = gaz.For[*testUserService](app.Container()).
		Provider(func(c *gaz.Container) (*testUserService, error) {
			db, resolveErr := gaz.Resolve[*testDatabase](c)
			if resolveErr != nil {
				return nil, resolveErr
			}
			return &testUserService{db: db}, nil
		})
	s.Require().NoError(err)

	// Build validates and wires
	err = app.Build()
	s.Require().NoError(err)

	// Resolve services
	userSvc, err := gaz.Resolve[*testUserService](app.Container())
	s.Require().NoError(err)
	s.NotNil(userSvc.db)
	s.Equal("postgres://localhost", userSvc.db.dsn)
}

func (s *IntegrationSuite) TestModulesWithFluentAPI() {
	// Test: Modules integrate with provider registration
	// This exercises the Module() method from Plan 02

	app := gaz.New()

	// Database module
	app.Module("database",
		func(c *gaz.Container) error {
			return gaz.For[*testDatabase](c).ProviderFunc(func(_ *gaz.Container) *testDatabase {
				return &testDatabase{dsn: "postgres://db"}
			})
		},
		func(c *gaz.Container) error {
			return gaz.For[*testCache](c).ProviderFunc(func(_ *gaz.Container) *testCache {
				return &testCache{addr: "redis://cache"}
			})
		},
	)

	// Service module that depends on database module
	app.Module("services",
		func(c *gaz.Container) error {
			return gaz.For[*testUserService](
				c,
			).Provider(func(c *gaz.Container) (*testUserService, error) {
				db, resolveErr := gaz.Resolve[*testDatabase](c)
				if resolveErr != nil {
					return nil, resolveErr
				}
				return &testUserService{db: db}, nil
			})
		},
	)

	err := app.Build()
	s.Require().NoError(err)

	// Verify cross-module dependencies work
	userSvc, err := gaz.Resolve[*testUserService](app.Container())
	s.Require().NoError(err)
	s.Equal("postgres://db", userSvc.db.dsn)
}

func (s *IntegrationSuite) TestCobraWithFullLifecycle() {
	// Test: Cobra integration with complete app lifecycle
	// This exercises WithCobra(), FromContext(), Start() from Plan 03

	var startCalled, stopCalled atomic.Bool

	app := gaz.New(gaz.WithShutdownTimeout(time.Second))

	// Register service with lifecycle hooks using For[T] API
	// Service implements di.Starter and di.Stopper interfaces - no fluent hooks needed
	err := gaz.For[*testLifecycleService](app.Container()).
		Eager(). // Must be eager to have hooks called
		ProviderFunc(func(_ *gaz.Container) *testLifecycleService {
			return &testLifecycleService{
				onStart: func() { startCalled.Store(true) },
				onStop:  func() { stopCalled.Store(true) },
			}
		})
	s.Require().NoError(err)

	var cmdExecuted bool

	rootCmd := &cobra.Command{
		Use: "test",
		RunE: func(cmd *cobra.Command, _ []string) error {
			cmdExecuted = true

			// Verify app is accessible
			gotApp := gaz.FromContext(cmd.Context())
			s.NotNil(gotApp)

			// Verify service was started
			s.True(startCalled.Load())

			return nil
		},
	}

	app.WithCobra(rootCmd)

	rootCmd.SetArgs([]string{})
	err = rootCmd.Execute()
	s.Require().NoError(err)

	s.True(cmdExecuted)
	s.True(startCalled.Load())
	s.True(stopCalled.Load()) // Stopped in PostRunE
}

// =============================================================================
// Error Handling Tests
// =============================================================================

func (s *IntegrationSuite) TestBuildAggregatesAllErrors() {
	app := gaz.New()

	// Register duplicate services using For[T] (which returns immediate error on duplicate)
	err := gaz.For[*testDatabase](app.Container()).ProviderFunc(func(_ *gaz.Container) *testDatabase {
		return &testDatabase{}
	})
	s.Require().NoError(err)

	err = gaz.For[*testDatabase](app.Container()).ProviderFunc(func(_ *gaz.Container) *testDatabase {
		return &testDatabase{}
	})
	s.Require().Error(err) // Duplicate error on registration
	s.Require().ErrorIs(err, gaz.ErrDuplicate)

	// Duplicate module name
	app.Module("dup").Module("dup")

	err = app.Build()
	s.Require().Error(err)

	// Should contain module duplicate error
	s.Require().ErrorIs(err, gaz.ErrDuplicateModule)
}

func (s *IntegrationSuite) TestMissingDependencyDetected() {
	app := gaz.New()

	// Service that depends on unregistered type (eager so Build tries to instantiate)
	_ = gaz.For[*testUserService](app.Container()).
		Eager().
		Provider(func(c *gaz.Container) (*testUserService, error) {
			db, resolveErr := gaz.Resolve[*testDatabase](c) // Not registered!
			if resolveErr != nil {
				return nil, resolveErr
			}
			return &testUserService{db: db}, nil
		})

	err := app.Build()
	s.Require().Error(err)
	s.ErrorIs(err, gaz.ErrNotFound)
}

func (s *IntegrationSuite) TestCyclicDependencyDetected() {
	app := gaz.New()

	// A depends on B, B depends on A
	_ = gaz.For[*serviceA](
		app.Container(),
	).Eager().
		Provider(func(c *gaz.Container) (*serviceA, error) {
			b, resolveErr := gaz.Resolve[*serviceB](c)
			if resolveErr != nil {
				return nil, resolveErr
			}
			return &serviceA{b: b}, nil
		})

	_ = gaz.For[*serviceB](app.Container()).Provider(func(c *gaz.Container) (*serviceB, error) {
		a, resolveErr := gaz.Resolve[*serviceA](c)
		if resolveErr != nil {
			return nil, resolveErr
		}
		return &serviceB{a: a}, nil
	})

	err := app.Build()
	s.Require().Error(err)
	s.ErrorIs(err, gaz.ErrCycle)
}

func (s *IntegrationSuite) TestModuleRegistrationError() {
	app := gaz.New()

	// Module with failing registration
	app.Module("failing",
		func(c *gaz.Container) error {
			// Register once
			_ = gaz.For[*testDatabase](c).ProviderFunc(func(_ *gaz.Container) *testDatabase {
				return &testDatabase{}
			})
			// Register again - should fail
			return gaz.For[*testDatabase](c).ProviderFunc(func(_ *gaz.Container) *testDatabase {
				return &testDatabase{}
			})
		},
	)

	err := app.Build()
	s.Require().Error(err)
	s.Contains(err.Error(), "failing") // Module name should be in error
	s.ErrorIs(err, gaz.ErrDuplicate)
}

// =============================================================================
// Edge Cases
// =============================================================================

func (s *IntegrationSuite) TestEmptyAppBuildsSuccessfully() {
	app := gaz.New()
	err := app.Build()
	s.Require().NoError(err)
}

func (s *IntegrationSuite) TestBuildIsIdempotent() {
	app := gaz.New()

	_ = gaz.For[*testDatabase](app.Container()).ProviderFunc(func(_ *gaz.Container) *testDatabase {
		return &testDatabase{}
	})

	err1 := app.Build()
	s.Require().NoError(err1)

	err2 := app.Build() // Second call
	s.Require().NoError(err2)

	err3 := app.Build() // Third call
	s.Require().NoError(err3)
}

func (s *IntegrationSuite) TestNestedModuleDependencies() {
	// Test: Services in module B can depend on services in module A

	app := gaz.New()

	app.Module("infrastructure",
		func(c *gaz.Container) error {
			return gaz.For[*testDatabase](c).ProviderFunc(func(_ *gaz.Container) *testDatabase {
				return &testDatabase{dsn: "infra-db"}
			})
		},
	).Module("domain",
		func(c *gaz.Container) error {
			return gaz.For[*testUserService](
				c,
			).Provider(func(c *gaz.Container) (*testUserService, error) {
				// Depend on service from infrastructure module
				db, resolveErr := gaz.Resolve[*testDatabase](c)
				if resolveErr != nil {
					return nil, resolveErr
				}
				return &testUserService{db: db}, nil
			})
		},
	)

	err := app.Build()
	s.Require().NoError(err)

	userSvc, err := gaz.Resolve[*testUserService](app.Container())
	s.Require().NoError(err)
	s.Equal("infra-db", userSvc.db.dsn)
}

func (s *IntegrationSuite) TestCobraSubcommandHierarchy() {
	// Test: Nested subcommands all have access to App

	app := gaz.New()
	_ = gaz.For[*testDatabase](app.Container()).ProviderFunc(func(_ *gaz.Container) *testDatabase {
		return &testDatabase{dsn: "test-db"}
	})

	var level1App, level2App *gaz.App

	rootCmd := &cobra.Command{Use: "root"}
	level1Cmd := &cobra.Command{
		Use: "level1",
		RunE: func(cmd *cobra.Command, _ []string) error {
			level1App = gaz.FromContext(cmd.Context())
			return nil
		},
	}
	level2Cmd := &cobra.Command{
		Use: "level2",
		RunE: func(cmd *cobra.Command, _ []string) error {
			level2App = gaz.FromContext(cmd.Context())
			return nil
		},
	}

	level1Cmd.AddCommand(level2Cmd)
	rootCmd.AddCommand(level1Cmd)

	app.WithCobra(rootCmd)

	// Execute level1 command first
	rootCmd.SetArgs([]string{"level1"})
	err := rootCmd.Execute()
	s.Require().NoError(err)
	s.Same(app, level1App)

	// Execute nested command
	rootCmd.SetArgs([]string{"level1", "level2"})
	err = rootCmd.Execute()
	s.Require().NoError(err)
	s.Same(app, level2App)
}

func (s *IntegrationSuite) TestEmptyModulesAreValid() {
	// Test: Empty modules are valid (from Plan 02 decision)

	app := gaz.New()
	app.Module("empty") // No registrations

	err := app.Build()
	s.Require().NoError(err)
}

func (s *IntegrationSuite) TestFluentForTRegistration() {
	// Test: For[T]() pattern for service registration

	app := gaz.New()

	// Register singleton using For[T]()
	err := gaz.For[*testDatabase](app.Container()).Provider(func(_ *gaz.Container) (*testDatabase, error) {
		return &testDatabase{dsn: "singleton"}, nil
	})
	s.Require().NoError(err)

	// Register transient using For[T]().Transient()
	err = gaz.For[*testRequest](app.Container()).Transient().Provider(func(_ *gaz.Container) (*testRequest, error) {
		return &testRequest{id: 1}, nil
	})
	s.Require().NoError(err)

	// Register instance using For[T]().Instance()
	err = gaz.For[*testCache](app.Container()).Instance(&testCache{addr: "redis://instance"})
	s.Require().NoError(err)

	// Register via module (still uses For[T] pattern inside)
	app.Module("core",
		func(c *gaz.Container) error {
			return gaz.For[*testLogger](c).ProviderFunc(func(_ *gaz.Container) *testLogger {
				return &testLogger{level: "info"}
			})
		},
	)

	err = app.Build()
	s.Require().NoError(err)

	// Verify all services are registered
	db, err := gaz.Resolve[*testDatabase](app.Container())
	s.Require().NoError(err)
	s.Equal("singleton", db.dsn)

	cache, err := gaz.Resolve[*testCache](app.Container())
	s.Require().NoError(err)
	s.Equal("redis://instance", cache.addr)

	logger, err := gaz.Resolve[*testLogger](app.Container())
	s.Require().NoError(err)
	s.Equal("info", logger.level)
}

func (s *IntegrationSuite) TestCobraWithModulesAndLifecycle() {
	// Test: Full integration - modules + Cobra + lifecycle hooks

	var started, stopped atomic.Bool

	app := gaz.New()

	// Infrastructure module with lifecycle
	// Service implements di.Starter and di.Stopper interfaces - no fluent hooks needed
	app.Module("infrastructure",
		func(c *gaz.Container) error {
			return gaz.For[*testDatabase](c).
				Eager().
				ProviderFunc(func(_ *gaz.Container) *testDatabase {
					return &testDatabase{
						dsn:     "production-db",
						onStart: func() { started.Store(true) },
						onStop:  func() { stopped.Store(true) },
					}
				})
		},
	)

	// Application module depending on infrastructure
	app.Module("application",
		func(c *gaz.Container) error {
			return gaz.For[*testUserService](
				c,
			).Provider(func(c *gaz.Container) (*testUserService, error) {
				db, resolveErr := gaz.Resolve[*testDatabase](c)
				if resolveErr != nil {
					return nil, resolveErr
				}
				return &testUserService{db: db}, nil
			})
		},
	)

	var dbDSN string
	rootCmd := &cobra.Command{
		Use: "app",
		RunE: func(cmd *cobra.Command, _ []string) error {
			gotApp := gaz.FromContext(cmd.Context())
			userSvc, resolveErr := gaz.Resolve[*testUserService](gotApp.Container())
			if resolveErr != nil {
				return resolveErr
			}
			dbDSN = userSvc.db.dsn
			return nil
		},
	}

	app.WithCobra(rootCmd)

	rootCmd.SetArgs([]string{})
	err := rootCmd.Execute()
	s.Require().NoError(err)

	s.True(started.Load(), "database OnStart should be called")
	s.True(stopped.Load(), "database OnStop should be called")
	s.Equal("production-db", dbDSN)
}

func (s *IntegrationSuite) TestCobraConfigIntegration() {
	// Test: Cobra flags map to config via WithConfig and WithCobra

	type AppConfig struct {
		Port int
		Name string
	}

	var cfg AppConfig
	app := gaz.New().WithConfig(&cfg)

	rootCmd := &cobra.Command{
		Use: "app",
		RunE: func(_ *cobra.Command, _ []string) error {
			return nil
		},
	}

	rootCmd.Flags().Int("port", 8080, "port number")
	rootCmd.Flags().String("name", "default", "app name")

	app.WithCobra(rootCmd)

	// Bind flags override default
	rootCmd.SetArgs([]string{"--port", "9090", "--name", "override"})
	err := rootCmd.Execute()
	s.Require().NoError(err)

	s.Equal(9090, cfg.Port)
	s.Equal("override", cfg.Name)
}

// =============================================================================
// Test Helper Types
// =============================================================================

type testDatabase struct {
	dsn     string
	onStart func()
	onStop  func()
}

// OnStart implements di.Starter for testDatabase.
func (d *testDatabase) OnStart(_ context.Context) error {
	if d.onStart != nil {
		d.onStart()
	}
	return nil
}

// OnStop implements di.Stopper for testDatabase.
func (d *testDatabase) OnStop(_ context.Context) error {
	if d.onStop != nil {
		d.onStop()
	}
	return nil
}

type testCache struct {
	addr string
}

type testUserService struct {
	db *testDatabase
}

type testLifecycleService struct {
	onStart func()
	onStop  func()
}

// OnStart implements di.Starter for testLifecycleService.
func (s *testLifecycleService) OnStart(_ context.Context) error {
	if s.onStart != nil {
		s.onStart()
	}
	return nil
}

// OnStop implements di.Stopper for testLifecycleService.
func (s *testLifecycleService) OnStop(_ context.Context) error {
	if s.onStop != nil {
		s.onStop()
	}
	return nil
}

type testRequest struct {
	id int
}

type testLogger struct {
	level string
}

type serviceA struct {
	b *serviceB
}

type serviceB struct {
	a *serviceA
}
