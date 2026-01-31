package config

import (
	"testing"

	"github.com/petabytecl/gaz/di"
	"github.com/stretchr/testify/require"
)

func TestNewModule(t *testing.T) {
	t.Run("zero arguments works with defaults", func(t *testing.T) {
		c := di.New()

		// Register module
		module := NewModule()
		err := module.Register(c)
		require.NoError(t, err)
	})

	t.Run("returns valid di.Module", func(t *testing.T) {
		mod := NewModule()
		require.NotNil(t, mod)
		require.Equal(t, "config", mod.Name())
	})

	t.Run("accepts options", func(t *testing.T) {
		// Test that options can be passed (even if none are currently defined)
		mod := NewModule()
		require.NotNil(t, mod)

		c := di.New()
		err := mod.Register(c)
		require.NoError(t, err)
	})
}
