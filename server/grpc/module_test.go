package grpc

import (
	"log/slog"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/petabytecl/gaz/di"
)

func TestNewModule(t *testing.T) {
	// Test with default options.
	t.Run("defaults", func(t *testing.T) {
		c := di.New()
		require.NoError(t, di.For[*slog.Logger](c).Instance(slog.Default()))

		module := NewModule()
		err := module.Register(c)
		require.NoError(t, err)

		// Verify server was registered.
		require.True(t, di.Has[*Server](c))
	})

	// Test with custom port.
	t.Run("with port", func(t *testing.T) {
		c := di.New()
		require.NoError(t, di.For[*slog.Logger](c).Instance(slog.Default()))

		module := NewModule(WithPort(9999))
		err := module.Register(c)
		require.NoError(t, err)

		// Verify config has custom port.
		cfg, err := di.Resolve[Config](c)
		require.NoError(t, err)
		require.Equal(t, 9999, cfg.Port)
	})

	// Test with reflection disabled.
	t.Run("with reflection disabled", func(t *testing.T) {
		c := di.New()
		require.NoError(t, di.For[*slog.Logger](c).Instance(slog.Default()))

		module := NewModule(WithReflection(false))
		err := module.Register(c)
		require.NoError(t, err)

		cfg, err := di.Resolve[Config](c)
		require.NoError(t, err)
		require.False(t, cfg.Reflection)
	})

	// Test with dev mode.
	t.Run("with dev mode", func(t *testing.T) {
		c := di.New()
		require.NoError(t, di.For[*slog.Logger](c).Instance(slog.Default()))

		module := NewModule(WithDevMode(true))
		err := module.Register(c)
		require.NoError(t, err)

		// Dev mode is passed to server constructor, not stored in config.
		// Just verify registration succeeds.
		require.True(t, di.Has[*Server](c))
	})
}

func TestModule(t *testing.T) {
	// Test the Module function (used when Config is pre-registered).
	t.Run("successful registration and resolution", func(t *testing.T) {
		c := di.New()
		require.NoError(t, di.For[*slog.Logger](c).Instance(slog.Default()))

		// Pre-register config.
		cfg := DefaultConfig()
		cfg.Port = 5000
		require.NoError(t, di.For[Config](c).Instance(cfg))

		err := Module(c, false)
		require.NoError(t, err)

		// Verify server was registered.
		require.True(t, di.Has[*Server](c))

		// Actually resolve to trigger provider callback.
		server, err := di.Resolve[*Server](c)
		require.NoError(t, err)
		require.NotNil(t, server)
	})

	t.Run("resolve fails without config", func(t *testing.T) {
		c := di.New()
		require.NoError(t, di.For[*slog.Logger](c).Instance(slog.Default()))

		// Register module WITHOUT pre-registering Config.
		err := Module(c, false)
		require.NoError(t, err) // Registration succeeds

		// Resolution fails because Config is missing.
		_, err = di.Resolve[*Server](c)
		require.Error(t, err)
		require.Contains(t, err.Error(), "grpc config")
	})

	t.Run("resolve succeeds with slog.Default fallback", func(t *testing.T) {
		c := di.New()
		// NO logger registered - should fallback to slog.Default()

		// Pre-register config.
		cfg := DefaultConfig()
		cfg.Port = 0 // Use any available port
		require.NoError(t, di.For[Config](c).Instance(cfg))

		err := Module(c, false)
		require.NoError(t, err) // Registration succeeds

		// Resolution should succeed with slog.Default() fallback
		server, err := di.Resolve[*Server](c)
		require.NoError(t, err)
		require.NotNil(t, server)
	})
}

func TestConfigSetDefaults(t *testing.T) {
	cfg := Config{}
	cfg.SetDefaults()

	require.Equal(t, DefaultPort, cfg.Port)
	require.Equal(t, DefaultMaxMsgSize, cfg.MaxRecvMsgSize)
	require.Equal(t, DefaultMaxMsgSize, cfg.MaxSendMsgSize)
}

func TestConfigValidate(t *testing.T) {
	t.Run("valid config", func(t *testing.T) {
		cfg := DefaultConfig()
		require.NoError(t, cfg.Validate())
	})

	t.Run("invalid port - zero", func(t *testing.T) {
		cfg := DefaultConfig()
		cfg.Port = 0
		require.Error(t, cfg.Validate())
		require.Contains(t, cfg.Validate().Error(), "port")
	})

	t.Run("invalid port - too high", func(t *testing.T) {
		cfg := DefaultConfig()
		cfg.Port = 70000
		require.Error(t, cfg.Validate())
		require.Contains(t, cfg.Validate().Error(), "port")
	})

	t.Run("invalid max recv msg size", func(t *testing.T) {
		cfg := DefaultConfig()
		cfg.MaxRecvMsgSize = 0
		require.Error(t, cfg.Validate())
		require.Contains(t, cfg.Validate().Error(), "max_recv_msg_size")
	})

	t.Run("invalid max send msg size", func(t *testing.T) {
		cfg := DefaultConfig()
		cfg.MaxSendMsgSize = 0
		require.Error(t, cfg.Validate())
		require.Contains(t, cfg.Validate().Error(), "max_send_msg_size")
	})
}
