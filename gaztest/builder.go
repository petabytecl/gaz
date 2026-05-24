package gaztest

import (
	"errors"
	"fmt"
	"reflect"
	"time"

	"github.com/petabytecl/gaz"
	"github.com/petabytecl/gaz/di"
)

// DefaultTimeout is the default timeout for test apps (5 seconds).
const DefaultTimeout = 5 * time.Second

// TB is a subset of testing.TB required by gaztest.
// This interface is compatible with both *testing.T and *testing.B.
type TB interface {
	Logf(string, ...any)
	Errorf(string, ...any)
	Fatalf(string, ...any)
	FailNow()
	Cleanup(func())
	Helper()
}

// replacement stores a mock instance to replace a registered type.
type replacement struct {
	typeName string
	instance any
}

type testModule struct {
	apply func(*gaz.App)
}

// Builder configures a test application.
// Create with New(t), configure with fluent methods, and call Build() to get the App.
type Builder struct {
	tb           TB
	timeout      time.Duration
	replacements []replacement
	baseApp      *gaz.App
	modules      []testModule
	configMap    map[string]any
	errs         []error
}

// New creates a new Builder for configuring test apps.
// The default timeout is 5 seconds, suitable for most test scenarios.
func New(tb TB) *Builder {
	return &Builder{
		tb:      tb,
		timeout: DefaultTimeout,
	}
}

// WithTimeout sets a custom timeout for start/stop operations.
// The default timeout is 5 seconds.
func (b *Builder) WithTimeout(d time.Duration) *Builder {
	b.timeout = d
	return b
}

// WithApp sets a base gaz.App to use for the test.
// This allows testing with pre-registered services that can be replaced with mocks.
// The app does not need to be pre-built; gaztest.Build() will call Build() after
// applying replacements. If the app is already built and no replacements are needed,
// Build() is idempotent and safe to call again.
//
// Note: WithApp cannot be used together with WithModules - they are mutually exclusive.
// Use either WithApp for pre-built apps or WithModules for module-based registration.
func (b *Builder) WithApp(app *gaz.App) *Builder {
	b.baseApp = app
	return b
}

// WithModules registers the given DI modules with the test app during build.
// Use WithGazModules for feature packages that return gaz.Module values.
//
// Note: WithModules cannot be used together with WithApp - they are mutually exclusive.
// Build() will return an error if both are used.
//
// Example:
//
//	app, err := gaztest.New(t).
//	    WithModules(di.NewModuleFunc("worker", registerWorker)).
//	    Build()
func (b *Builder) WithModules(modules ...di.Module) *Builder {
	for _, module := range modules {
		adapted, err := adaptDIModule(module)
		if err != nil {
			b.errs = append(b.errs, err)
			continue
		}
		b.modules = append(b.modules, adapted)
	}
	return b
}

// WithGazModules registers the given gaz modules with the test app during build.
// This is for feature modules that are normally applied through app.Use().
//
// Note: WithGazModules cannot be used together with WithApp - they are mutually exclusive.
// Build() will return an error if both are used.
//
// Example:
//
//	app, err := gaztest.New(t).
//	    WithGazModules(workermod.New()).
//	    Build()
func (b *Builder) WithGazModules(modules ...gaz.Module) *Builder {
	for _, module := range modules {
		adapted, err := adaptGazModule(module)
		if err != nil {
			b.errs = append(b.errs, err)
			continue
		}
		b.modules = append(b.modules, adapted)
	}
	return b
}

func adaptDIModule(module di.Module) (testModule, error) {
	if module == nil {
		return testModule{}, errors.New("gaztest: WithModules: module cannot be nil")
	}

	return testModule{
		apply: func(app *gaz.App) {
			app.UseDI(module)
		},
	}, nil
}

func adaptGazModule(module gaz.Module) (testModule, error) {
	if module == nil {
		return testModule{}, errors.New("gaztest: WithGazModules: module cannot be nil")
	}

	return testModule{
		apply: func(app *gaz.App) {
			app.Use(module)
		},
	}, nil
}

// WithConfigMap injects raw config values for testing.
// The values are merged into the app's configuration via viper.MergeConfigMap
// before the app is built.
//
// Example:
//
//	app, err := gaztest.New(t).
//	    WithConfigMap(map[string]any{
//	        "worker.pool_size": 2,
//	        "health.port": 0,
//	    }).
//	    Build()
func (b *Builder) WithConfigMap(values map[string]any) *Builder {
	b.configMap = values
	return b
}

// Replace registers a mock instance to replace a type in the container.
// The type to replace is inferred from the instance using reflection.
//
// Replace must be called before Build() and requires that:
//  1. The instance is not nil
//  2. The type is registered in the container (via WithApp)
//
// If these conditions are not met, Build() will return an error.
func (b *Builder) Replace(instance any) *Builder {
	if instance == nil {
		b.errs = append(b.errs, errors.New("gaztest: Replace: instance cannot be nil"))
		return b
	}

	instanceType := reflect.TypeOf(instance)
	typeName := di.TypeNameReflect(instanceType)

	b.replacements = append(b.replacements, replacement{
		typeName: typeName,
		instance: instance,
	})
	return b
}

// Build creates the test app with all configured replacements.
// It returns an error if:
//   - Any Replace() call had nil instance
//   - A replacement type is not registered in the container
//   - Both WithApp and WithModules are used (mutually exclusive)
//   - The underlying gaz.App fails to build
//
// Replacements are applied before Build() to ensure they respect the post-Build guard.
// Build registers t.Cleanup() to automatically stop the app when the test completes.
func (b *Builder) Build() (*App, error) {
	// Check for accumulated errors from Replace() calls
	if len(b.errs) > 0 {
		return nil, errors.Join(b.errs...)
	}

	// Check for conflicting patterns: cannot use WithApp and WithModules together
	if b.baseApp != nil && len(b.modules) > 0 {
		b.errs = append(b.errs, errors.New("gaztest: cannot use WithApp and WithModules together"))
		return nil, errors.Join(b.errs...)
	}

	var gazApp *gaz.App

	// Use base app or create new one
	if b.baseApp != nil {
		gazApp = b.baseApp
	} else {
		gazApp = gaz.New(
			gaz.WithShutdownTimeout(b.timeout),
			gaz.WithPerHookTimeout(b.timeout),
		)

		// Register modules if provided
		for _, m := range b.modules {
			m.apply(gazApp)
		}
	}

	// Apply config map if provided
	if b.configMap != nil {
		if err := gazApp.MergeConfigMap(b.configMap); err != nil {
			return nil, fmt.Errorf("gaztest: failed to merge config map: %w", err)
		}
	}

	// Apply replacements to container
	for _, r := range b.replacements {
		if !gazApp.Container().HasService(r.typeName) {
			return nil, fmt.Errorf("gaztest: Replace: type %s not registered in container", r.typeName)
		}
		// Create replacement service and register it
		svc := di.NewInstanceServiceAny(r.typeName, r.typeName, r.instance)
		if err := gazApp.Container().ReplaceService(r.typeName, svc); err != nil {
			return nil, fmt.Errorf("gaztest: replace %s: %w", r.typeName, err)
		}
	}

	// Build and validate if not already built
	if err := gazApp.Build(); err != nil {
		return nil, err
	}

	app := &App{
		app:     gazApp,
		tb:      b.tb,
		timeout: b.timeout,
	}

	// Register automatic cleanup
	b.tb.Cleanup(func() {
		app.cleanup()
	})

	return app, nil
}
