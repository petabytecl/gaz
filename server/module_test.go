package server

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/petabytecl/gaz"
	"github.com/petabytecl/gaz/di"
	"github.com/petabytecl/gaz/server/gateway"
	"github.com/petabytecl/gaz/server/grpc"
	shttp "github.com/petabytecl/gaz/server/http"
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
		require.True(t, di.Has[*gateway.Gateway](c))
		// http.Server is registered via gateway module -> http module
		require.True(t, di.Has[*shttp.Server](c))
	})

	// Test module name.
	t.Run("module name", func(t *testing.T) {
		module := NewModule()
		require.Equal(t, "server", module.Name())
	})
}
