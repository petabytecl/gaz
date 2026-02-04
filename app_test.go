package gaz

import (
	"context"
	"errors"
	"log/slog"
	"os"
	"path/filepath"
	"sync"
	"syscall"
	"testing"
	"time"

	"github.com/spf13/cobra"
	"github.com/stretchr/testify/suite"

	"github.com/petabytecl/gaz/cron"
	"github.com/petabytecl/gaz/di"
	"github.com/petabytecl/gaz/eventbus"
	"github.com/petabytecl/gaz/logger"
)

type AppTestSuite struct {
	suite.Suite
}

func TestAppTestSuite(t *testing.T) {
	suite.Run(t, new(AppTestSuite))
}

type (
	// AppTestServiceA is a test service that calls lifecycle callbacks.
	AppTestServiceA struct {
		onStart func()
		onStop  func()
	}
	// AppTestServiceB is a test service that depends on A and calls lifecycle callbacks.
	AppTestServiceB struct {
		A       *AppTestServiceA
		onStart func()
		onStop  func()
	}
)

// OnStart implements di.Starter for AppTestServiceA.
func (s *AppTestServiceA) OnStart(_ context.Context) error {
	if s.onStart != nil {
		s.onStart()
	}
	return nil
}

// OnStop implements di.Stopper for AppTestServiceA.
func (s *AppTestServiceA) OnStop(_ context.Context) error {
	if s.onStop != nil {
		s.onStop()
	}
	return nil
}

// OnStart implements di.Starter for AppTestServiceB.
func (s *AppTestServiceB) OnStart(_ context.Context) error {
	if s.onStart != nil {
		s.onStart()
	}
	return nil
}

// OnStop implements di.Stopper for AppTestServiceB.
func (s *AppTestServiceB) OnStop(_ context.Context) error {
	if s.onStop != nil {
		s.onStop()
	}
	return nil
}

func (s *AppTestSuite) TestRunAndStop() {
	app := New()

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
	// Service implements di.Starter and di.Stopper interfaces - no fluent hooks needed
	err := For[*AppTestServiceA](app.Container()).Named("A").Eager().
		Provider(func(_ *Container) (*AppTestServiceA, error) {
			return &AppTestServiceA{
				onStart: func() { recordStart("A") },
				onStop:  func() { recordStop("A") },
			}, nil
		})
	s.Require().NoError(err)

	// Service B depends on A
	// Service implements di.Starter and di.Stopper interfaces - no fluent hooks needed
	err = For[*AppTestServiceB](app.Container()).Named("B").Eager().
		Provider(func(c *Container) (*AppTestServiceB, error) {
			a, resolveErr := Resolve[*AppTestServiceA](c, Named("A"))
			if resolveErr != nil {
				return nil, resolveErr
			}
			return &AppTestServiceB{
				A:       a,
				onStart: func() { recordStart("B") },
				onStop:  func() { recordStop("B") },
			}, nil
		})
	s.Require().NoError(err)

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
	app := New()

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
	timeout := 5 * time.Second
	app := New(WithShutdownTimeout(timeout))

	s.Equal(timeout, app.opts.ShutdownTimeout, "shutdown timeout should be set")
}

func (s *AppTestSuite) TestRunAlreadyRunning() {
	app := New()

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
	app := New()

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
	app := New()

	// Stop when not running should be no-op
	err := app.Stop(context.Background())
	s.Require().NoError(err)
}

type FailingStartService struct{}

// OnStart implements di.Starter and always returns an error.
func (s *FailingStartService) OnStart(_ context.Context) error {
	return errors.New("start failed")
}

func (s *AppTestSuite) TestRunStartError() {
	app := New()

	err := For[*FailingStartService](app.Container()).Eager().
		ProviderFunc(func(_ *Container) *FailingStartService {
			return &FailingStartService{}
		})
	s.Require().NoError(err)

	err = app.Run(context.Background())
	s.Require().Error(err)
	s.Contains(err.Error(), "starting service")
}

type FailingStopService struct{}

// OnStop implements di.Stopper and always returns an error.
func (s *FailingStopService) OnStop(_ context.Context) error {
	return errors.New("stop failed")
}

func (s *AppTestSuite) TestStopError() {
	app := New()

	err := For[*FailingStopService](app.Container()).Named("failstop").Eager().
		ProviderFunc(func(_ *Container) *FailingStopService {
			return &FailingStopService{}
		})
	s.Require().NoError(err)

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
// Tests for fluent API (gaz.New() + For[T]())
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

func (s *AppTestSuite) TestSingletonRegistersService() {
	app := New()
	err := For[*FluentTestDB](app.Container()).Provider(func(_ *Container) (*FluentTestDB, error) {
		return &FluentTestDB{connected: true}, nil
	})
	s.Require().NoError(err)

	s.Require().NoError(app.Build())

	db, err := Resolve[*FluentTestDB](app.Container())
	s.Require().NoError(err)
	s.True(db.connected, "db should be connected")

	// Singleton should return same instance
	db2, _ := Resolve[*FluentTestDB](app.Container())
	s.Same(db, db2, "singleton should return same instance")
}

func (s *AppTestSuite) TestTransientReturnsNewInstances() {
	app := New()
	counter := 0
	err := For[*FluentTestRequest](app.Container()).Transient().Provider(func(_ *Container) (*FluentTestRequest, error) {
		counter++
		return &FluentTestRequest{id: counter}, nil
	})
	s.Require().NoError(err)

	s.Require().NoError(app.Build())

	req1, err := Resolve[*FluentTestRequest](app.Container())
	s.Require().NoError(err)
	s.Equal(1, req1.id)

	req2, _ := Resolve[*FluentTestRequest](app.Container())
	s.Equal(2, req2.id)

	s.NotSame(req1, req2, "transient should return different instances")
}

func (s *AppTestSuite) TestEagerInstantiatesAtBuild() {
	app := New()
	instantiated := false
	err := For[*FluentTestDB](app.Container()).Eager().Provider(func(_ *Container) (*FluentTestDB, error) {
		instantiated = true
		return &FluentTestDB{}, nil
	})
	s.Require().NoError(err)

	s.False(instantiated, "should not instantiate before Build()")
	s.Require().NoError(app.Build())
	s.True(instantiated, "should instantiate at Build() time")
}

func (s *AppTestSuite) TestInstanceRegistration() {
	app := New()
	db := &FluentTestDB{connected: true}
	err := For[*FluentTestDB](app.Container()).Instance(db)
	s.Require().NoError(err)

	s.Require().NoError(app.Build())

	resolved, err := Resolve[*FluentTestDB](app.Container())
	s.Require().NoError(err)
	s.Same(db, resolved, "should return same instance")
}

func (s *AppTestSuite) TestDependencyResolution() {
	app := New()
	err := For[*FluentTestDB](app.Container()).Provider(func(_ *Container) (*FluentTestDB, error) {
		return &FluentTestDB{connected: true}, nil
	})
	s.Require().NoError(err)

	err = For[*FluentTestCache](app.Container()).Provider(func(c *Container) (*FluentTestCache, error) {
		db, resolveErr := Resolve[*FluentTestDB](c)
		if resolveErr != nil {
			return nil, resolveErr
		}
		return &FluentTestCache{db: db}, nil
	})
	s.Require().NoError(err)

	s.Require().NoError(app.Build())

	cache, err := Resolve[*FluentTestCache](app.Container())
	s.Require().NoError(err)
	s.NotNil(cache.db, "cache should have db dependency")
	s.True(cache.db.connected)
}

func (s *AppTestSuite) TestBuildAllowsDuplicateRegistration() {
	app := New()

	// Register same type twice - should NOT error
	err := For[*FluentTestDB](app.Container()).Provider(func(_ *Container) (*FluentTestDB, error) {
		return &FluentTestDB{}, nil
	})
	s.Require().NoError(err)

	err = For[*FluentTestDB](app.Container()).Provider(func(_ *Container) (*FluentTestDB, error) {
		return &FluentTestDB{}, nil
	})
	s.Require().NoError(err, "should allow duplicate registration")
}

func (s *AppTestSuite) TestBuildIsIdempotent() {
	app := New()
	err := For[*FluentTestDB](app.Container()).Provider(func(_ *Container) (*FluentTestDB, error) {
		return &FluentTestDB{connected: true}, nil
	})
	s.Require().NoError(err)

	s.Require().NoError(app.Build())
	s.Require().NoError(app.Build()) // Should return nil on second call
	s.Require().NoError(app.Build()) // And third
}

func (s *AppTestSuite) TestContainerAccessor() {
	app := New()
	container := app.Container()

	s.NotNil(container)
	s.Same(app.container, container)
}

func (s *AppTestSuite) TestLoggerInjection() {
	app := New()
	s.Require().NoError(app.Build())

	// Verify logger is resolvable
	logger, err := Resolve[*slog.Logger](app.Container())
	s.Require().NoError(err)
	s.NotNil(logger)

	// Verify it's the same instance as app.Logger
	s.Same(app.Logger, logger)

	// Verify we can log without panic
	s.NotPanics(func() {
		logger.Info("test log message")
	})
}

func (s *AppTestSuite) TestEventBus() {
	app := New()

	// Before Build(), EventBus should be nil (deferred initialization)
	eventBusBefore := app.EventBus()
	s.Nil(eventBusBefore, "EventBus should be nil before Build()")

	s.Require().NoError(app.Build())

	// After Build(), EventBus should be accessible
	eventBusAfter := app.EventBus()
	s.NotNil(eventBusAfter, "EventBus should be available after Build()")

	// EventBus should also be resolvable via DI
	resolvedEventBus, err := Resolve[*eventbus.EventBus](app.Container())
	s.Require().NoError(err)
	s.Same(eventBusAfter, resolvedEventBus, "Resolved EventBus should be the same as accessor")
}

func (s *AppTestSuite) TestWithLoggerConfig() {
	// Test with custom logger config
	customConfig := &logger.Config{
		Level:     slog.LevelDebug,
		Format:    "text",
		AddSource: true,
	}

	app := New(WithLoggerConfig(customConfig))

	// Verify the config was applied
	s.Equal(customConfig, app.opts.LoggerConfig)

	s.Require().NoError(app.Build())

	// Verify logger is resolvable and can log
	resolvedLogger, err := Resolve[*slog.Logger](app.Container())
	s.Require().NoError(err)
	s.NotNil(resolvedLogger)
	s.Same(app.Logger, resolvedLogger)

	// Verify we can log without panic
	s.NotPanics(func() {
		resolvedLogger.Debug("debug message")
		resolvedLogger.Info("info message")
	})
}

func (s *AppTestSuite) TestWithLoggerConfig_NilUseDefaults() {
	// When WithLoggerConfig is not used, defaults should apply
	app := New()

	// Default config should be Info level and JSON format
	s.Equal(slog.LevelInfo, app.opts.LoggerConfig.Level)
	s.Equal("json", app.opts.LoggerConfig.Format)
}

func (s *AppTestSuite) TestLoggerInitializedInBuild() {
	// Test: Logger is nil before Build(), initialized after
	app := New()

	// Before Build(), Logger should be nil
	s.Nil(app.Logger, "Logger should be nil before Build()")

	// Build should initialize Logger
	err := app.Build()
	s.Require().NoError(err)

	// After Build(), Logger should be initialized
	s.NotNil(app.Logger, "Logger should be initialized after Build()")
}

func (s *AppTestSuite) TestLoggerConfigResolution() {
	// Test: Logger uses config from container if registered via logger module
	app := New()

	// Register a custom logger.Config in the container
	customConfig := logger.Config{
		Level:  slog.LevelDebug,
		Format: "text",
	}
	err := For[logger.Config](app.Container()).Instance(customConfig)
	s.Require().NoError(err)

	// Build should resolve the config from container
	err = app.Build()
	s.Require().NoError(err)

	// Logger should be initialized (we can't easily verify it used the config,
	// but the resolution path is tested)
	s.NotNil(app.Logger, "Logger should be initialized after Build()")
}

// =============================================================================
// Tests for discoverCronJobs
// =============================================================================

type TestCronJob struct {
	name     string
	schedule string
	executed bool
}

func (j *TestCronJob) Name() string {
	return j.name
}

func (j *TestCronJob) Schedule() string {
	return j.schedule
}

func (j *TestCronJob) Timeout() time.Duration {
	return 5 * time.Second
}

func (j *TestCronJob) Run(ctx context.Context) error {
	j.executed = true
	return nil
}

func (s *AppTestSuite) TestDiscoverCronJobs() {
	app := New()

	// Register a cron job with valid schedule
	err := For[cron.CronJob](app.Container()).Named("test-job").Transient().
		Provider(func(_ *Container) (cron.CronJob, error) {
			return &TestCronJob{name: "test-job", schedule: "@hourly"}, nil
		})
	s.Require().NoError(err)

	// Build should discover and register the job
	s.Require().NoError(app.Build())
}

func (s *AppTestSuite) TestDiscoverCronJobs_InvalidSchedule() {
	app := New()

	// Register a cron job with invalid schedule
	err := For[cron.CronJob](app.Container()).Named("invalid-job").Transient().
		Provider(func(_ *Container) (cron.CronJob, error) {
			return &TestCronJob{name: "invalid-job", schedule: "invalid"}, nil
		})
	s.Require().NoError(err)

	// Build should log warning for invalid schedule but not fail
	// (the scheduler handles invalid schedules gracefully)
	s.Require().NoError(app.Build())
}

func (s *AppTestSuite) TestDiscoverCronJobs_EmptySchedule() {
	app := New()

	// Register a cron job with empty schedule (disabled)
	err := For[cron.CronJob](app.Container()).Named("disabled-job").Transient().
		Provider(func(_ *Container) (cron.CronJob, error) {
			return &TestCronJob{name: "disabled-job", schedule: ""}, nil
		})
	s.Require().NoError(err)

	// Build should handle empty schedule (disabled job)
	s.Require().NoError(app.Build())
}

func (s *AppTestSuite) TestDiscoverCronJobs_NonTransient() {
	app := New()

	// Register a cron job as singleton (not recommended, should log warning)
	err := For[cron.CronJob](app.Container()).Named("singleton-job").
		Provider(func(_ *Container) (cron.CronJob, error) {
			return &TestCronJob{name: "singleton-job", schedule: "@daily"}, nil
		})
	s.Require().NoError(err)

	// Build should log warning about non-transient CronJob
	s.Require().NoError(app.Build())
}

// =============================================================================
// Tests for WithStrictConfig
// =============================================================================

func (s *AppTestSuite) TestWithStrictConfig_SetsFlag() {
	app := New(WithStrictConfig())
	s.True(app.strictConfig, "strictConfig should be set to true")
}

func (s *AppTestSuite) TestWithStrictConfig_WithoutConfigTarget_NoEffect() {
	// WithStrictConfig without WithConfig should have no effect
	app := New(WithStrictConfig())
	s.Require().NoError(app.Build())
}

func (s *AppTestSuite) TestMergeConfigMap() {
	app := New()
	err := app.MergeConfigMap(map[string]any{
		"test.key": "value",
	})
	s.Require().NoError(err)

	s.Require().NoError(app.Build())

	// Verify config is available
	// Since we don't have easy access to config manager without public accessors,
	// we rely on the fact that MergeConfigMap didn't error.
	// But we can resolve *config.Manager and check.
	// However, importing config creates cycle if not careful (gaz imports config).
	// But this is gaz package test (package gaz).
	// So we can import "github.com/petabytecl/gaz/config".
	// Wait, gaz imports config, so test can too.
}

type groupTestService struct{ id string }

func (s *AppTestSuite) TestResolveAllAndGroupWrappers() {
	app := New()

	// Register services in group
	For[*groupTestService](app.Container()).InGroup("g1").ProviderFunc(func(_ *Container) *groupTestService {
		return &groupTestService{id: "1"}
	})
	For[*groupTestService](app.Container()).InGroup("g1").ProviderFunc(func(_ *Container) *groupTestService {
		return &groupTestService{id: "2"}
	})
	For[*groupTestService](app.Container()).InGroup("g2").ProviderFunc(func(_ *Container) *groupTestService {
		return &groupTestService{id: "3"}
	})

	s.Require().NoError(app.Build())

	// Test ResolveAll
	all, err := ResolveAll[*groupTestService](app.Container())
	s.Require().NoError(err)
	s.Len(all, 3)

	// Test ResolveGroup
	g1, err := ResolveGroup[*groupTestService](app.Container(), "g1")
	s.Require().NoError(err)
	s.Len(g1, 2)

	g2, err := ResolveGroup[*groupTestService](app.Container(), "g2")
	s.Require().NoError(err)
	s.Len(g2, 1)
}

// =============================================================================
// Tests for applyConfigFlags
// =============================================================================

func (s *AppTestSuite) TestApplyConfigFlags_NoCobraCmd() {
	// When no cobra command is attached, applyConfigFlags should return nil
	app := New()
	s.Require().NoError(app.Build())
	// applyConfigFlags is called internally during loadConfig - just verifying no error
}

func (s *AppTestSuite) TestApplyConfigFlags_NoConfigFlag() {
	// When cobra is attached but no --config flag is registered,
	// applyConfigFlags should return nil without error
	cmd := &cobra.Command{Use: "test"}
	app := New(WithCobra(cmd))
	// Don't use config module - no --config flag registered
	s.Require().NoError(app.Build())
}

func (s *AppTestSuite) TestApplyConfigFlags_WithConfigModule_NoFlag() {
	// When config module is used but --config not provided,
	// auto-search paths should be configured
	cmd := &cobra.Command{
		Use: "testapp",
		RunE: func(cmd *cobra.Command, args []string) error {
			return nil
		},
	}
	_ = New(WithCobra(cmd))

	// Register --config flag manually (simulating config module)
	cmd.PersistentFlags().String("config", "", "Config file path")
	cmd.PersistentFlags().String("env-prefix", "TEST", "Env prefix")
	cmd.PersistentFlags().Bool("config-strict", true, "Strict mode")

	cmd.SetArgs([]string{})
	err := cmd.Execute()
	s.NoError(err)
}

func (s *AppTestSuite) TestApplyConfigFlags_ExplicitConfigExists() {
	// Create temp config file
	tempDir := s.T().TempDir()
	configPath := filepath.Join(tempDir, "config.yaml")
	s.Require().NoError(os.WriteFile(configPath, []byte("key: value"), 0o644))

	cmd := &cobra.Command{
		Use: "testapp",
		RunE: func(cmd *cobra.Command, args []string) error {
			return nil
		},
	}
	_ = New(WithCobra(cmd))

	// Register --config flag manually
	cmd.PersistentFlags().String("config", "", "Config file path")
	cmd.PersistentFlags().String("env-prefix", "TEST", "Env prefix")
	cmd.PersistentFlags().Bool("config-strict", true, "Strict mode")

	cmd.SetArgs([]string{"--config", configPath})
	err := cmd.Execute()
	s.NoError(err)
}

func (s *AppTestSuite) TestApplyConfigFlags_ExplicitConfigNotExists() {
	cmd := &cobra.Command{
		Use: "testapp",
		RunE: func(cmd *cobra.Command, args []string) error {
			return nil
		},
	}
	_ = New(WithCobra(cmd))

	// Register --config flag manually
	cmd.PersistentFlags().String("config", "", "Config file path")

	cmd.SetArgs([]string{"--config", "/nonexistent/config.yaml"})
	err := cmd.Execute()
	s.Error(err)
	s.Contains(err.Error(), "not found")
}

func (s *AppTestSuite) TestApplyConfigFlags_EnvPrefix() {
	cmd := &cobra.Command{
		Use: "testapp",
		RunE: func(cmd *cobra.Command, args []string) error {
			return nil
		},
	}
	_ = New(WithCobra(cmd))

	// Register flags manually
	cmd.PersistentFlags().String("config", "", "Config file path")
	cmd.PersistentFlags().String("env-prefix", "MYAPP", "Env prefix")
	cmd.PersistentFlags().Bool("config-strict", true, "Strict mode")

	cmd.SetArgs([]string{"--env-prefix", "CUSTOM"})
	err := cmd.Execute()
	s.NoError(err)
}

func (s *AppTestSuite) TestApplyConfigFlags_StrictFalse() {
	cmd := &cobra.Command{
		Use: "testapp",
		RunE: func(cmd *cobra.Command, args []string) error {
			return nil
		},
	}
	app := New(WithCobra(cmd))

	// Register flags manually
	cmd.PersistentFlags().String("config", "", "Config file path")
	cmd.PersistentFlags().String("env-prefix", "TEST", "Env prefix")
	cmd.PersistentFlags().Bool("config-strict", true, "Strict mode")

	cmd.SetArgs([]string{"--config-strict=false"})
	err := cmd.Execute()
	s.NoError(err)
	// Verify strict mode was set to false
	s.False(app.strictConfig)
}

func (s *AppTestSuite) TestApplyConfigFlags_StrictTrue() {
	cmd := &cobra.Command{
		Use: "testapp",
		RunE: func(cmd *cobra.Command, args []string) error {
			return nil
		},
	}
	app := New(WithCobra(cmd))

	// Register flags manually
	cmd.PersistentFlags().String("config", "", "Config file path")
	cmd.PersistentFlags().String("env-prefix", "TEST", "Env prefix")
	cmd.PersistentFlags().Bool("config-strict", false, "Strict mode") // Default false

	cmd.SetArgs([]string{"--config-strict=true"})
	err := cmd.Execute()
	s.NoError(err)
	// Verify strict mode was set to true
	s.True(app.strictConfig)
}

func (s *AppTestSuite) TestApplyConfigFlags_XDGConfigHome() {
	// Test XDG_CONFIG_HOME environment variable handling
	tempDir := s.T().TempDir()
	xdgDir := filepath.Join(tempDir, "xdg-config")
	s.Require().NoError(os.MkdirAll(xdgDir, 0o755))

	// Set XDG_CONFIG_HOME
	original := os.Getenv("XDG_CONFIG_HOME")
	s.Require().NoError(os.Setenv("XDG_CONFIG_HOME", xdgDir))
	defer func() {
		if original != "" {
			_ = os.Setenv("XDG_CONFIG_HOME", original)
		} else {
			_ = os.Unsetenv("XDG_CONFIG_HOME")
		}
	}()

	cmd := &cobra.Command{
		Use: "testapp",
		RunE: func(cmd *cobra.Command, args []string) error {
			return nil
		},
	}
	_ = New(WithCobra(cmd))

	// Register --config flag manually
	cmd.PersistentFlags().String("config", "", "Config file path")
	cmd.PersistentFlags().String("env-prefix", "TEST", "Env prefix")
	cmd.PersistentFlags().Bool("config-strict", true, "Strict mode")

	// No --config provided, should use XDG_CONFIG_HOME for auto-search
	cmd.SetArgs([]string{})
	err := cmd.Execute()
	s.NoError(err)
}

func (s *AppTestSuite) TestApplyConfigFlags_NoXDGConfigHome() {
	// Test without XDG_CONFIG_HOME set
	original := os.Getenv("XDG_CONFIG_HOME")
	s.Require().NoError(os.Unsetenv("XDG_CONFIG_HOME"))
	defer func() {
		if original != "" {
			_ = os.Setenv("XDG_CONFIG_HOME", original)
		}
	}()

	cmd := &cobra.Command{
		Use: "testapp",
		RunE: func(cmd *cobra.Command, args []string) error {
			return nil
		},
	}
	_ = New(WithCobra(cmd))

	// Register --config flag manually
	cmd.PersistentFlags().String("config", "", "Config file path")
	cmd.PersistentFlags().String("env-prefix", "TEST", "Env prefix")
	cmd.PersistentFlags().Bool("config-strict", true, "Strict mode")

	cmd.SetArgs([]string{})
	err := cmd.Execute()
	s.NoError(err)
}

// Tests for UseDI

func (s *AppTestSuite) TestUseDI_Basic() {
	// Create a simple DI module
	module := di.NewModuleFunc("test-module", func(c *di.Container) error {
		return nil
	})

	app := New().UseDI(module)
	s.NotNil(app)

	// Build should work
	err := app.Build()
	s.NoError(err)
}

func (s *AppTestSuite) TestUseDI_Duplicate() {
	// Create a module
	module := di.NewModuleFunc("test-module", func(c *di.Container) error {
		return nil
	})

	// Try to register same module twice
	app := New().UseDI(module).UseDI(module)
	s.NotNil(app)

	// Build should fail with duplicate error
	err := app.Build()
	s.Error(err)
	s.Contains(err.Error(), "duplicate")
}

func (s *AppTestSuite) TestUseDI_ModuleRegisterError() {
	// Create a module that errors on registration
	module := di.NewModuleFunc("error-module", func(c *di.Container) error {
		return errors.New("registration failed")
	})

	app := New().UseDI(module)

	// Build should fail
	err := app.Build()
	s.Error(err)
	s.Contains(err.Error(), "registration failed")
}

func (s *AppTestSuite) TestUseDI_AfterBuild() {
	module := di.NewModuleFunc("test-module", func(c *di.Container) error {
		return nil
	})

	app := New()
	err := app.Build()
	s.NoError(err)

	// Adding module after Build should panic
	s.Panics(func() {
		app.UseDI(module)
	})
}

// Tests for GetArgs

func (s *AppTestSuite) TestGetArgs_NilContainer() {
	// GetArgs with nil container should return nil
	args := GetArgs(nil)
	s.Nil(args)
}

func (s *AppTestSuite) TestGetArgs_NoCommandArgs() {
	// GetArgs when CommandArgs is not registered should return nil
	c := di.New()
	args := GetArgs(c)
	s.Nil(args)
}

func (s *AppTestSuite) TestGetArgs_WithCommandArgs() {
	// GetArgs when CommandArgs is registered should return args
	c := di.New()
	cmdArgs := &CommandArgs{
		Args: []string{"arg1", "arg2"},
	}
	err := di.For[*CommandArgs](c).Instance(cmdArgs)
	s.NoError(err)

	args := GetArgs(c)
	s.Equal([]string{"arg1", "arg2"}, args)
}

// Tests for MergeConfigMap error cases

func (s *AppTestSuite) TestMergeConfigMap_NoConfigManager() {
	// MergeConfigMap without config manager should return nil (no-op)
	app := New()
	// Don't call loadConfig - configMgr is nil
	err := app.MergeConfigMap(map[string]any{"key": "value"})
	s.NoError(err) // Returns nil when configMgr is nil
}
