package server

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/petabytecl/gaz"
	"github.com/petabytecl/gaz/di"
	"github.com/petabytecl/gaz/server/grpc"
	"github.com/petabytecl/gaz/server/vanguard"
)

func TestNewModule(t *testing.T) {
	// Test with defaults.
	t.Run("defaults", func(t *testing.T) {
		app := gaz.New()

		module := NewModule()
		err := module.Apply(app)
		require.NoError(t, err)

		err = app.Build()
		require.NoError(t, err)

		c := app.Container()

		// Verify servers were registered.
		require.True(t, di.Has[*grpc.Server](c))
		require.True(t, di.Has[*vanguard.Server](c))
	})

	// Test gRPC SkipListener is forced true.
	t.Run("grpc skip listener", func(t *testing.T) {
		app := gaz.New()

		module := NewModule()
		err := module.Apply(app)
		require.NoError(t, err)

		err = app.Build()
		require.NoError(t, err)

		c := app.Container()

		cfg, err := di.Resolve[grpc.Config](c)
		require.NoError(t, err)
		require.True(t, cfg.SkipListener, "gRPC SkipListener must be true when using server module")
	})

	// Test module name.
	t.Run("module name", func(t *testing.T) {
		module := NewModule()
		require.Equal(t, "server", module.Name())
	})
}
