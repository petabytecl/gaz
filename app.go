package gaz

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"reflect"
	"sort"
	"sync"
	"syscall"
	"time"

	"github.com/petabytecl/gaz/logger"
)

const (
	defaultShutdownTimeout = 30 * time.Second
	defaultPerHookTimeout  = 10 * time.Second
)

// providerWithErrorReturnCount is the expected number of return values for a provider
// that returns (T, error).
const providerWithErrorReturnCount = 2

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

// AppOption configures AppOptions.
//
// Deprecated: Use Option instead.
type AppOption func(*AppOptions)

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

// withShutdownTimeoutLegacy is the legacy version for NewApp().
func withShutdownTimeoutLegacy(d time.Duration) AppOption {
	return func(o *AppOptions) {
		o.ShutdownTimeout = d
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
	configManager *ConfigManager

	// Provider config tracking
	providerConfigs []providerConfigEntry // collected from ConfigProvider implementers

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
// Use the fluent provider methods (ProvideSingleton, ProvideTransient, etc.)
// to register services, then call Build() and Run().
//
// Example:
//
//	app := gaz.New(gaz.WithShutdownTimeout(10 * time.Second))
//	app.ProvideSingleton(NewDatabase).
//	    ProvideTransient(NewRequest)
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

	// Register Logger in container
	if err := app.registerInstance(app.Logger); err != nil {
		// Should not happen as container is empty
		panic(fmt.Errorf("failed to register logger: %w", err))
	}

	return app
}

// NewApp creates a new App with the given container and options.
//
// Deprecated: Use New() with fluent provider methods instead.
func NewApp(c *Container, opts ...AppOption) *App {
	options := AppOptions{
		ShutdownTimeout: defaultShutdownTimeout,
		PerHookTimeout:  defaultPerHookTimeout,
	}
	for _, opt := range opts {
		opt(&options)
	}

	// For legacy NewApp, use default logger since we can't easily configure it via AppOption
	// without breaking changes or adding new AppOption types.
	// We'll create a default logger here.
	defaultLogger := logger.NewLogger(&logger.Config{
		Level:  slog.LevelInfo,
		Format: "json",
	})

	app := &App{
		container: c,
		opts:      options,
		Logger:    defaultLogger,
	}

	// Register Logger in container
	// Ignore error if already registered (user might have put one in container)
	_ = app.registerInstance(defaultLogger)

	return app
}

// Container returns the underlying container.
// This is useful for advanced use cases or testing.
func (a *App) Container() *Container {
	return a.container
}

// WithConfig configures the application to load configuration into the target struct.
// The target must be a pointer to a struct.
//
// The configuration is loaded from:
// 1. Defaults (via Defaulter interface)
// 2. Config files (yaml, json, toml) in specified paths
// 3. Environment variables (if WithEnvPrefix is set)
// 4. Flags (if WithCobra is used)
//
// The config object is automatically registered as a singleton instance in the container.
func (a *App) WithConfig(target any, opts ...ConfigOption) *App {
	if a.built {
		panic("gaz: cannot configure config after Build()")
	}

	a.configManager = NewConfigManager(target, opts...)

	// Register the config object in the container
	// We register the pointer itself, as that's what will be populated
	if err := a.registerInstance(target); err != nil {
		a.buildErrors = append(a.buildErrors, err)
	}

	return a
}

// ProvideSingleton registers a provider as a singleton (one instance per container).
// The provider function must have signature: func(*Container) (T, error) or func(*Container) T
// Returns the App for method chaining.
//
// Example:
//
//	app.ProvideSingleton(func(c *gaz.Container) (*Database, error) {
//	    return NewDatabase(), nil
//	})
func (a *App) ProvideSingleton(provider any) *App {
	if a.built {
		panic("gaz: cannot add providers after Build()")
	}
	if err := a.registerProvider(provider, scopeSingleton, true); err != nil {
		a.buildErrors = append(a.buildErrors, err)
	}
	return a
}

// ProvideTransient registers a provider as transient (new instance per resolution).
// The provider function must have signature: func(*Container) (T, error) or func(*Container) T
// Returns the App for method chaining.
func (a *App) ProvideTransient(provider any) *App {
	if a.built {
		panic("gaz: cannot add providers after Build()")
	}
	if err := a.registerProvider(provider, scopeTransient, true); err != nil {
		a.buildErrors = append(a.buildErrors, err)
	}
	return a
}

// ProvideEager registers a provider as an eager singleton (instantiated at Build time).
// The provider function must have signature: func(*Container) (T, error) or func(*Container) T
// Returns the App for method chaining.
func (a *App) ProvideEager(provider any) *App {
	if a.built {
		panic("gaz: cannot add providers after Build()")
	}
	if err := a.registerProvider(provider, scopeSingleton, false); err != nil {
		a.buildErrors = append(a.buildErrors, err)
	}
	return a
}

// ProvideInstance registers a pre-built value as a singleton.
// Returns the App for method chaining.
func (a *App) ProvideInstance(instance any) *App {
	if a.built {
		panic("gaz: cannot add providers after Build()")
	}
	if err := a.registerInstance(instance); err != nil {
		a.buildErrors = append(a.buildErrors, err)
	}
	return a
}

// registerProvider uses reflection to extract the return type and register the provider.
func (a *App) registerProvider(provider any, scope serviceScope, lazy bool) error {
	providerType := reflect.TypeOf(provider)
	if providerType == nil || providerType.Kind() != reflect.Func {
		return fmt.Errorf("%w: provider must be a function", ErrInvalidProvider)
	}

	// Validate input: must accept *Container
	if providerType.NumIn() != 1 {
		return fmt.Errorf(
			"%w: provider must accept exactly one argument (*Container)",
			ErrInvalidProvider,
		)
	}
	containerType := reflect.TypeOf((*Container)(nil))
	if providerType.In(0) != containerType {
		return fmt.Errorf("%w: provider argument must be *Container", ErrInvalidProvider)
	}

	// Validate output: must return (T) or (T, error)
	numOut := providerType.NumOut()
	hasErrorReturn := numOut == providerWithErrorReturnCount
	if numOut < 1 || numOut > 2 {
		return fmt.Errorf("%w: provider must return (T) or (T, error)", ErrInvalidProvider)
	}
	if hasErrorReturn {
		errorType := reflect.TypeOf((*error)(nil)).Elem()
		if !providerType.Out(1).Implements(errorType) {
			return fmt.Errorf("%w: second return value must be error", ErrInvalidProvider)
		}
	}

	returnType := providerType.Out(0)
	typeNameStr := typeName(returnType)

	// Create a wrapped provider that handles both (T) and (T, error) signatures
	providerValue := reflect.ValueOf(provider)
	wrappedProvider := func(c *Container) (any, error) {
		results := providerValue.Call([]reflect.Value{reflect.ValueOf(c)})
		instance := results[0].Interface()
		if hasErrorReturn && !results[1].IsNil() {
			err, _ := results[1].Interface().(error)
			return nil, err
		}
		return instance, nil
	}

	// Check for duplicate registration
	if a.container.hasService(typeNameStr) {
		return fmt.Errorf("%w: %s", ErrDuplicate, typeNameStr)
	}

	// Create appropriate service wrapper
	var svc serviceWrapper
	switch {
	case scope == scopeTransient:
		svc = newTransientAny(typeNameStr, typeNameStr, wrappedProvider)
	case !lazy:
		svc = newEagerSingletonAny(typeNameStr, typeNameStr, wrappedProvider, nil, nil)
	default:
		svc = newLazySingletonAny(typeNameStr, typeNameStr, wrappedProvider, nil, nil)
	}

	a.container.register(typeNameStr, svc)
	return nil
}

// registerInstance registers a pre-built instance using reflection.
func (a *App) registerInstance(instance any) error {
	instanceType := reflect.TypeOf(instance)
	if instanceType == nil {
		return fmt.Errorf("%w: instance cannot be nil", ErrInvalidProvider)
	}

	typeNameStr := typeName(instanceType)

	// Check for duplicate registration
	if a.container.hasService(typeNameStr) {
		return fmt.Errorf("%w: %s", ErrDuplicate, typeNameStr)
	}

	svc := newInstanceServiceAny(typeNameStr, typeNameStr, instance, nil, nil)
	a.container.register(typeNameStr, svc)
	return nil
}

// loadConfig loads the configuration from all sources.
func (a *App) loadConfig() error {
	if a.configManager == nil {
		return nil
	}
	return a.configManager.Load()
}

// getSortedServiceNames returns service names in sorted order for deterministic iteration.
func (a *App) getSortedServiceNames() []string {
	a.container.mu.RLock()
	names := make([]string, 0, len(a.container.services))
	for name := range a.container.services {
		names = append(names, name)
	}
	a.container.mu.RUnlock()
	sort.Strings(names)
	return names
}

// collectProviderConfigs iterates registered services, collects config from ConfigProvider
// implementers, detects key collisions, registers provider flags with ConfigManager,
// validates required fields, and registers ProviderValues.
func (a *App) collectProviderConfigs() error {
	keyOwners := make(map[string]string)
	var collisionErrors []error

	// Iterate in sorted order for deterministic dependency graph recording
	for _, typeName := range a.getSortedServiceNames() {
		a.container.mu.RLock()
		svc, exists := a.container.services[typeName]
		a.container.mu.RUnlock()
		if !exists {
			continue
		}

		wrapper, ok := svc.(serviceWrapper)
		if !ok || wrapper.isTransient() {
			continue
		}

		// Use resolveByName() instead of getInstance() to ensure dependencies are recorded
		instance, err := a.container.resolveByName(typeName, nil)
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

	return a.registerProviderFlags()
}

// registerProviderFlags registers collected provider flags with ConfigManager and validates.
func (a *App) registerProviderFlags() error {
	if a.configManager == nil {
		return nil
	}

	var validationErrors []error
	for _, entry := range a.providerConfigs {
		if err := a.configManager.RegisterProviderFlags(entry.namespace, entry.flags); err != nil {
			return err
		}
		errs := a.configManager.ValidateProviderFlags(entry.namespace, entry.flags)
		validationErrors = append(validationErrors, errs...)
	}

	if len(validationErrors) > 0 {
		return errors.Join(validationErrors...)
	}

	pv := &ProviderValues{v: a.configManager.Viper()}
	return a.registerInstance(pv)
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

	// Collect provider configs from registered services
	if err := a.collectProviderConfigs(); err != nil {
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
	graph := a.container.getGraph()
	services := make(map[string]serviceWrapper)
	a.container.mu.RLock()
	for k, v := range a.container.services {
		if w, ok := v.(serviceWrapper); ok {
			services[k] = w
		}
	}
	a.container.mu.RUnlock()

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
				if startErr := svc.start(ctx); startErr != nil {
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
	graph := a.container.getGraph()
	services := make(map[string]serviceWrapper)
	a.container.mu.RLock()
	for k, v := range a.container.services {
		if w, ok := v.(serviceWrapper); ok {
			services[k] = w
		}
	}
	a.container.mu.RUnlock()

	startupOrder, err := ComputeStartupOrder(graph, services)
	if err != nil {
		close(done)
		// Should not happen if Build passed, unless graph changed (impossible after Build)
		return err
	}
	shutdownOrder := ComputeShutdownOrder(startupOrder)

	lastErr := a.stopServices(ctx, shutdownOrder, services)

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

	return lastErr
}

// stopServices stops services sequentially with per-hook timeout and blame logging.
func (a *App) stopServices(
	ctx context.Context,
	order [][]string,
	services map[string]serviceWrapper,
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
				errCh <- svc.stop(hookCtx)
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
