package grpc

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/petabytecl/gaz"
	"github.com/petabytecl/gaz/di"
)

func TestNewModule(t *testing.T) {
	// Test with defaults.
	t.Run("defaults", func(t *testing.T) {
		app := gaz.New()

		// Register module
		module := NewModule()
		// Apply module to app
		err := module.Apply(app)
		require.NoError(t, err)

		// Build app to trigger registration
		err = app.Build()
		require.NoError(t, err)

		// Verify server was registered in container
		c := app.Container()
		require.True(t, di.Has[*Server](c))

		// Verify config has defaults
		cfg, err := di.Resolve[Config](c)
		require.NoError(t, err)
		require.Equal(t, DefaultPort, cfg.Port)
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
