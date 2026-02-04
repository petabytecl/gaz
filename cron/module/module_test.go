package module

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/petabytecl/gaz"
	"github.com/petabytecl/gaz/cron"
)

func TestNew(t *testing.T) {
	t.Run("creates valid module", func(t *testing.T) {
		mod := New()
		require.NotNil(t, mod)
	})

	t.Run("integrates with gaz.App", func(t *testing.T) {
		app := gaz.New()
		app.Use(New())

		err := app.Build()
		require.NoError(t, err)

		// Verify Scheduler is registered and resolvable
		sched, err := gaz.Resolve[*cron.Scheduler](app.Container())
		require.NoError(t, err)
		require.NotNil(t, sched)
	})
}
