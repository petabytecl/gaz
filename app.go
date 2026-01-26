package gaz

import (
	"time"
)

// AppOptions configuration for App.
type AppOptions struct {
	ShutdownTimeout time.Duration
}

// AppOption configures AppOptions.
type AppOption func(*AppOptions)

// WithShutdownTimeout sets the timeout for graceful shutdown.
func WithShutdownTimeout(d time.Duration) AppOption {
	return func(o *AppOptions) {
		o.ShutdownTimeout = d
	}
}

// App is the application runtime wrapper.
// It orchestrates dependency injection, lifecycle management, and signal handling.
type App struct {
	container *Container
	opts      AppOptions
}

// NewApp creates a new App with the given container and options.
func NewApp(c *Container, opts ...AppOption) *App {
	options := AppOptions{
		ShutdownTimeout: 30 * time.Second, // Default
	}
	for _, opt := range opts {
		opt(&options)
	}

	return &App{
		container: c,
		opts:      options,
	}
}
