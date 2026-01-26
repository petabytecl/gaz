package gaz

import "context"

// HookFunc is a function that performs a lifecycle action.
type HookFunc func(context.Context) error

// HookConfig holds configuration for lifecycle hooks.
type HookConfig struct {
	// We can add fields here later, e.g., Timeout time.Duration
}

// HookOption configures a lifecycle hook.
type HookOption func(*HookConfig)

// Starter is an interface for services that need to perform action on startup.
// If a service implements this, OnStart will be called automatically after creation.
type Starter interface {
	OnStart(context.Context) error
}

// Stopper is an interface for services that need to perform action on shutdown.
// If a service implements this, OnStop will be called automatically during container shutdown.
type Stopper interface {
	OnStop(context.Context) error
}
