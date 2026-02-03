package http

import (
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/petabytecl/gaz/di"
)

func TestNewModule(t *testing.T) {
	// Test with default options.
	t.Run("defaults", func(t *testing.T) {
		c := di.New()

		module := NewModule()
		err := module.Register(c)
		require.NoError(t, err)

		// Verify server was registered.
		require.True(t, di.Has[*Server](c))

		// Verify config has defaults.
		cfg, err := di.Resolve[Config](c)
		require.NoError(t, err)
		require.Equal(t, DefaultPort, cfg.Port)
	})

	// Test with custom port.
	t.Run("with port", func(t *testing.T) {
		c := di.New()

		module := NewModule(WithPort(3000))
		err := module.Register(c)
		require.NoError(t, err)

		cfg, err := di.Resolve[Config](c)
		require.NoError(t, err)
		require.Equal(t, 3000, cfg.Port)
	})

	// Test with custom timeouts.
	t.Run("with timeouts", func(t *testing.T) {
		c := di.New()

		module := NewModule(
			WithReadTimeout(1*time.Second),
			WithWriteTimeout(2*time.Second),
			WithIdleTimeout(3*time.Second),
			WithReadHeaderTimeout(500*time.Millisecond),
		)
		err := module.Register(c)
		require.NoError(t, err)

		cfg, err := di.Resolve[Config](c)
		require.NoError(t, err)
		require.Equal(t, 1*time.Second, cfg.ReadTimeout)
		require.Equal(t, 2*time.Second, cfg.WriteTimeout)
		require.Equal(t, 3*time.Second, cfg.IdleTimeout)
		require.Equal(t, 500*time.Millisecond, cfg.ReadHeaderTimeout)
	})

	// Test with custom handler.
	t.Run("with handler", func(t *testing.T) {
		c := di.New()

		handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {})
		module := NewModule(WithHandler(handler))
		err := module.Register(c)
		require.NoError(t, err)

		require.True(t, di.Has[*Server](c))
	})
}

func TestModule(t *testing.T) {
	// Test the Module function (used when Config is pre-registered).
	t.Run("successful registration and resolution", func(t *testing.T) {
		c := di.New()

		// Pre-register config.
		cfg := DefaultConfig()
		cfg.Port = 4000
		require.NoError(t, di.For[Config](c).Instance(cfg))

		err := Module(c)
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

		// Register module WITHOUT pre-registering Config.
		err := Module(c)
		require.NoError(t, err) // Registration succeeds

		// Resolution fails because Config is missing.
		_, err = di.Resolve[*Server](c)
		require.Error(t, err)
		require.Contains(t, err.Error(), "http config")
	})

	t.Run("uses default logger when not provided", func(t *testing.T) {
		c := di.New()
		// NO logger registered - should use slog.Default().

		// Pre-register config.
		cfg := DefaultConfig()
		require.NoError(t, di.For[Config](c).Instance(cfg))

		err := Module(c)
		require.NoError(t, err)

		// Resolution succeeds with default logger.
		server, err := di.Resolve[*Server](c)
		require.NoError(t, err)
		require.NotNil(t, server)
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
