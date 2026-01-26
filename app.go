// Package gaz provides a simple, type-safe dependency injection container
// with lifecycle management for Go applications.
package gaz

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/signal"
	"reflect"
	"sync"
	"syscall"
	"time"
)

const defaultShutdownTimeout = 30 * time.Second

// AppOptions configuration for App.
type AppOptions struct {
	ShutdownTimeout time.Duration
}

// AppOption configures AppOptions.
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
	built       bool    // tracks if Build() was called
	buildErrors []error // collects registration errors for Build()

	mu      sync.Mutex
	running bool
	stopCh  chan struct{}
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
		},
	}
	for _, opt := range opts {
		opt(app)
	}
	return app
}

// NewApp creates a new App with the given container and options.
// Deprecated: Use New() with fluent provider methods instead.
func NewApp(c *Container, opts ...AppOption) *App {
	options := AppOptions{
		ShutdownTimeout: defaultShutdownTimeout,
	}
	for _, opt := range opts {
		opt(&options)
	}

	return &App{
		container: c,
		opts:      options,
	}
}

// Container returns the underlying container.
// This is useful for advanced use cases or testing.
func (a *App) Container() *Container {
	return a.container
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
		return fmt.Errorf("%w: provider must accept exactly one argument (*Container)", ErrInvalidProvider)
	}
	containerType := reflect.TypeOf((*Container)(nil))
	if providerType.In(0) != containerType {
		return fmt.Errorf("%w: provider argument must be *Container", ErrInvalidProvider)
	}

	// Validate output: must return (T) or (T, error)
	numOut := providerType.NumOut()
	if numOut < 1 || numOut > 2 {
		return fmt.Errorf("%w: provider must return (T) or (T, error)", ErrInvalidProvider)
	}
	if numOut == 2 {
		errorType := reflect.TypeOf((*error)(nil)).Elem()
		if !providerType.Out(1).Implements(errorType) {
			return fmt.Errorf("%w: second return value must be error", ErrInvalidProvider)
		}
	}

	returnType := providerType.Out(0)
	typeName := returnType.String()

	// Create a wrapped provider that handles both (T) and (T, error) signatures
	providerValue := reflect.ValueOf(provider)
	wrappedProvider := func(c *Container) (any, error) {
		results := providerValue.Call([]reflect.Value{reflect.ValueOf(c)})
		instance := results[0].Interface()
		if numOut == 2 && !results[1].IsNil() {
			return nil, results[1].Interface().(error)
		}
		return instance, nil
	}

	// Check for duplicate registration
	if a.container.hasService(typeName) {
		return fmt.Errorf("%w: %s", ErrDuplicate, typeName)
	}

	// Create appropriate service wrapper
	var svc serviceWrapper
	switch {
	case scope == scopeTransient:
		svc = newTransientAny(typeName, typeName, wrappedProvider)
	case !lazy:
		svc = newEagerSingletonAny(typeName, typeName, wrappedProvider, nil, nil)
	default:
		svc = newLazySingletonAny(typeName, typeName, wrappedProvider, nil, nil)
	}

	a.container.register(typeName, svc)
	return nil
}

// registerInstance registers a pre-built instance using reflection.
func (a *App) registerInstance(instance any) error {
	instanceType := reflect.TypeOf(instance)
	if instanceType == nil {
		return fmt.Errorf("%w: instance cannot be nil", ErrInvalidProvider)
	}

	typeName := instanceType.String()

	// Check for duplicate registration
	if a.container.hasService(typeName) {
		return fmt.Errorf("%w: %s", ErrDuplicate, typeName)
	}

	svc := newInstanceServiceAny(typeName, typeName, instance, nil, nil)
	a.container.register(typeName, svc)
	return nil
}

// Run executes the application lifecycle.
// It builds the container, starts services in order, and waits for a signal or stop call.
func (a *App) Run(ctx context.Context) error {
	if err := a.container.Build(); err != nil {
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

	// Start services layer by layer
	for _, layer := range startupOrder {
		var wg sync.WaitGroup
		errCh := make(chan error, len(layer))

		for _, name := range layer {
			svc := services[name]
			wg.Add(1)
			go func() {
				defer wg.Done()
				if startErr := svc.start(ctx); startErr != nil {
					errCh <- fmt.Errorf("starting service %s: %w", name, startErr)
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

	// Signal handling
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, os.Interrupt, syscall.SIGTERM)
	defer signal.Stop(sigCh)

	// Block until stopped
	select {
	case <-ctx.Done():
		// Context cancelled, initiate shutdown
		shutdownCtx, cancel := context.WithTimeout(context.Background(), a.opts.ShutdownTimeout)
		defer cancel()
		return a.Stop(shutdownCtx)
	case <-sigCh:
		// Signal received, initiate shutdown
		shutdownCtx, cancel := context.WithTimeout(context.Background(), a.opts.ShutdownTimeout)
		defer cancel()
		return a.Stop(shutdownCtx)
	case <-a.stopCh:
		// Stopped externally (Stop() called)
		return nil
	}
}

// Stop initiates graceful shutdown of the application.
func (a *App) Stop(ctx context.Context) error {
	a.mu.Lock()
	if !a.running {
		a.mu.Unlock()
		return nil
	}
	// We do NOT set running=false here. It happens in Run's defer.
	// We close stopCh to signal Run to exit, BUT we do work first.
	// Wait, if we do work first, Run is still waiting.
	// After work, we close stopCh, Run returns.
	a.mu.Unlock()

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
		// Should not happen if Build passed, unless graph changed (impossible after Build)
		return err
	}
	shutdownOrder := ComputeShutdownOrder(startupOrder)

	var lastErr error

	// Stop services layer by layer
	for _, layer := range shutdownOrder {
		var wg sync.WaitGroup
		errCh := make(chan error, len(layer))

		for _, name := range layer {
			svc := services[name]
			wg.Add(1)
			go func() {
				defer wg.Done()
				if stopErr := svc.stop(ctx); stopErr != nil {
					errCh <- fmt.Errorf("stopping service %s: %w", name, stopErr)
				}
			}()
		}
		wg.Wait()
		close(errCh)

		// Collect errors but continue shutdown
		for shutdownErr := range errCh {
			if lastErr == nil {
				lastErr = shutdownErr
			} else {
				lastErr = errors.Join(lastErr, shutdownErr)
			}
		}
	}

	// Signal Run to exit
	a.mu.Lock()
	select {
	case <-a.stopCh:
		// Already closed
	default:
		close(a.stopCh)
	}
	a.mu.Unlock()

	return lastErr
}
