package gaz

import (
	"fmt"
	"log/slog"
	"reflect"

	"github.com/petabytecl/gaz/cron"
	"github.com/petabytecl/gaz/di"
	"github.com/petabytecl/gaz/eventbus"
	"github.com/petabytecl/gaz/logger"
	"github.com/petabytecl/gaz/worker"
)

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
		a.Logger, a.logCloser = logger.NewLoggerWithCloser(a.opts.LoggerConfig)
	} else {
		// Logger module provided config - use it
		a.Logger, a.logCloser = logger.NewLoggerWithCloser(&cfg)
	}

	// Set as process-wide default logger (B12: explicit, not hidden side-effect)
	logger.SetGlobal(a.Logger)

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
	subs, err := newRuntimeSubsystems(runtimeSubsystemsDeps{
		logger:          a.Logger,
		container:       a.container,
		shutdownTimeout: a.opts.ShutdownTimeout,
		stopFunc:        a.Stop,
	})
	if err != nil {
		return err
	}

	a.workerMgr = subs.workerMgr
	a.scheduler = subs.scheduler
	a.cronCtx = subs.cronCtx
	a.cronCancel = subs.cronCancel
	a.eventBus = subs.eventBus
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
	if err := a.container.Register(typeNameStr, svc); err != nil {
		return fmt.Errorf("gaz: register %s: %w", typeNameStr, err)
	}
	return nil
}

// workerType is cached for efficient interface checks during discovery.
//
//nolint:gochecknoglobals // Package-level for reflect type caching.
var workerType = reflect.TypeOf((*worker.Worker)(nil)).Elem()

// eventBusType is cached so the lifecycle plan can recognize the framework
// EventBus registration without resolving it as a user worker.
//
//nolint:gochecknoglobals // Package-level for reflect type caching.
var eventBusType = reflect.TypeOf((*eventbus.EventBus)(nil))

// cronJobType is cached for efficient interface checks during discovery.
//
//nolint:gochecknoglobals // Package-level for reflect type caching.
var cronJobType = reflect.TypeOf((*cron.CronJob)(nil)).Elem()

// Build validates all registrations and instantiates eager services.
// It aggregates all errors and returns them using errors.Join.
// Build is idempotent - calling it multiple times after success returns nil.
func (a *App) Build() error {
	a.mu.Lock()
	defer a.mu.Unlock()

	if a.built {
		return nil
	}

	if err := a.loadConfig(); err != nil {
		return err
	}

	if err := a.runBuildPhases(); err != nil {
		return err
	}

	a.built = true
	return nil
}
