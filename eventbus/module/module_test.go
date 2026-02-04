package module

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/petabytecl/gaz/di"
	"github.com/petabytecl/gaz/eventbus"
)

func TestNew(t *testing.T) {
	t.Run("creates valid module", func(t *testing.T) {
		mod := New()
		require.NotNil(t, mod)
	})

	// Note: gaz.App auto-registers *eventbus.EventBus in initializeSubsystems(),
	// so using eventbusmod.New() with gaz.App is redundant. The module is intended
	// for di.Container usage or when the auto-registration behavior changes.
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
