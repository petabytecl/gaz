package di

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
// If a service implements this, OnStart will be called automatically after creation.
type Starter interface {
	OnStart(context.Context) error
}

// Stopper is an interface for services that need to perform action on shutdown.
// If a service implements this, OnStop will be called automatically during container shutdown.
type Stopper interface {
	OnStop(context.Context) error
}
