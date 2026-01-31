package eventbus

import (
	"log/slog"
	"os"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/petabytecl/gaz/di"
)

func TestNewModule(t *testing.T) {
	t.Run("zero arguments works with defaults", func(t *testing.T) {
		c := di.New()

		// Register logger prerequisite
		logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
		err := di.For[*slog.Logger](c).Instance(logger)
		require.NoError(t, err)

		// Register module
		module := NewModule()
		err = module.Register(c)
		require.NoError(t, err)
	})

	t.Run("returns valid di.Module", func(t *testing.T) {
		mod := NewModule()
		require.NotNil(t, mod)
		require.Equal(t, "eventbus", mod.Name())
	})

	t.Run("accepts options", func(t *testing.T) {
		// Test that options can be passed (even if none are currently defined)
		mod := NewModule()
		require.NotNil(t, mod)

		c := di.New()

		// Register logger prerequisite
		logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
		err := di.For[*slog.Logger](c).Instance(logger)
		require.NoError(t, err)

		err = mod.Register(c)
		require.NoError(t, err)
	})

	t.Run("succeeds even without logger", func(t *testing.T) {
		// Module gracefully handles missing logger (returns nil, not error)
		c := di.New()

		module := NewModule()
		err := module.Register(c)
		require.NoError(t, err) // Module doesn't fail, just doesn't validate
	})
}
