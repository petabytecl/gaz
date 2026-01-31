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
		moduleFn := NewModule()
		err := moduleFn(c)
		require.NoError(t, err)
	})

	t.Run("returns valid function", func(t *testing.T) {
		mod := NewModule()
		require.NotNil(t, mod)
	})

	t.Run("accepts options", func(t *testing.T) {
		// Test that options can be passed (even if none are currently defined)
		mod := NewModule()
		require.NotNil(t, mod)

		c := di.New()
		err := mod(c)
		require.NoError(t, err)
	})
}
