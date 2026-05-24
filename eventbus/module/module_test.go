package module

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/petabytecl/gaz"
	"github.com/petabytecl/gaz/di"
	"github.com/petabytecl/gaz/eventbus"
)

func TestNew(t *testing.T) {
	t.Run("creates valid module", func(t *testing.T) {
		mod := New()
		require.NotNil(t, mod)
		require.Equal(t, eventbus.ModuleName, mod.Name())
	})

	t.Run("works with gaz.App", func(t *testing.T) {
		app := gaz.New()
		app.Use(New())

		require.NoError(t, app.Build())
		require.NotNil(t, app.EventBus())

		bus, err := gaz.Resolve[*eventbus.EventBus](app.Container())
		require.NoError(t, err)
		require.Same(t, app.EventBus(), bus)

		bus.Close()
	})

	t.Run("works with di.Container directly", func(t *testing.T) {
		c := di.New()

		// Apply module directly to container
		err := eventbus.Module(c)
		require.NoError(t, err)

		// Build container
		err = c.Build()
		require.NoError(t, err)

		// Verify EventBus is registered and resolvable
		bus, err := di.Resolve[*eventbus.EventBus](c)
		require.NoError(t, err)
		require.NotNil(t, bus)

		// Cleanup
		bus.Close()
	})
}
