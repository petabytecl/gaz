package cron

import (
	"log/slog"
	"testing"

	"github.com/petabytecl/gaz/di"
	"github.com/stretchr/testify/require"
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
		require.Equal(t, "cron", module.Name())
	})

	t.Run("works without logger registered", func(t *testing.T) {
		c := di.New()

		// Don't register logger - module should handle gracefully
		module := NewModule()
		err := module.Register(c)
		require.NoError(t, err, "module should not error without logger")
	})
}
