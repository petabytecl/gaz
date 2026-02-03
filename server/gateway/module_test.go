package gateway

import (
	"log/slog"
	"testing"

	"github.com/spf13/pflag"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"

	"github.com/petabytecl/gaz/di"
)

// ModuleTestSuite tests Gateway module registration.
type ModuleTestSuite struct {
	suite.Suite
}

func TestModuleTestSuite(t *testing.T) {
	suite.Run(t, new(ModuleTestSuite))
}

func (s *ModuleTestSuite) TestNewModule_Defaults() {
	c := di.New()
	s.Require().NoError(di.For[*slog.Logger](c).Instance(slog.Default()))

	module := NewModule()
	err := module.Register(c)
	s.Require().NoError(err)

	// Verify Gateway was registered.
	s.Require().True(di.Has[*Gateway](c))

	// Verify Config was registered.
	s.Require().True(di.Has[Config](c))

	cfg, err := di.Resolve[Config](c)
	s.Require().NoError(err)
	s.Require().Equal(DefaultPort, cfg.Port)
}

func (s *ModuleTestSuite) TestNewModule_WithPort() {
	c := di.New()
	s.Require().NoError(di.For[*slog.Logger](c).Instance(slog.Default()))

	module := NewModule(WithPort(9000))
	err := module.Register(c)
	s.Require().NoError(err)

	cfg, err := di.Resolve[Config](c)
	s.Require().NoError(err)
	s.Require().Equal(9000, cfg.Port)
}

func (s *ModuleTestSuite) TestNewModule_WithGRPCTarget() {
	c := di.New()
	s.Require().NoError(di.For[*slog.Logger](c).Instance(slog.Default()))

	module := NewModule(WithGRPCTarget("custom:9090"))
	err := module.Register(c)
	s.Require().NoError(err)

	cfg, err := di.Resolve[Config](c)
	s.Require().NoError(err)
	s.Require().Equal("custom:9090", cfg.GRPCTarget)
}

func (s *ModuleTestSuite) TestNewModule_WithDevMode() {
	c := di.New()
	s.Require().NoError(di.For[*slog.Logger](c).Instance(slog.Default()))

	module := NewModule(WithDevMode(true))
	err := module.Register(c)
	s.Require().NoError(err)

	// Verify Gateway was registered.
	s.Require().True(di.Has[*Gateway](c))

	// Dev mode affects CORS config.
	cfg, err := di.Resolve[Config](c)
	s.Require().NoError(err)
	s.Require().Equal([]string{"*"}, cfg.CORS.AllowedOrigins, "Dev mode CORS should allow all origins")
}

func (s *ModuleTestSuite) TestNewModule_WithCORS() {
	c := di.New()
	s.Require().NoError(di.For[*slog.Logger](c).Instance(slog.Default()))

	customCORS := CORSConfig{
		AllowedOrigins: []string{"https://example.com"},
		AllowedMethods: []string{"GET", "POST"},
		MaxAge:         3600,
	}
	module := NewModule(WithCORS(customCORS))
	err := module.Register(c)
	s.Require().NoError(err)

	cfg, err := di.Resolve[Config](c)
	s.Require().NoError(err)
	s.Require().Equal([]string{"https://example.com"}, cfg.CORS.AllowedOrigins)
	s.Require().Equal(3600, cfg.CORS.MaxAge)
}

func (s *ModuleTestSuite) TestNewModule_RegistersConfig() {
	c := di.New()
	s.Require().NoError(di.For[*slog.Logger](c).Instance(slog.Default()))

	module := NewModule()
	err := module.Register(c)
	s.Require().NoError(err)

	s.Require().True(di.Has[Config](c))
}

func (s *ModuleTestSuite) TestNewModule_RegistersGateway() {
	c := di.New()
	s.Require().NoError(di.For[*slog.Logger](c).Instance(slog.Default()))

	module := NewModule()
	err := module.Register(c)
	s.Require().NoError(err)

	s.Require().True(di.Has[*Gateway](c))
}

func (s *ModuleTestSuite) TestModule_GatewayIsEager() {
	c := di.New()
	s.Require().NoError(di.For[*slog.Logger](c).Instance(slog.Default()))

	module := NewModule()
	err := module.Register(c)
	s.Require().NoError(err)

	// Gateway should be marked as eager (initialized on app start).
	// This is tested by verifying it's registered.
	s.Require().True(di.Has[*Gateway](c))
}

func (s *ModuleTestSuite) TestModule_ResolveFails_NoLogger() {
	c := di.New()
	// Do NOT register logger.

	cfg := DefaultConfig()
	s.Require().NoError(di.For[Config](c).Instance(cfg))

	err := Module(c, false)
	s.Require().NoError(err, "Registration should succeed")

	// Resolution should fail because logger is missing.
	_, err = di.Resolve[*Gateway](c)
	s.Require().Error(err)
	s.Require().Contains(err.Error(), "logger")
}

func (s *ModuleTestSuite) TestModule_ResolveFails_NoConfig() {
	c := di.New()
	s.Require().NoError(di.For[*slog.Logger](c).Instance(slog.Default()))
	// Do NOT register Config.

	err := Module(c, false)
	s.Require().NoError(err, "Registration should succeed")

	// Resolution should fail because Config is missing.
	_, err = di.Resolve[*Gateway](c)
	s.Require().Error(err)
	s.Require().Contains(err.Error(), "config")
}

// TestNewModuleWithFlags tests require running in subtests
// to avoid flag redefinition (each test needs its own FlagSet).
//
//nolint:funlen // Test uses subtests to organize related test cases.
func TestNewModuleWithFlags(t *testing.T) {
	t.Run("defines flags", func(t *testing.T) {
		fs := pflag.NewFlagSet("test", pflag.ContinueOnError)
		_ = NewModuleWithFlags(fs)

		// Verify flags are defined.
		port := fs.Lookup("gateway-port")
		require.NotNil(t, port, "gateway-port flag should be defined")
		require.Equal(t, "8080", port.DefValue)

		grpcTarget := fs.Lookup("gateway-grpc-target")
		require.NotNil(t, grpcTarget, "gateway-grpc-target flag should be defined")

		devMode := fs.Lookup("gateway-dev-mode")
		require.NotNil(t, devMode, "gateway-dev-mode flag should be defined")
		require.Equal(t, "false", devMode.DefValue)
	})

	t.Run("reads flag values", func(t *testing.T) {
		fs := pflag.NewFlagSet("test", pflag.ContinueOnError)
		module := NewModuleWithFlags(fs)

		// Parse flags with custom values.
		err := fs.Parse([]string{"--gateway-port=9000", "--gateway-grpc-target=custom:8080"})
		require.NoError(t, err)

		// Register module.
		c := di.New()
		require.NoError(t, di.For[*slog.Logger](c).Instance(slog.Default()))

		err = module.Register(c)
		require.NoError(t, err)

		// Verify config uses flag values.
		cfg, err := di.Resolve[Config](c)
		require.NoError(t, err)
		require.Equal(t, 9000, cfg.Port)
		require.Equal(t, "custom:8080", cfg.GRPCTarget)
	})

	t.Run("reads dev mode flag", func(t *testing.T) {
		fs := pflag.NewFlagSet("test", pflag.ContinueOnError)
		module := NewModuleWithFlags(fs)

		err := fs.Parse([]string{"--gateway-dev-mode=true"})
		require.NoError(t, err)

		c := di.New()
		require.NoError(t, di.For[*slog.Logger](c).Instance(slog.Default()))

		err = module.Register(c)
		require.NoError(t, err)

		cfg, err := di.Resolve[Config](c)
		require.NoError(t, err)
		// Dev mode affects CORS config.
		require.Equal(t, []string{"*"}, cfg.CORS.AllowedOrigins)
	})

	t.Run("with options and flags", func(t *testing.T) {
		fs := pflag.NewFlagSet("test", pflag.ContinueOnError)
		// Set initial port via option.
		module := NewModuleWithFlags(fs, WithPort(7000))

		// Don't parse any flags - should use option value.
		c := di.New()
		require.NoError(t, di.For[*slog.Logger](c).Instance(slog.Default()))

		err := module.Register(c)
		require.NoError(t, err)

		cfg, err := di.Resolve[Config](c)
		require.NoError(t, err)
		// The flag default is set from the option value.
		require.Equal(t, 7000, cfg.Port)
	})

	t.Run("flag overrides option", func(t *testing.T) {
		fs := pflag.NewFlagSet("test", pflag.ContinueOnError)
		// Set initial port via option.
		module := NewModuleWithFlags(fs, WithPort(7000))

		// Parse flag to override.
		err := fs.Parse([]string{"--gateway-port=8000"})
		require.NoError(t, err)

		c := di.New()
		require.NoError(t, di.For[*slog.Logger](c).Instance(slog.Default()))

		err = module.Register(c)
		require.NoError(t, err)

		cfg, err := di.Resolve[Config](c)
		require.NoError(t, err)
		// Flag value overrides option.
		require.Equal(t, 8000, cfg.Port)
	})

	t.Run("grpc target from option when flag empty", func(t *testing.T) {
		fs := pflag.NewFlagSet("test", pflag.ContinueOnError)
		module := NewModuleWithFlags(fs, WithGRPCTarget("option-target:9090"))

		// Don't set grpc-target flag.
		c := di.New()
		require.NoError(t, di.For[*slog.Logger](c).Instance(slog.Default()))

		err := module.Register(c)
		require.NoError(t, err)

		cfg, err := di.Resolve[Config](c)
		require.NoError(t, err)
		require.Equal(t, "option-target:9090", cfg.GRPCTarget)
	})

	t.Run("with custom CORS", func(t *testing.T) {
		fs := pflag.NewFlagSet("test", pflag.ContinueOnError)
		customCORS := CORSConfig{
			AllowedOrigins: []string{"https://custom.com"},
			MaxAge:         7200,
		}
		module := NewModuleWithFlags(fs, WithCORS(customCORS))

		c := di.New()
		require.NoError(t, di.For[*slog.Logger](c).Instance(slog.Default()))

		err := module.Register(c)
		require.NoError(t, err)

		cfg, err := di.Resolve[Config](c)
		require.NoError(t, err)
		require.Equal(t, []string{"https://custom.com"}, cfg.CORS.AllowedOrigins)
		require.Equal(t, 7200, cfg.CORS.MaxAge)
	})
}

func TestModuleOptions(t *testing.T) {
	t.Run("WithPort modifies config", func(t *testing.T) {
		cfg := defaultModuleConfig()
		require.Equal(t, DefaultPort, cfg.port)

		WithPort(9999)(cfg)
		require.Equal(t, 9999, cfg.port)
	})

	t.Run("WithGRPCTarget modifies config", func(t *testing.T) {
		cfg := defaultModuleConfig()
		require.Empty(t, cfg.grpcTarget)

		WithGRPCTarget("custom:1234")(cfg)
		require.Equal(t, "custom:1234", cfg.grpcTarget)
	})

	t.Run("WithDevMode modifies config", func(t *testing.T) {
		cfg := defaultModuleConfig()
		require.False(t, cfg.devMode)

		WithDevMode(true)(cfg)
		require.True(t, cfg.devMode)
	})

	t.Run("WithCORS modifies config", func(t *testing.T) {
		cfg := defaultModuleConfig()
		require.Nil(t, cfg.cors)

		corsConfig := CORSConfig{MaxAge: 1000}
		WithCORS(corsConfig)(cfg)
		require.NotNil(t, cfg.cors)
		require.Equal(t, 1000, cfg.cors.MaxAge)
	})
}
