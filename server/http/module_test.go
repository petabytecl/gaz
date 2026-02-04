package http

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

		module := NewModule()
		err := module.Apply(app)
		require.NoError(t, err)

		err = app.Build()
		require.NoError(t, err)

		c := app.Container()

		// Verify server was registered.
		require.True(t, di.Has[*Server](c))

		// Verify config has defaults.
		cfg, err := di.Resolve[Config](c)
		require.NoError(t, err)
		require.Equal(t, DefaultPort, cfg.Port)
	})
}

func TestConfigSetDefaults(t *testing.T) {
	cfg := Config{}
	cfg.SetDefaults()

	require.Equal(t, DefaultPort, cfg.Port)
	require.Equal(t, DefaultReadTimeout, cfg.ReadTimeout)
	require.Equal(t, DefaultWriteTimeout, cfg.WriteTimeout)
	require.Equal(t, DefaultIdleTimeout, cfg.IdleTimeout)
	require.Equal(t, DefaultReadHeaderTimeout, cfg.ReadHeaderTimeout)
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

	t.Run("invalid read timeout", func(t *testing.T) {
		cfg := DefaultConfig()
		cfg.ReadTimeout = 0
		require.Error(t, cfg.Validate())
		require.Contains(t, cfg.Validate().Error(), "read_timeout")
	})

	t.Run("invalid write timeout", func(t *testing.T) {
		cfg := DefaultConfig()
		cfg.WriteTimeout = 0
		require.Error(t, cfg.Validate())
		require.Contains(t, cfg.Validate().Error(), "write_timeout")
	})

	t.Run("invalid idle timeout", func(t *testing.T) {
		cfg := DefaultConfig()
		cfg.IdleTimeout = 0
		require.Error(t, cfg.Validate())
		require.Contains(t, cfg.Validate().Error(), "idle_timeout")
	})

	t.Run("invalid read header timeout", func(t *testing.T) {
		cfg := DefaultConfig()
		cfg.ReadHeaderTimeout = 0
		require.Error(t, cfg.Validate())
		require.Contains(t, cfg.Validate().Error(), "read_header_timeout")
	})
}
