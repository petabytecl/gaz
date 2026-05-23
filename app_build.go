package gaz

import (
	"context"
	"errors"
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
			if err := a.Stop(ctx); err != nil {
				log.Warn("critical-fail shutdown completed with errors", slog.Any("error", err))
			}
		}()
	})

	// Scheduler with cancellable context
	a.cronCtx, a.cronCancel = context.WithCancel(context.Background())
	a.scheduler = cron.NewScheduler(a.container, a.cronCtx, log)

	// EventBus
	a.eventBus = eventbus.New(log)

	// Register EventBus in container
	if err := For[*eventbus.EventBus](a.container).Instance(a.eventBus); err != nil {
		return fmt.Errorf("register eventbus: %w", err)
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
	if err := a.autoRegisterHealth(); err != nil {
		errs = append(errs, err)
	}

	// Delegate to container.Build() for eager instantiation
	if err := a.container.Build(); err != nil {
		errs = append(errs, err)
	}

	if len(errs) == 0 {
		plan, planErr := newLifecyclePlan(a.container)
		if planErr != nil {
			errs = append(errs, planErr)
		} else {
			a.cachedLifecyclePlan = plan
		}
	}

	if len(errs) == 0 {
		plan := a.cachedLifecyclePlan
		errs = append(errs, plan.registerRuntimeParticipants(
			a.container,
			a.workerMgr,
			a.eventBus,
			a.scheduler,
			a.getLogger(),
		)...)
	}

	if len(errs) > 0 {
		return errors.Join(errs...)
	}

	a.built = true
	return nil
}
