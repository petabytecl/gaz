package server

import (
	"log/slog"
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/petabytecl/gaz/di"
	"github.com/petabytecl/gaz/server/grpc"
	shttp "github.com/petabytecl/gaz/server/http"
)

func TestNewModule(t *testing.T) {
	// Test with default options.
	t.Run("defaults", func(t *testing.T) {
		c := di.New()
		require.NoError(t, di.For[*slog.Logger](c).Instance(slog.Default()))

		module := NewModule()
		err := module.Register(c)
		require.NoError(t, err)

		// Verify both servers were registered.
		require.True(t, di.Has[*grpc.Server](c))
		require.True(t, di.Has[*shttp.Server](c))
	})

	// Test with custom ports.
	t.Run("with custom ports", func(t *testing.T) {
		c := di.New()
		require.NoError(t, di.For[*slog.Logger](c).Instance(slog.Default()))

		module := NewModule(
			WithGRPCPort(9000),
			WithHTTPPort(3000),
		)
		err := module.Register(c)
		require.NoError(t, err)

		// Verify gRPC config has custom port.
		grpcCfg, err := di.Resolve[grpc.Config](c)
		require.NoError(t, err)
		require.Equal(t, 9000, grpcCfg.Port)

		// Verify HTTP config has custom port.
		httpCfg, err := di.Resolve[shttp.Config](c)
		require.NoError(t, err)
		require.Equal(t, 3000, httpCfg.Port)
	})

	// Test with reflection disabled.
	t.Run("with reflection disabled", func(t *testing.T) {
		c := di.New()
		require.NoError(t, di.For[*slog.Logger](c).Instance(slog.Default()))

		module := NewModule(WithGRPCReflection(false))
		err := module.Register(c)
		require.NoError(t, err)

		grpcCfg, err := di.Resolve[grpc.Config](c)
		require.NoError(t, err)
		require.False(t, grpcCfg.Reflection)
	})

	// Test with dev mode.
	t.Run("with dev mode", func(t *testing.T) {
		c := di.New()
		require.NoError(t, di.For[*slog.Logger](c).Instance(slog.Default()))

		module := NewModule(WithGRPCDevMode(true))
		err := module.Register(c)
		require.NoError(t, err)

		// Dev mode is internal to server, just verify registration.
		require.True(t, di.Has[*grpc.Server](c))
	})

	// Test with custom HTTP handler.
	t.Run("with http handler", func(t *testing.T) {
		c := di.New()
		require.NoError(t, di.For[*slog.Logger](c).Instance(slog.Default()))

		handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {})
		module := NewModule(WithHTTPHandler(handler))
		err := module.Register(c)
		require.NoError(t, err)

		require.True(t, di.Has[*shttp.Server](c))
	})

	// Test module name.
	t.Run("module name", func(t *testing.T) {
		module := NewModule()
		require.Equal(t, "server", module.Name())
	})
}
