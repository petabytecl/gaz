package config

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/petabytecl/gaz/di"
)

func TestNewModule(t *testing.T) {
	t.Run("is compatibility no-op", func(t *testing.T) {
		c := di.New()

		module := NewModule()
		err := module.Register(c)
		require.NoError(t, err)
		require.Empty(t, c.List())
	})

	t.Run("returns valid di.Module", func(t *testing.T) {
		mod := NewModule()
		require.NotNil(t, mod)
		require.Equal(t, ModuleName, mod.Name())
	})

	t.Run("ignores nil options", func(t *testing.T) {
		mod := NewModule(nil)
		require.NotNil(t, mod)

		c := di.New()
		err := mod.Register(c)
		require.NoError(t, err)
		require.Empty(t, c.List())
	})
}
