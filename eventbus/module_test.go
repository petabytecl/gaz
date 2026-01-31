package eventbus

import (
	"log/slog"
	"os"
	"testing"

	"github.com/petabytecl/gaz/di"
	"github.com/stretchr/testify/require"
)

func TestNewModule(t *testing.T) {
	t.Run("zero arguments works with defaults", func(t *testing.T) {
		c := di.New()

		// Register logger prerequisite
		logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
		err := di.For[*slog.Logger](c).Instance(logger)
		require.NoError(t, err)

		// Register module
		moduleFn := NewModule()
		err = moduleFn(c)
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

		// Register logger prerequisite
		logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
		err := di.For[*slog.Logger](c).Instance(logger)
		require.NoError(t, err)

		err = mod(c)
		require.NoError(t, err)
	})

	t.Run("succeeds even without logger", func(t *testing.T) {
		// Module gracefully handles missing logger (returns nil, not error)
		c := di.New()

		moduleFn := NewModule()
		err := moduleFn(c)
		require.NoError(t, err) // Module doesn't fail, just doesn't validate
	})
}
