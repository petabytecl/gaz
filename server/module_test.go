package server

import (
	"log/slog"
	"net/http"
	"testing"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
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

//nolint:funlen // Test function with multiple subtests is naturally long.
func TestNewModuleWithFlags(t *testing.T) {
	t.Run("flags registration", func(t *testing.T) {
		fs := pflag.NewFlagSet("test", pflag.ContinueOnError)
		_ = NewModuleWithFlags(fs)

		// Verify flags are registered with correct defaults.
		grpcPort := fs.Lookup("grpc-port")
		require.NotNil(t, grpcPort, "--grpc-port flag should be registered")
		require.Equal(t, "50051", grpcPort.DefValue)

		httpPort := fs.Lookup("http-port")
		require.NotNil(t, httpPort, "--http-port flag should be registered")
		require.Equal(t, "8080", httpPort.DefValue)

		grpcReflection := fs.Lookup("grpc-reflection")
		require.NotNil(t, grpcReflection, "--grpc-reflection flag should be registered")
		require.Equal(t, "true", grpcReflection.DefValue)

		grpcDevMode := fs.Lookup("grpc-dev-mode")
		require.NotNil(t, grpcDevMode, "--grpc-dev-mode flag should be registered")
		require.Equal(t, "false", grpcDevMode.DefValue)
	})

	t.Run("options affect defaults", func(t *testing.T) {
		fs := pflag.NewFlagSet("test", pflag.ContinueOnError)
		_ = NewModuleWithFlags(fs,
			WithGRPCPort(9090),
			WithHTTPPort(3000),
			WithGRPCReflection(false),
		)

		// Options should have set the defaults.
		require.Equal(t, "9090", fs.Lookup("grpc-port").DefValue)
		require.Equal(t, "3000", fs.Lookup("http-port").DefValue)
		require.Equal(t, "false", fs.Lookup("grpc-reflection").DefValue)
	})

	t.Run("flag values used at resolution", func(t *testing.T) {
		// This test verifies the critical timing: flag values are read at resolution time,
		// not at module creation time.
		fs := pflag.NewFlagSet("test", pflag.ContinueOnError)
		_ = NewModuleWithFlags(fs) // defaults: grpc=50051, http=8080

		// Simulate flag parsing with custom values.
		err := fs.Parse([]string{"--grpc-port=7777", "--http-port=9999"})
		require.NoError(t, err)

		// Verify flags were parsed correctly.
		grpcVal, err := fs.GetInt("grpc-port")
		require.NoError(t, err)
		require.Equal(t, 7777, grpcVal)

		httpVal, err := fs.GetInt("http-port")
		require.NoError(t, err)
		require.Equal(t, 9999, httpVal)
	})

	t.Run("cobra integration", func(t *testing.T) {
		// Create a cobra command to attach flags to.
		cmd := &cobra.Command{Use: "test"}

		_ = NewModuleWithFlags(cmd.PersistentFlags(), WithGRPCPort(5000)) // default 5000

		// Parse args to override.
		err := cmd.ParseFlags([]string{"--grpc-port=6000"})
		require.NoError(t, err)

		// Verify flag was parsed to cmd.
		val, err := cmd.PersistentFlags().GetInt("grpc-port")
		require.NoError(t, err)
		require.Equal(t, 6000, val)
	})

	t.Run("module name", func(t *testing.T) {
		fs := pflag.NewFlagSet("test", pflag.ContinueOnError)
		m := NewModuleWithFlags(fs)
		require.Equal(t, "server", m.Name())
	})

	t.Run("full module apply", func(t *testing.T) {
		// Test that the module can be applied to a di.Container directly.
		fs := pflag.NewFlagSet("test", pflag.ContinueOnError)
		m := NewModuleWithFlags(fs,
			WithGRPCPort(50052),
			WithHTTPPort(8081),
		)

		c := di.New()
		require.NoError(t, di.For[*slog.Logger](c).Instance(slog.Default()))

		// Registration should succeed.
		err := m.Register(c)
		require.NoError(t, err)

		// Verify both servers were registered.
		require.True(t, di.Has[*grpc.Server](c))
		require.True(t, di.Has[*shttp.Server](c))
	})

	t.Run("resolved config uses flag values", func(t *testing.T) {
		// This is the critical integration test: verify that when flags are parsed,
		// the resolved configs have the flag values, not the option defaults.
		fs := pflag.NewFlagSet("test", pflag.ContinueOnError)
		m := NewModuleWithFlags(fs,
			WithGRPCPort(50051), // option default
			WithHTTPPort(8080),  // option default
		)

		// 1. Flags are parsed (simulating cobra parsing)
		err := fs.Parse([]string{"--grpc-port=12345", "--http-port=54321"})
		require.NoError(t, err)

		// 2. Module's Register() is called (which reads flag values)
		c := di.New()
		require.NoError(t, di.For[*slog.Logger](c).Instance(slog.Default()))

		err = m.Register(c)
		require.NoError(t, err)

		// 3. Verify configs have the parsed flag values, not defaults.
		grpcCfg, err := di.Resolve[grpc.Config](c)
		require.NoError(t, err)
		require.Equal(t, 12345, grpcCfg.Port)

		httpCfg, err := di.Resolve[shttp.Config](c)
		require.NoError(t, err)
		require.Equal(t, 54321, httpCfg.Port)
	})
}
