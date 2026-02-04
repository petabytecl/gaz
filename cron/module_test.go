package cron

import (
	"log/slog"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/petabytecl/gaz/di"
)

func TestModule(t *testing.T) {
	t.Run("registers Scheduler", func(t *testing.T) {
		c := di.New()

		// Register logger prerequisite
		err := di.For[*slog.Logger](c).Instance(slog.Default())
		require.NoError(t, err)

		// Apply module
		err = Module(c)
		require.NoError(t, err)

		// Build container
		err = c.Build()
		require.NoError(t, err)

		// Verify Scheduler resolves
		sched, err := di.Resolve[*Scheduler](c)
		require.NoError(t, err)
		require.NotNil(t, sched)
	})

	t.Run("uses slog.Default when logger not registered", func(t *testing.T) {
		c := di.New()

		// Apply module without registering logger
		err := Module(c)
		require.NoError(t, err)

		// Build container
		err = c.Build()
		require.NoError(t, err)

		// Scheduler should still resolve (using slog.Default)
		sched, err := di.Resolve[*Scheduler](c)
		require.NoError(t, err)
		require.NotNil(t, sched)
	})
}
