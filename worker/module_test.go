package worker

import (
	"log/slog"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/petabytecl/gaz/di"
)

func TestNewModule(t *testing.T) {
	t.Run("zero arguments works with defaults", func(t *testing.T) {
		c := di.New()

		// Register logger prerequisite (normally done by gaz.New())
		err := di.For[*slog.Logger](c).Instance(slog.Default())
		require.NoError(t, err)

		// Apply module
		module := NewModule()
		err = module.Register(c)
		require.NoError(t, err)
	})

	t.Run("returns valid di.Module", func(t *testing.T) {
		module := NewModule()
		require.NotNil(t, module)
		require.Equal(t, "worker", module.Name())
	})

	t.Run("works without logger registered", func(t *testing.T) {
		c := di.New()

		// Don't register logger - module should handle gracefully
		module := NewModule()
		err := module.Register(c)
		require.NoError(t, err, "module should not error without logger")
	})
}
