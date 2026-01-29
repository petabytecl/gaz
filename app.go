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

	"github.com/petabytecl/gaz/config"
	cfgviper "github.com/petabytecl/gaz/config/viper"
	"github.com/petabytecl/gaz/di"
	"github.com/petabytecl/gaz/logger"
	"github.com/petabytecl/gaz/worker"
)

const (
	defaultShutdownTimeout = 30 * time.Second
	defaultPerHookTimeout  = 10 * time.Second
)

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

// App is the application runtime wrapper.
// It orchestrates dependency injection, lifecycle management, and signal handling.
type App struct {
	container   *Container
	opts        AppOptions
	built       bool            // tracks if Build() was called
	buildErrors []error         // collects registration errors for Build()
	modules     map[string]bool // tracks registered module names for duplicate detection

	// Logger instance
	Logger *slog.Logger

	// Configuration
	configMgr    *config.Manager
	configTarget any

	// Provider config tracking
	providerConfigs []providerConfigEntry // collected from ConfigProvider implementers

	// Idempotency tracking for operations that may run during RegisterCobraFlags
	configLoaded             bool
	providerValuesRegistered bool
	providerConfigsCollected bool

	// Worker management
	workerMgr *worker.Manager

	mu      sync.Mutex
	running bool
	stopCh  chan struct{}
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
		},
		modules: make(map[string]bool),
	}
	for _, opt := range opts {
		opt(app)
	}

	// Initialize Logger
	if app.opts.LoggerConfig == nil {
		app.opts.LoggerConfig = &logger.Config{
			Level:  slog.LevelInfo,
			Format: "json",
		}
	}
	app.Logger = logger.NewLogger(app.opts.LoggerConfig)

	// Initialize WorkerManager
	app.workerMgr = worker.NewManager(app.Logger)
	app.workerMgr.SetCriticalFailHandler(func() {
		app.Logger.Error("critical worker failed, initiating shutdown")
		go func() {
			ctx, cancel := context.WithTimeout(context.Background(), app.opts.ShutdownTimeout)
			defer cancel()
			_ = app.Stop(ctx)
		}()
	})

	// Register Logger in container using For[T]() pattern
	if err := For[*slog.Logger](app.container).Instance(app.Logger); err != nil {
		// Should not happen as container is empty
		panic(fmt.Errorf("failed to register logger: %w", err))
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

// registerInstance registers a pre-built instance using reflection.
func (a *App) registerInstance(instance any) error {
	instanceType := reflect.TypeOf(instance)
	if instanceType == nil {
		return fmt.Errorf("%w: instance cannot be nil", ErrInvalidProvider)
	}

	typeNameStr := typeName(instanceType)

	// Check for duplicate registration
	if a.container.HasService(typeNameStr) {
		return fmt.Errorf("%w: %s", ErrDuplicate, typeNameStr)
	}

	svc := di.NewInstanceServiceAny(typeNameStr, typeNameStr, instance, nil, nil)
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
		if err := a.configMgr.LoadInto(a.configTarget); err != nil {
			return err
		}
	} else {
		// Otherwise just load the config file (for ConfigProvider pattern)
		if err := a.configMgr.Load(); err != nil {
			return err
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

		// Use ResolveByName() instead of GetInstance() to ensure dependencies are recorded
		instance, err := a.container.ResolveByName(typeName, nil)
		if err != nil {
			continue // Skip services that fail to resolve
		}

		cp, ok := instance.(ConfigProvider)
		if !ok {
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
			return err
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
func (a *App) discoverWorkers() error {
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
				a.Logger.Warn("failed to register worker",
					"name", name,
					"error", regErr,
				)
			}
		}
	})
	return nil
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

	// Collect provider configs from registered services
	// Now providers can inject *ProviderValues as a dependency
	if err := a.collectProviderConfigs(); err != nil {
		errs = append(errs, err)
	}

	// Discover workers from registered services
	if err := a.discoverWorkers(); err != nil {
		errs = append(errs, err)
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
	if err := a.workerMgr.Start(ctx); err != nil {
		// Rollback
		shutdownCtx, cancel := context.WithTimeout(context.Background(), a.opts.ShutdownTimeout)
		defer cancel()
		_ = a.Stop(shutdownCtx)
		return fmt.Errorf("starting workers: %w", err)
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
func (a *App) Stop(ctx context.Context) error {
	a.mu.Lock()
	wasRunning := a.running
	a.mu.Unlock()

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
			a.Logger.Error(msg)
			fmt.Fprintln(os.Stderr, msg)
			callExitFunc(1)
		}
	}()

	// Compute shutdown order (reverse of startup)
	// We need to re-compute because we don't store it.
	graph := a.container.GetGraph()
	services := make(map[string]di.ServiceWrapper)
	a.container.ForEachService(func(name string, svc di.ServiceWrapper) {
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

	// Stop workers first (they may depend on services)
	a.Logger.InfoContext(ctx, "stopping workers")
	if err := a.workerMgr.Stop(); err != nil {
		errs = append(errs, fmt.Errorf("stopping workers: %w", err))
	}

	if err := a.stopServices(ctx, shutdownOrder, services); err != nil {
		errs = append(errs, err)
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
