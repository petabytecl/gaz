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

// Builder configures a test application.
// Create with New(t), configure with fluent methods, and call Build() to get the App.
type Builder struct {
	tb           TB
	timeout      time.Duration
	replacements []replacement
	baseApp      *gaz.App
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
// The base app should have been built already.
func (b *Builder) WithApp(app *gaz.App) *Builder {
	b.baseApp = app
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
//   - The underlying gaz.App fails to build
//
// Build registers t.Cleanup() to automatically stop the app when the test completes.
func (b *Builder) Build() (*App, error) {
	// Check for accumulated errors from Replace() calls
	if len(b.errs) > 0 {
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
	}

	// Apply replacements to container
	for _, r := range b.replacements {
		if !gazApp.Container().HasService(r.typeName) {
			return nil, fmt.Errorf("gaztest: Replace: type %s not registered in container", r.typeName)
		}
		// Create replacement service and register it
		svc := di.NewInstanceServiceAny(r.typeName, r.typeName, r.instance, nil, nil)
		gazApp.Container().Register(r.typeName, svc)
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
