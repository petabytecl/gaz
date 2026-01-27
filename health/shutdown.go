package health

import (
	"context"
	"errors"
	"sync/atomic"
)

// ShutdownCheck is a readiness checker that fails when the application is shutting down.
type ShutdownCheck struct {
	shuttingDown atomic.Bool
}

// NewShutdownCheck creates a new ShutdownCheck.
func NewShutdownCheck() *ShutdownCheck {
	return &ShutdownCheck{}
}

// Check implements the CheckFunc signature.
func (c *ShutdownCheck) Check(_ context.Context) error {
	if c.shuttingDown.Load() {
		return errors.New("application is shutting down")
	}
	return nil
}

// MarkShuttingDown sets the state to shutting down, causing Check to return an error.
func (c *ShutdownCheck) MarkShuttingDown() {
	c.shuttingDown.Store(true)
}
