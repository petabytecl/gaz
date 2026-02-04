package cron

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/petabytecl/gaz/di"
)

// Module registers cron infrastructure into the DI container.
// It provides a *Scheduler that can schedule and execute cron jobs.
//
// The logger is optional - if not registered, slog.Default() is used.
// The di.Container is used as the Resolver since it implements ResolveByName.
//
// For CLI/App integration with flags, use the cron/module subpackage:
//
//	import cronmod "github.com/petabytecl/gaz/cron/module"
//	app.Use(cronmod.New())
func Module(c *di.Container) error {
	if err := di.For[*Scheduler](c).Provider(func(c *di.Container) (*Scheduler, error) {
		// Logger is optional - use default if not registered
		logger := slog.Default()
		if l, err := di.Resolve[*slog.Logger](c); err == nil {
			logger = l
		}

		// di.Container implements Resolver interface via ResolveByName
		// Use context.Background() for standalone DI usage
		return NewScheduler(c, context.Background(), logger), nil
	}); err != nil {
		return fmt.Errorf("register scheduler: %w", err)
	}
	return nil
}
