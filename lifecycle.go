package gaz

import (
	"context"
	"time"
)

// HookFunc is a function that performs a lifecycle action.
type HookFunc func(context.Context) error

// HookConfig holds configuration for lifecycle hooks.
type HookConfig struct {
	// Timeout is the per-hook timeout for shutdown. If zero, uses App's PerHookTimeout.
	Timeout time.Duration
}

// WithHookTimeout sets a custom timeout for this specific hook.
// If not set, the hook uses the App's default PerHookTimeout.
func WithHookTimeout(d time.Duration) HookOption {
	return func(cfg *HookConfig) {
		cfg.Timeout = d
	}
}

// HookOption configures a lifecycle hook.
type HookOption func(*HookConfig)

// Starter is an interface for services that need to perform action on startup.
// Implementing this interface is the sole mechanism for lifecycle participation.
// OnStart is called automatically after container Build() when the service is
// first instantiated. Hooks are called in dependency order: dependencies start first.
//
// This interface is auto-detected by the DI container. No registration of lifecycle
// hooks is needed - simply implement the interface.
type Starter interface {
	OnStart(context.Context) error
}

// Stopper is an interface for services that need to perform action on shutdown.
// Implementing this interface is the sole mechanism for lifecycle participation.
// OnStop is called automatically during graceful shutdown. Hooks are called in
// reverse dependency order: dependents stop first, then their dependencies.
//
// This interface is auto-detected by the DI container. No registration of lifecycle
// hooks is needed - simply implement the interface.
type Stopper interface {
	OnStop(context.Context) error
}
