package gaz

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"reflect"

	"github.com/petabytecl/gaz/config"
	cfgviper "github.com/petabytecl/gaz/config/viper"
	"github.com/petabytecl/gaz/cron"
	"github.com/petabytecl/gaz/di"
	"github.com/petabytecl/gaz/eventbus"
	"github.com/petabytecl/gaz/health"
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

	// Apply config module flags if present (--config, --env-prefix, --config-strict)
	if err := a.applyConfigFlags(); err != nil {
		return err
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

// applyConfigFlags reads --config, --env-prefix, --config-strict flags and
// recreates the config manager with appropriate options.
// This is called at the start of loadConfig() and only applies if the
// config module registered these flags.
func (a *App) applyConfigFlags() error {
	if a.cobraCmd == nil {
		return nil
	}

	flags := a.cobraCmd.Flags()

	// Only apply if config module registered --config flag
	configFlag := flags.Lookup("config")
	if configFlag == nil {
		return nil
	}

	var opts []config.Option
	opts = append(opts, config.WithBackend(cfgviper.New()))

	// --config flag: explicit config file path
	configPath := configFlag.Value.String()
	if configPath != "" {
		// Explicit config file - validate exists
		if _, statErr := os.Stat(configPath); statErr != nil {
			return fmt.Errorf("config: file not found: %s", configPath)
		}
		opts = append(opts, config.WithConfigFile(configPath))
	} else {
		// Auto-search: cwd first, then XDG config dir
		xdgConfig := os.Getenv("XDG_CONFIG_HOME")
		if xdgConfig == "" {
			if home, homeErr := os.UserHomeDir(); homeErr == nil {
				xdgConfig = filepath.Join(home, ".config")
			}
		}
		searchPaths := []string{"."}
		if xdgConfig != "" {
			appName := a.cobraCmd.Root().Name()
			if appName != "" {
				searchPaths = append(searchPaths, filepath.Join(xdgConfig, appName))
			}
		}
		opts = append(opts, config.WithSearchPaths(searchPaths...))
	}

	// --env-prefix flag
	if envPrefixFlag := flags.Lookup("env-prefix"); envPrefixFlag != nil {
		envPrefix := envPrefixFlag.Value.String()
		if envPrefix != "" {
			opts = append(opts, config.WithEnvPrefix(envPrefix))
		}
	}

	// --config-strict flag
	if strictFlag := flags.Lookup("config-strict"); strictFlag != nil {
		if strictFlag.Value.String() == "true" {
			a.strictConfig = true
		} else if strictFlag.Value.String() == "false" {
			a.strictConfig = false
		}
	}

	// Recreate config manager with collected options
	a.configMgr = config.New(opts...)

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
