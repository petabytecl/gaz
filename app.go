package gaz

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"reflect"
	"sync"
	"syscall"
	"time"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"

	"github.com/petabytecl/gaz/config"
	cfgviper "github.com/petabytecl/gaz/config/viper"
	"github.com/petabytecl/gaz/cron"
	"github.com/petabytecl/gaz/di"
	"github.com/petabytecl/gaz/eventbus"
	"github.com/petabytecl/gaz/health"
	"github.com/petabytecl/gaz/logger"
	"github.com/petabytecl/gaz/worker"
)

const (
	defaultShutdownTimeout = 30 * time.Second
	defaultPerHookTimeout  = 10 * time.Second
)

// configProviderType is cached for efficient interface checks.
//
//nolint:gochecknoglobals // Package-level for reflect type caching.
var configProviderType = reflect.TypeOf((*ConfigProvider)(nil)).Elem()

// exitFunc is the function called for force exit. Variable for testability.
// Protected by exitFuncMu for thread-safe access during tests.
//
//nolint:gochecknoglobals // Package-level for test injection of os.Exit.
var (
	exitFunc   = os.Exit
	exitFuncMu sync.RWMutex
)

// callExitFunc safely calls exitFunc with proper synchronization.
func callExitFunc(code int) {
	exitFuncMu.RLock()
	fn := exitFunc
	exitFuncMu.RUnlock()
	fn(code)
}

// AppOptions configuration for App.
type AppOptions struct {
	ShutdownTimeout time.Duration
	PerHookTimeout  time.Duration
	LoggerConfig    *logger.Config
}

// Option configures App settings.

// Option configures App settings.
type Option func(*App)

// WithShutdownTimeout sets the timeout for graceful shutdown.
// Default is 30 seconds.
func WithShutdownTimeout(d time.Duration) Option {
	return func(a *App) {
		a.opts.ShutdownTimeout = d
	}
}

// WithPerHookTimeout sets the default timeout for each shutdown hook.
// Default is 10 seconds. Individual hooks can override via WithHookTimeout.
func WithPerHookTimeout(d time.Duration) Option {
	return func(a *App) {
		a.opts.PerHookTimeout = d
	}
}

// WithLoggerConfig sets the logger configuration.
func WithLoggerConfig(cfg *logger.Config) Option {
	return func(a *App) {
		a.opts.LoggerConfig = cfg
	}
}

// WithStrictConfig enables strict configuration validation.
// If enabled, Build() fails if the config file contains any keys
// that are not mapped to fields in the config struct.
// This helps catch typos and obsolete configuration.
//
// Strict validation is only applied when a config target is set
// via WithConfig(). It has no effect on ConfigProvider pattern.
func WithStrictConfig() Option {
	return func(a *App) {
		a.strictConfig = true
	}
}

// App is the application runtime wrapper.
// It orchestrates dependency injection, lifecycle management, and signal handling.
type App struct {
	container   *Container
	opts        AppOptions
	built       bool            // tracks if Build() was called
	buildErrors []error         // collects registration errors for Build()
	modules     map[string]bool // tracks registered module names for duplicate detection
	cobraCmd    *cobra.Command  // cobra command for module flags integration
	flagFns     []func(*pflag.FlagSet)

	// Logger instance - nil until Build() is called
	Logger *slog.Logger

	// Configuration
	configMgr    *config.Manager
	configTarget any
	strictConfig bool // enables strict config validation

	// Provider config tracking
	providerConfigs []providerConfigEntry // collected from ConfigProvider implementers

	// Idempotency tracking for operations that may run during RegisterCobraFlags
	configLoaded             bool
	providerValuesRegistered bool
	providerConfigsCollected bool
	loggerInitialized        bool // tracks if initializeLogger was called

	// Worker management - nil until Build() is called
	workerMgr *worker.Manager

	// Cron scheduler - nil until Build() is called
	scheduler *cron.Scheduler

	// EventBus for pub/sub - nil until Build() is called
	eventBus *eventbus.EventBus

	mu      sync.Mutex
	running bool
	stopCh  chan struct{}

	// Stop idempotency
	stopOnce sync.Once
	stopErr  error
}

// providerConfigEntry stores config information from a ConfigProvider.
type providerConfigEntry struct {
	providerName string       // type name of the provider
	namespace    string       // from ConfigNamespace()
	flags        []ConfigFlag // from ConfigFlags()
}

// New creates a new App with the given options.
// Use the For[T]() fluent API to register services, then call Build() and Run().
//
// Logger, WorkerManager, Scheduler, and EventBus are NOT created here.
// They are initialized in Build() after config is loaded and flags are parsed.
// This allows logger CLI flags to take effect.
//
// Example:
//
//	app := gaz.New(gaz.WithShutdownTimeout(10 * time.Second))
//	gaz.For[*Database](app.Container()).Provider(NewDatabase)
//	gaz.For[*Request](app.Container()).Transient().Provider(NewRequest)
//	if err := app.Build(); err != nil {
//	    log.Fatal(err)
//	}
//	app.Run(ctx)
func New(opts ...Option) *App {
	app := &App{
		container: NewContainer(),
		opts: AppOptions{
			ShutdownTimeout: defaultShutdownTimeout,
			PerHookTimeout:  defaultPerHookTimeout,
			LoggerConfig: &logger.Config{
				Level:  slog.LevelInfo,
				Format: "json",
			},
		},
		modules: make(map[string]bool),
	}
	for _, opt := range opts {
		opt(app)
	}

	// Initialize ConfigManager with convention-based defaults:
	// - Looks for config.yaml/json/toml in current directory
	// - Environment variables override config file values
	// Use WithConfig() to customize options or load into a struct.
	app.configMgr = config.New(
		config.WithBackend(cfgviper.New()),
		config.WithName("config"),
		config.WithSearchPaths("."),
	)

	return app
}

// Container returns the underlying container.
// This is useful for advanced use cases or testing.
func (a *App) Container() *Container {
	return a.container
}

// AddFlagsFn registers a function that adds flags to the application.
// Flags are stored and applied when WithCobra option is processed or
// when a Cobra command is attached.
// If a Cobra command is already attached, flags are applied immediately.
func (a *App) AddFlagsFn(fn func(*pflag.FlagSet)) {
	if fn == nil {
		return
	}
	a.flagFns = append(a.flagFns, fn)

	// If cobra command is already attached, apply flags immediately
	if a.cobraCmd != nil {
		fn(a.cobraCmd.PersistentFlags())
	}
}

// initializeLogger creates the logger from resolved config or defaults.
// Called during Build() after config is loaded and flags are parsed.
// This method is idempotent - subsequent calls return nil.
func (a *App) initializeLogger() error {
	if a.loggerInitialized {
		return nil
	}

	// Check if logger.Config is available (logger module registered)
	cfg, err := Resolve[logger.Config](a.container)
	if err != nil {
		// No logger module - use option config or defaults
		if a.opts.LoggerConfig == nil {
			a.opts.LoggerConfig = &logger.Config{
				Level:  slog.LevelInfo,
				Format: "text",
			}
		}
		a.Logger = logger.NewLogger(a.opts.LoggerConfig)
	} else {
		// Logger module provided config - use it
		a.Logger = logger.NewLogger(&cfg)
	}

	// Register Logger in container
	if regErr := For[*slog.Logger](a.container).Instance(a.Logger); regErr != nil {
		return fmt.Errorf("register logger: %w", regErr)
	}

	a.loggerInitialized = true
	return nil
}

// initializeSubsystems creates WorkerManager, Scheduler, EventBus.
// Called during Build() after logger is initialized.
func (a *App) initializeSubsystems() error {
	// Use slog.Default() if Logger is nil (shouldn't happen after initializeLogger)
	log := a.Logger
	if log == nil {
		log = slog.Default()
	}

	// WorkerManager
	a.workerMgr = worker.NewManager(log)
	a.workerMgr.SetCriticalFailHandler(func() {
		log.Error("critical worker failed, initiating shutdown")
		go func() {
			ctx, cancel := context.WithTimeout(context.Background(), a.opts.ShutdownTimeout)
			defer cancel()
			_ = a.Stop(ctx)
		}()
	})

	// Scheduler
	a.scheduler = cron.NewScheduler(a.container, context.Background(), log)

	// EventBus
	a.eventBus = eventbus.New(log)

	// Register EventBus in container
	if err := For[*eventbus.EventBus](a.container).Instance(a.eventBus); err != nil {
		return fmt.Errorf("register eventbus: %w", err)
	}
	return nil
}

// EventBus returns the application's EventBus for pub/sub.
// Returns nil if called before Build().
// Prefer injecting *eventbus.EventBus as a dependency instead.
func (a *App) EventBus() *eventbus.EventBus {
	return a.eventBus
}

// getLogger returns the app's logger or slog.Default() if not initialized.
// This allows methods to safely log before Build() is called.
func (a *App) getLogger() *slog.Logger {
	if a.Logger != nil {
		return a.Logger
	}
	return slog.Default()
}

// WithConfig configures the application to load configuration into the target struct.
// The target must be a pointer to a struct, or nil to only customize config options.
//
// The configuration is loaded from:
// 1. Defaults (via Defaulter interface)
// 2. Config files (yaml, json, toml) in specified paths
// 3. Environment variables (if WithEnvPrefix is set)
// 4. Flags (if WithCobra is used)
//
// By default, gaz looks for config.yaml in the current directory. Use this method to:
// - Load config into a struct (target != nil)
// - Customize config options (change search paths, env prefix, etc.)
//
// If you only use ConfigProvider pattern for config, you don't need to call this method.
func (a *App) WithConfig(target any, opts ...config.Option) *App {
	if a.built {
		panic("gaz: cannot configure config after Build()")
	}

	// If options provided, recreate config manager with new options
	// Options are applied on top of viper backend (always required)
	if len(opts) > 0 {
		configOpts := make([]config.Option, 0, len(opts)+1)
		configOpts = append(configOpts, config.WithBackend(cfgviper.New()))
		configOpts = append(configOpts, opts...)
		a.configMgr = config.New(configOpts...)
	}

	// If target provided, set it for loading and register in container
	if target != nil {
		a.configTarget = target
		if err := a.registerInstance(target); err != nil {
			a.buildErrors = append(a.buildErrors, err)
		}
	}

	return a
}

// configMapMerger is implemented by backends that support merging config maps.
type configMapMerger interface {
	MergeConfigMap(cfg map[string]any) error
}

// MergeConfigMap merges raw config values into the app's configuration.
// This is primarily intended for testing scenarios where you want to inject
// config values without loading from files.
//
// Must be called before Build().
// Panics if called after Build().
func (a *App) MergeConfigMap(cfg map[string]any) error {
	if a.built {
		panic("gaz: cannot merge config after Build()")
	}
	if a.configMgr == nil {
		return nil
	}
	backend := a.configMgr.Backend()
	if merger, ok := backend.(configMapMerger); ok {
		if err := merger.MergeConfigMap(cfg); err != nil {
			return fmt.Errorf("gaz: merge config map: %w", err)
		}
		return nil
	}
	// Fallback: use Set() for each key (loses nested structure but works for flat keys)
	for k, v := range cfg {
		backend.Set(k, v)
	}
	return nil
}

// registerInstance registers a pre-built instance using reflection.
func (a *App) registerInstance(instance any) error {
	instanceType := reflect.TypeOf(instance)
	if instanceType == nil {
		return fmt.Errorf("%w: instance cannot be nil", ErrDIInvalidProvider)
	}

	typeNameStr := typeName(instanceType)

	// Check for duplicate registration
	if a.container.HasService(typeNameStr) {
		return fmt.Errorf("%w: %s", ErrDIDuplicate, typeNameStr)
	}

	svc := di.NewInstanceServiceAny(typeNameStr, typeNameStr, instance)
	a.container.Register(typeNameStr, svc)
	return nil
}

// loadConfig loads the configuration from all sources.
// This method is idempotent - subsequent calls return nil after first load.
func (a *App) loadConfig() error {
	if a.configLoaded {
		return nil // Already loaded
	}
	if a.configMgr == nil {
		return nil
	}
	// If a target struct is provided, load and unmarshal into it
	if a.configTarget != nil {
		if a.strictConfig {
			if err := a.configMgr.LoadIntoStrict(a.configTarget); err != nil {
				return fmt.Errorf("loading config (strict mode): %w", err)
			}
		} else {
			if err := a.configMgr.LoadInto(a.configTarget); err != nil {
				return fmt.Errorf("loading config into target: %w", err)
			}
		}
	} else {
		// Otherwise just load the config file (for ConfigProvider pattern)
		if err := a.configMgr.Load(); err != nil {
			return fmt.Errorf("loading config: %w", err)
		}
	}
	a.configLoaded = true
	return nil
}

// registerProviderValuesEarly registers ProviderValues as an instance
// immediately after config loading, BEFORE providers are instantiated.
// This allows providers to inject *ProviderValues as a dependency.
// This method is idempotent - subsequent calls return nil after first registration.
func (a *App) registerProviderValuesEarly() error {
	if a.providerValuesRegistered {
		return nil // Already registered
	}
	if a.configMgr == nil {
		return nil
	}
	pv := &ProviderValues{backend: a.configMgr.Backend()}
	if err := a.registerInstance(pv); err != nil {
		return err
	}
	a.providerValuesRegistered = true
	return nil
}

// getSortedServiceNames returns service names in sorted order for deterministic iteration.
func (a *App) getSortedServiceNames() []string {
	return a.container.List()
}

// collectProviderConfigs iterates registered services, collects config from ConfigProvider
// implementers, detects key collisions, registers provider flags with ConfigManager,
// validates required fields, and registers ProviderValues.
// This method is idempotent - subsequent calls return nil after first collection.
func (a *App) collectProviderConfigs() error {
	if a.providerConfigsCollected {
		return nil // Already collected
	}
	keyOwners := make(map[string]string)
	var collisionErrors []error

	// Iterate in sorted order for deterministic dependency graph recording
	for _, typeName := range a.getSortedServiceNames() {
		wrapper, exists := a.container.GetService(typeName)
		if !exists {
			continue
		}

		if wrapper.IsTransient() {
			continue
		}

		// Check if service type implements ConfigProvider BEFORE instantiation
		// This avoids side effects of instantiating non-ConfigProvider services
		serviceType := wrapper.ServiceType()
		if serviceType == nil {
			continue
		}

		// For pointer types, check both pointer and element type
		if !serviceType.Implements(configProviderType) {
			// Also check pointer-to-type in case methods are on *T
			if serviceType.Kind() != reflect.Ptr {
				ptrType := reflect.PointerTo(serviceType)
				if !ptrType.Implements(configProviderType) {
					continue
				}
			} else {
				continue
			}
		}

		// Only now instantiate - we know it implements ConfigProvider
		instance, err := a.container.ResolveByName(typeName, nil)
		if err != nil {
			continue // Skip services that fail to resolve
		}

		cp, ok := instance.(ConfigProvider)
		if !ok {
			// This shouldn't happen if type check above is correct, but be defensive
			continue
		}

		namespace := cp.ConfigNamespace()
		flags := cp.ConfigFlags()

		a.providerConfigs = append(a.providerConfigs, providerConfigEntry{
			providerName: typeName,
			namespace:    namespace,
			flags:        flags,
		})

		// Check for collisions
		for _, flag := range flags {
			fullKey := namespace + "." + flag.Key
			if existingProvider, found := keyOwners[fullKey]; found {
				collisionErrors = append(collisionErrors, fmt.Errorf(
					"%w: key %q registered by both %q and %q",
					ErrConfigKeyCollision, fullKey, existingProvider, typeName,
				))
			} else {
				keyOwners[fullKey] = typeName
			}
		}
	}

	if len(collisionErrors) > 0 {
		return errors.Join(collisionErrors...)
	}

	// Set flag BEFORE registerProviderFlags to avoid re-entry issues
	a.providerConfigsCollected = true
	return a.registerProviderFlags()
}

// registerProviderFlags registers collected provider flags with ConfigManager and validates.
// Note: ProviderValues is already registered by registerProviderValuesEarly().
func (a *App) registerProviderFlags() error {
	if a.configMgr == nil {
		return nil
	}

	var validationErrors []error
	for _, entry := range a.providerConfigs {
		// Convert gaz.ConfigFlag to config.ConfigFlag
		cfgFlags := make([]config.ConfigFlag, len(entry.flags))
		for i, f := range entry.flags {
			cfgFlags[i] = config.ConfigFlag{
				Key:      f.Key,
				Default:  f.Default,
				Required: f.Required,
			}
		}

		if err := a.configMgr.RegisterProviderFlags(entry.namespace, cfgFlags); err != nil {
			return fmt.Errorf("registering provider flags for %s: %w", entry.namespace, err)
		}
		errs := a.configMgr.ValidateProviderFlags(entry.namespace, cfgFlags)
		validationErrors = append(validationErrors, errs...)
	}

	if len(validationErrors) > 0 {
		return errors.Join(validationErrors...)
	}

	return nil
}

// discoverWorkers iterates registered services and registers those implementing
// worker.Worker interface with the WorkerManager.
func (a *App) discoverWorkers() {
	a.container.ForEachService(func(name string, svc di.ServiceWrapper) {
		// Skip transient services
		if svc.IsTransient() {
			return
		}

		// Try to resolve and check for Worker interface
		instance, err := a.container.ResolveByName(name, nil)
		if err != nil {
			return // Skip services that fail to resolve
		}

		if w, ok := instance.(worker.Worker); ok {
			// Register with default options
			// Providers can customize via WithWorkerOptions in future
			if regErr := a.workerMgr.Register(w); regErr != nil {
				a.getLogger().Warn("failed to register worker",
					"name", name,
					"error", regErr,
				)
			}
		}
	})
}

// discoverCronJobs iterates registered services and registers those implementing
// cron.CronJob interface with the Scheduler.
//
// CronJobs should be registered using one of:
//
//	gaz.For[cron.CronJob](c).Transient().Provider(NewMyJob)
//	gaz.For[cron.CronJob](c).Named("job-name").Transient().Provider(NewMyJob)
//
// This ensures the service is registered with the CronJob interface type,
// allowing discovery without resolving unrelated transient services.
func (a *App) discoverCronJobs() {
	cronJobTypeName := di.TypeName[cron.CronJob]()

	a.container.ForEachService(func(name string, svc di.ServiceWrapper) {
		// Only process services registered as cron.CronJob interface
		// TypeName() returns the interface type, so check if name equals it
		// or if it's a named registration (name != type, but typeName = interface)
		if svc.TypeName() != cronJobTypeName {
			return
		}

		// CronJobs should be transient (new instance per execution)
		if !svc.IsTransient() {
			a.getLogger().Warn("CronJob should be transient",
				"name", name,
			)
		}

		// Try to resolve and check for CronJob interface
		instance, err := a.container.ResolveByName(name, nil)
		if err != nil {
			return // Skip services that fail to resolve
		}

		if job, ok := instance.(cron.CronJob); ok {
			// Register with scheduler using service name for later resolution
			if regErr := a.scheduler.RegisterJob(
				name,           // serviceName for container resolution
				job.Name(),     // human name for logging
				job.Schedule(), // cron expression
				job.Timeout(),  // execution timeout
			); regErr != nil {
				a.getLogger().Warn("failed to register cron job",
					"name", job.Name(),
					"error", regErr,
				)
			}
		}
	})
}

// Build validates all registrations and instantiates eager services.
// It aggregates all errors and returns them using errors.Join.
// Build is idempotent - calling it multiple times after success returns nil.
func (a *App) Build() error {
	a.mu.Lock()
	defer a.mu.Unlock()

	if a.built {
		return nil // Already built, idempotent
	}

	// Load configuration first
	if err := a.loadConfig(); err != nil {
		return err
	}

	// Collect any registration errors
	var errs []error
	errs = append(errs, a.buildErrors...)

	// Register ProviderValues EARLY so providers can inject it
	if err := a.registerProviderValuesEarly(); err != nil {
		errs = append(errs, err)
	}

	// Initialize Logger BEFORE collecting provider configs
	// This allows logger config from modules to be used
	if err := a.initializeLogger(); err != nil {
		errs = append(errs, err)
	}

	// Initialize subsystems (WorkerManager, Scheduler, EventBus) after logger
	if err := a.initializeSubsystems(); err != nil {
		errs = append(errs, err)
	}

	// Collect provider configs from registered services
	// Now providers can inject *ProviderValues as a dependency
	if err := a.collectProviderConfigs(); err != nil {
		errs = append(errs, err)
	}

	// Auto-register health module if config implements HealthConfigProvider
	// and health module is not already registered
	if a.configTarget != nil {
		if hp, ok := a.configTarget.(health.HealthConfigProvider); ok {
			// Only auto-register if health module not already applied
			if !a.modules["health"] {
				cfg := hp.HealthConfig()

				// Register health.Config in container
				if err := For[health.Config](a.container).Instance(cfg); err != nil {
					errs = append(errs, fmt.Errorf("register health config: %w", err))
				} else {
					// Create and apply health module
					healthModule := NewModule("health").
						Provide(health.Module).
						Build()
					if applyErr := healthModule.Apply(a); applyErr != nil {
						errs = append(errs, fmt.Errorf("apply health module: %w", applyErr))
					} else {
						a.modules["health"] = true
					}
				}
			}
		}
	}

	// Discover workers from registered services
	a.discoverWorkers()

	// Register EventBus with worker manager for lifecycle management
	if err := a.workerMgr.Register(a.eventBus); err != nil {
		errs = append(errs, fmt.Errorf("registering eventbus: %w", err))
	}

	// Discover cron jobs from registered services
	a.discoverCronJobs()

	// Register scheduler with worker manager (only if jobs exist)
	if a.scheduler.JobCount() > 0 {
		if err := a.workerMgr.Register(a.scheduler); err != nil {
			errs = append(errs, fmt.Errorf("registering scheduler: %w", err))
		}
	}

	// Delegate to container.Build() for eager instantiation
	if err := a.container.Build(); err != nil {
		errs = append(errs, err)
	}

	if len(errs) > 0 {
		return errors.Join(errs...)
	}

	a.built = true
	return nil
}

// Run executes the application lifecycle.
// It builds the container, starts services in order, and waits for a signal or stop call.
func (a *App) Run(ctx context.Context) error {
	if err := a.Build(); err != nil {
		return err
	}

	a.mu.Lock()
	if a.running {
		a.mu.Unlock()
		return errors.New("app is already running")
	}
	a.stopCh = make(chan struct{})
	a.running = true
	a.mu.Unlock()

	defer func() {
		a.mu.Lock()
		a.running = false
		a.mu.Unlock()
	}()

	// Compute startup order
	graph := a.container.GetGraph()
	services := make(map[string]di.ServiceWrapper)
	a.container.ForEachService(func(name string, svc di.ServiceWrapper) {
		// Skip workers - they have their own lifecycle via WorkerManager
		// Workers implement OnStart/OnStop which looks like di.Starter/di.Stopper,
		// but they should only be started/stopped by WorkerManager, not the DI layer.
		if !svc.IsTransient() {
			if instance, err := a.container.ResolveByName(name, nil); err == nil {
				if _, isWorker := instance.(worker.Worker); isWorker {
					return
				}
			}
		}
		services[name] = svc
	})

	startupOrder, err := ComputeStartupOrder(graph, services)
	if err != nil {
		return err
	}

	a.Logger.InfoContext(ctx, "starting application", "services_count", len(services))

	// Start services layer by layer
	for _, layer := range startupOrder {
		var wg sync.WaitGroup
		errCh := make(chan error, len(layer))

		for _, name := range layer {
			svc := services[name]
			wg.Add(1)
			go func() {
				defer wg.Done()
				start := time.Now()
				if startErr := svc.Start(ctx); startErr != nil {
					a.Logger.ErrorContext(
						ctx,
						"failed to start service",
						"name", name,
						"error", startErr,
					)
					errCh <- fmt.Errorf("starting service %s: %w", name, startErr)
				} else {
					a.Logger.InfoContext(
						ctx,
						"service started",
						"name", name,
						"duration", time.Since(start),
					)
				}
			}()
		}
		wg.Wait()
		close(errCh)

		if startupErr := <-errCh; startupErr != nil {
			// Rollback: stop everything we started?
			// For simplicity, we call Stop() which attempts to stop everything.
			// Ideally we only stop what started, but Stop() is safe to call on everything.
			// Use background context for rollback as original ctx might be fine but we are failing.
			shutdownCtx, cancel := context.WithTimeout(context.Background(), a.opts.ShutdownTimeout)
			defer cancel()
			_ = a.Stop(shutdownCtx)
			return startupErr
		}
	}

	// Start workers after all services started
	a.Logger.InfoContext(ctx, "starting workers")
	if workerErr := a.workerMgr.Start(ctx); workerErr != nil {
		// Rollback
		shutdownCtx, cancel := context.WithTimeout(context.Background(), a.opts.ShutdownTimeout)
		defer cancel()
		_ = a.Stop(shutdownCtx)
		return fmt.Errorf("starting workers: %w", workerErr)
	}

	return a.waitForShutdownSignal(ctx)
}

// waitForShutdownSignal blocks until a shutdown trigger (signal, context cancel, or Stop call).
// Returns the result of graceful shutdown.
func (a *App) waitForShutdownSignal(ctx context.Context) error {
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, os.Interrupt, syscall.SIGTERM)
	defer signal.Stop(sigCh)

	select {
	case <-ctx.Done():
		// Context cancelled, treat like SIGTERM (graceful, no double-signal)
		a.Logger.InfoContext(ctx, "Shutting down gracefully...", "reason", "context cancelled")
		shutdownCtx, cancel := context.WithTimeout(context.Background(), a.opts.ShutdownTimeout)
		defer cancel()
		return a.Stop(shutdownCtx)

	case sig := <-sigCh:
		return a.handleSignalShutdown(ctx, sig, sigCh)

	case <-a.stopCh:
		// Stopped externally (Stop() called)
		return nil
	}
}

// handleSignalShutdown handles graceful shutdown triggered by a signal.
// For SIGINT, it spawns a force-exit watcher that exits immediately on second SIGINT.
// For SIGTERM, it performs graceful shutdown without double-signal behavior.
func (a *App) handleSignalShutdown(
	ctx context.Context,
	sig os.Signal,
	sigCh <-chan os.Signal,
) error {
	// Log hint message about force exit option
	a.Logger.InfoContext(ctx, "Shutting down gracefully...", "hint", "Ctrl+C again to force")

	// Create shutdown context
	shutdownCtx, cancel := context.WithTimeout(context.Background(), a.opts.ShutdownTimeout)
	defer cancel()

	// Channel to receive shutdown result
	shutdownDone := make(chan error, 1)

	// Start graceful shutdown in goroutine so we can continue listening for signals
	go func() {
		shutdownDone <- a.Stop(shutdownCtx)
	}()

	// If SIGINT, spawn force-exit watcher goroutine
	if sig == os.Interrupt {
		go func() {
			select {
			case <-sigCh:
				// Second SIGINT received - force exit immediately
				a.Logger.ErrorContext(ctx, "Received second interrupt, forcing exit")
				callExitFunc(1)
			case <-shutdownDone:
				// Normal completion, watcher exits
			}
		}()
	}

	// Wait for shutdown to complete
	return <-shutdownDone
}

// Stop initiates graceful shutdown of the application.
// It executes OnStop hooks for all services in reverse dependency order.
// Safe to call even if Run() was not used (e.g., Cobra integration).
// Stop is idempotent - calling it multiple times returns the same result.
func (a *App) Stop(ctx context.Context) error {
	a.stopOnce.Do(func() {
		a.stopErr = a.doStop(ctx)
	})
	return a.stopErr
}

// doStop performs the actual shutdown. Called only once via stopOnce.
func (a *App) doStop(ctx context.Context) error {
	a.mu.Lock()
	wasRunning := a.running
	wasBuilt := a.built
	a.mu.Unlock()

	// If app was never built, there's nothing to stop
	if !wasBuilt {
		return nil
	}

	// Start global timeout force-exit goroutine
	done := make(chan struct{})
	go func() {
		select {
		case <-done:
			return
		case <-time.After(a.opts.ShutdownTimeout):
			msg := fmt.Sprintf(
				"shutdown: global timeout %s exceeded, forcing exit",
				a.opts.ShutdownTimeout,
			)
			a.getLogger().Error(msg)
			fmt.Fprintln(os.Stderr, msg)
			callExitFunc(1)
		}
	}()

	// Compute shutdown order (reverse of startup)
	// We need to re-compute because we don't store it.
	graph := a.container.GetGraph()
	services := make(map[string]di.ServiceWrapper)
	a.container.ForEachService(func(name string, svc di.ServiceWrapper) {
		// Skip workers - they have their own lifecycle via WorkerManager
		// Workers implement OnStart/OnStop which looks like di.Starter/di.Stopper,
		// but they should only be started/stopped by WorkerManager, not the DI layer.
		if !svc.IsTransient() {
			if instance, err := a.container.ResolveByName(name, nil); err == nil {
				if _, isWorker := instance.(worker.Worker); isWorker {
					return
				}
			}
		}
		services[name] = svc
	})

	startupOrder, err := ComputeStartupOrder(graph, services)
	if err != nil {
		close(done)
		// Should not happen if Build passed, unless graph changed (impossible after Build)
		return err
	}
	shutdownOrder := ComputeShutdownOrder(startupOrder)

	var errs []error

	// Get logger safely (uses slog.Default() if nil)
	log := a.getLogger()

	// Stop workers first (they may depend on services)
	log.InfoContext(ctx, "stopping workers")
	if a.workerMgr != nil {
		if workerStopErr := a.workerMgr.Stop(); workerStopErr != nil {
			errs = append(errs, fmt.Errorf("stopping workers: %w", workerStopErr))
		}
	}

	if serviceStopErr := a.stopServices(ctx, shutdownOrder, services); serviceStopErr != nil {
		errs = append(errs, serviceStopErr)
	}

	// Cancel the force-exit goroutine
	close(done)

	// Signal Run to exit (only if Run() was used)
	if wasRunning {
		a.mu.Lock()
		select {
		case <-a.stopCh:
			// Already closed
		default:
			close(a.stopCh)
		}
		a.mu.Unlock()
	}

	if len(errs) > 0 {
		return errors.Join(errs...)
	}
	return nil
}

// stopServices stops services sequentially with per-hook timeout and blame logging.
func (a *App) stopServices(
	ctx context.Context,
	order [][]string,
	services map[string]di.ServiceWrapper,
) error {
	var errs []error

	// Stop services layer by layer, sequentially within each layer
	for _, layer := range order {
		for _, name := range layer {
			svc := services[name]

			// Create per-hook timeout context
			timeout := a.opts.PerHookTimeout
			hookCtx, cancel := context.WithTimeout(ctx, timeout)

			// Run hook in goroutine so we can detect timeout
			start := time.Now()
			errCh := make(chan error, 1)
			go func() {
				errCh <- svc.Stop(hookCtx)
			}()

			// Wait for hook completion or timeout
			select {
			case stopErr := <-errCh:
				cancel()
				elapsed := time.Since(start)
				if stopErr != nil {
					a.Logger.ErrorContext(
						ctx,
						"failed to stop service",
						"name", name,
						"error", stopErr,
						"elapsed", elapsed,
					)
					errs = append(errs, fmt.Errorf("stopping service %s: %w", name, stopErr))
				} else {
					a.Logger.InfoContext(
						ctx,
						"service stopped",
						"name", name,
						"duration", elapsed,
					)
				}
			case <-hookCtx.Done():
				cancel()
				elapsed := time.Since(start)
				// Blame logging: hook exceeded timeout
				a.logBlame(name, timeout, elapsed)
				errs = append(
					errs,
					fmt.Errorf("stopping service %s: %w", name, context.DeadlineExceeded),
				)
				// Continue to next hook (don't wait for the timed-out hook)
			}
		}
	}

	if len(errs) > 0 {
		return errors.Join(errs...)
	}
	return nil
}

// logBlame logs blame information when a hook exceeds its timeout.
// Uses Logger first, falls back to stderr if Logger fails.
func (a *App) logBlame(hookName string, timeout, elapsed time.Duration) {
	msg := fmt.Sprintf("shutdown: %s exceeded %s timeout (elapsed: %s)", hookName, timeout, elapsed)

	// Try structured logger first
	if a.Logger != nil {
		a.Logger.Error(msg, "hook", hookName, "timeout", timeout, "elapsed", elapsed)
	}
	// Always write to stderr as fallback (guaranteed output even if logger is broken)
	fmt.Fprintln(os.Stderr, msg)
}
