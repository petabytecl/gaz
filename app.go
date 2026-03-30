package gaz

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"os"
	"reflect"
	"sync"
	"time"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"

	"github.com/petabytecl/gaz/config"
	cfgviper "github.com/petabytecl/gaz/config/viper"
	"github.com/petabytecl/gaz/cron"
	"github.com/petabytecl/gaz/eventbus"
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
	loggerInitialized        bool      // tracks if initializeLogger was called
	logCloser                io.Closer // logger file handle closer (nil for stdout/stderr)

	// Worker management - nil until Build() is called
	workerMgr *worker.Manager

	// Cron scheduler - nil until Build() is called
	scheduler  *cron.Scheduler
	cronCtx    context.Context    // lifecycle context for cron scheduler
	cronCancel context.CancelFunc // cancels cron scheduler context

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
