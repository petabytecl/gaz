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
	err := gaz.For[*testLifecycleService](app.Container()).
		OnStart(func(_ context.Context, _ *testLifecycleService) error {
			startCalled.Store(true)
			return nil
		}).
		OnStop(func(_ context.Context, _ *testLifecycleService) error {
			stopCalled.Store(true)
			return nil
		}).
		Eager(). // Must be eager to have hooks called
		ProviderFunc(func(_ *gaz.Container) *testLifecycleService {
			return &testLifecycleService{}
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

	// Register duplicate services using ProvideSingleton (which collects errors)
	app.ProvideSingleton(func(_ *gaz.Container) (*testDatabase, error) {
		return &testDatabase{}, nil
	})
	app.ProvideSingleton(func(_ *gaz.Container) (*testDatabase, error) {
		return &testDatabase{}, nil
	})

	// Duplicate module name
	app.Module("dup").Module("dup")

	err := app.Build()
	s.Require().Error(err)

	// Should contain both error types
	s.Require().ErrorIs(err, gaz.ErrDuplicate)
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

func (s *IntegrationSuite) TestFluentProviderMethodsChain() {
	// Test: All fluent provider methods on App return *App for chaining

	app := gaz.New()

	result := app.
		ProvideSingleton(func(_ *gaz.Container) (*testDatabase, error) {
			return &testDatabase{dsn: "singleton"}, nil
		}).
		ProvideTransient(func(_ *gaz.Container) (*testRequest, error) {
			return &testRequest{id: 1}, nil
		}).
		ProvideInstance(&testCache{addr: "redis://instance"}).
		Module("core",
			func(c *gaz.Container) error {
				return gaz.For[*testLogger](c).ProviderFunc(func(_ *gaz.Container) *testLogger {
					return &testLogger{level: "info"}
				})
			},
		)

	s.Same(app, result, "all methods should return same app for chaining")

	err := app.Build()
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
	app.Module("infrastructure",
		func(c *gaz.Container) error {
			return gaz.For[*testDatabase](c).
				OnStart(func(_ context.Context, _ *testDatabase) error {
					started.Store(true)
					return nil
				}).
				OnStop(func(_ context.Context, _ *testDatabase) error {
					stopped.Store(true)
					return nil
				}).
				Eager().
				ProviderFunc(func(_ *gaz.Container) *testDatabase {
					return &testDatabase{dsn: "production-db"}
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

// =============================================================================
// Test Helper Types
// =============================================================================

type testDatabase struct {
	dsn string
}

type testCache struct {
	addr string
}

type testUserService struct {
	db *testDatabase
}

type testLifecycleService struct{}

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
