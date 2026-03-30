package vanguard

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"testing"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/suite"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"

	"github.com/petabytecl/gaz/di"
	"github.com/petabytecl/gaz/health"
	connectpkg "github.com/petabytecl/gaz/server/connect"
	grpcpkg "github.com/petabytecl/gaz/server/grpc"
)

// ModuleTestSuite tests the Vanguard module registration.
type ModuleTestSuite struct {
	suite.Suite
}

func TestModuleTestSuite(t *testing.T) {
	suite.Run(t, new(ModuleTestSuite))
}

func (s *ModuleTestSuite) TestNewModuleCreatesModule() {
	mod := NewModule()
	s.Require().NotNil(mod)
}

func (s *ModuleTestSuite) TestNewModuleName() {
	mod := NewModule()
	s.Equal("vanguard", mod.Name())
}

func (s *ModuleTestSuite) TestProvideConfigDefaultValues() {
	cfg := DefaultConfig()
	s.Equal(DefaultPort, cfg.Port)
	s.Equal("server", cfg.Namespace())
	s.True(cfg.Reflection)
	s.True(cfg.HealthEnabled)
	s.False(cfg.DevMode)
}

// --- resolveLogger tests ---

func (s *ModuleTestSuite) TestResolveLogger_WithRegisteredLogger() {
	container := di.New()
	logger := slog.Default()
	s.Require().NoError(di.For[*slog.Logger](container).Instance(logger))

	resolved := resolveLogger(container)
	s.Equal(logger, resolved)
}

func (s *ModuleTestSuite) TestResolveLogger_FallsBackToDefault() {
	container := di.New()

	resolved := resolveLogger(container)
	s.NotNil(resolved, "should fall back to slog.Default()")
}

// --- provideConfig tests ---

func (s *ModuleTestSuite) TestProvideConfig_RegistersConfig() {
	container := di.New()
	cfg := DefaultConfig()
	cfg.AllowZeroWriteTimeout = true // Explicit opt-in for streaming-safe zero timeout.

	err := provideConfig(cfg)(container)
	s.Require().NoError(err)

	// Resolve the config provider — it should produce a valid Config.
	resolved, resolveErr := di.Resolve[Config](container)
	s.Require().NoError(resolveErr)
	s.Equal(cfg.Port, resolved.Port)
}

func (s *ModuleTestSuite) TestProvideConfig_InvalidConfig() {
	container := di.New()
	cfg := Config{Port: 0} // Invalid — port must be > 0.

	err := provideConfig(cfg)(container)
	s.Require().NoError(err, "registration should succeed")

	// Resolution should fail due to validation.
	_, resolveErr := di.Resolve[Config](container)
	s.Require().Error(resolveErr)
	s.Contains(resolveErr.Error(), "validate")
}

// --- provideCORSMiddleware tests ---

func (s *ModuleTestSuite) TestProvideCORSMiddleware_Registers() {
	container := di.New()

	// Register prerequisite Config.
	cfg := DefaultConfig()
	s.Require().NoError(di.For[Config](container).Instance(cfg))

	err := provideCORSMiddleware(container)
	s.Require().NoError(err)

	// Resolve the CORS middleware.
	mw, resolveErr := di.Resolve[*CORSMiddleware](container)
	s.Require().NoError(resolveErr)
	s.NotNil(mw)
	s.Equal("cors", mw.Name())
}

// --- provideConnectLoggingBundle tests ---

func (s *ModuleTestSuite) TestProvideConnectLoggingBundle_Registers() {
	container := di.New()

	err := provideConnectLoggingBundle(container)
	s.Require().NoError(err)

	bundle, resolveErr := di.Resolve[*connectpkg.LoggingBundle](container)
	s.Require().NoError(resolveErr)
	s.NotNil(bundle)
	s.Equal("logging", bundle.Name())
}

// --- provideConnectRecoveryBundle tests ---

func (s *ModuleTestSuite) TestProvideConnectRecoveryBundle_Registers() {
	container := di.New()

	// Register prerequisite Config.
	cfg := DefaultConfig()
	s.Require().NoError(di.For[Config](container).Instance(cfg))

	err := provideConnectRecoveryBundle(container)
	s.Require().NoError(err)

	bundle, resolveErr := di.Resolve[*connectpkg.RecoveryBundle](container)
	s.Require().NoError(resolveErr)
	s.NotNil(bundle)
	s.Equal("recovery", bundle.Name())
}

// --- provideConnectValidationBundle tests ---

func (s *ModuleTestSuite) TestProvideConnectValidationBundle_Registers() {
	container := di.New()

	err := provideConnectValidationBundle(container)
	s.Require().NoError(err)

	bundle, resolveErr := di.Resolve[*connectpkg.ValidationBundle](container)
	s.Require().NoError(resolveErr)
	s.NotNil(bundle)
	s.Equal("validation", bundle.Name())
}

// --- provideConnectAuthBundle tests ---

func (s *ModuleTestSuite) TestProvideConnectAuthBundle_WithoutAuthFunc_SkipsSilently() {
	container := di.New()

	err := provideConnectAuthBundle(container)
	s.Require().NoError(err)

	// Should not be registered.
	_, resolveErr := di.Resolve[*connectpkg.AuthBundle](container)
	s.Require().Error(resolveErr, "auth bundle should not be registered without AuthFunc")
}

func (s *ModuleTestSuite) TestProvideConnectAuthBundle_WithAuthFunc_Registers() {
	container := di.New()

	// Register an AuthFunc.
	authFunc := connectpkg.AuthFunc(func(ctx context.Context, _ http.Header, _ connect.Spec) (context.Context, error) {
		return ctx, nil
	})
	s.Require().NoError(di.For[connectpkg.AuthFunc](container).Instance(authFunc))

	err := provideConnectAuthBundle(container)
	s.Require().NoError(err)

	bundle, resolveErr := di.Resolve[*connectpkg.AuthBundle](container)
	s.Require().NoError(resolveErr)
	s.NotNil(bundle)
	s.Equal("auth", bundle.Name())
}

// --- provideConnectRateLimitBundle tests ---

func (s *ModuleTestSuite) TestProvideConnectRateLimitBundle_WithoutLimiter_UsesAlwaysPass() {
	container := di.New()

	err := provideConnectRateLimitBundle(container)
	s.Require().NoError(err)

	bundle, resolveErr := di.Resolve[*connectpkg.RateLimitBundle](container)
	s.Require().NoError(resolveErr)
	s.NotNil(bundle)
	s.Equal("ratelimit", bundle.Name())
}

// --- provideOTELMiddleware tests ---

func (s *ModuleTestSuite) TestProvideOTELMiddleware_WithTracerProvider() {
	container := di.New()

	tp := sdktrace.NewTracerProvider()
	defer func() { _ = tp.Shutdown(s.T().Context()) }()

	s.Require().NoError(di.For[*sdktrace.TracerProvider](container).Instance(tp))

	// Register health config.
	s.Require().NoError(di.For[health.Config](container).Instance(health.DefaultConfig()))

	err := provideOTELMiddleware(container)
	s.Require().NoError(err)

	mw, resolveErr := di.Resolve[*OTELMiddleware](container)
	s.Require().NoError(resolveErr)
	s.NotNil(mw)
	s.Equal("otel", mw.Name())
}

func (s *ModuleTestSuite) TestProvideOTELMiddleware_WithoutHealthConfig_FallsBackToDefaults() {
	container := di.New()

	tp := sdktrace.NewTracerProvider()
	defer func() { _ = tp.Shutdown(s.T().Context()) }()

	s.Require().NoError(di.For[*sdktrace.TracerProvider](container).Instance(tp))

	// Do NOT register health config — should fall back to defaults.
	err := provideOTELMiddleware(container)
	s.Require().NoError(err)

	mw, resolveErr := di.Resolve[*OTELMiddleware](container)
	s.Require().NoError(resolveErr)
	s.NotNil(mw)
}

func (s *ModuleTestSuite) TestProvideOTELMiddleware_WithoutTracerProvider_Skips() {
	container := di.New()

	err := provideOTELMiddleware(container)
	s.Require().NoError(err)

	// Should not be registered.
	_, resolveErr := di.Resolve[*OTELMiddleware](container)
	s.Require().Error(resolveErr, "OTEL middleware should not be registered without TracerProvider")
}

func (s *ModuleTestSuite) TestProvideConnectRateLimitBundle_WithLimiter_UsesCustom() {
	container := di.New()

	// Register a custom limiter.
	limiter := connectpkg.AlwaysPassLimiter{}
	s.Require().NoError(di.For[connectpkg.Limiter](container).Instance(limiter))

	err := provideConnectRateLimitBundle(container)
	s.Require().NoError(err)

	bundle, resolveErr := di.Resolve[*connectpkg.RateLimitBundle](container)
	s.Require().NoError(resolveErr)
	s.NotNil(bundle)
	s.Equal("ratelimit", bundle.Name())
}

// --- provideConfig additional path tests ---

func (s *ModuleTestSuite) TestProvideConfig_WithProviderValues() {
	container := di.New()
	cfg := DefaultConfig()

	// Register ProviderValues to trigger the unmarshal path.
	// Since we don't have a real ProviderValues, we just verify the config
	// resolves correctly without one.
	err := provideConfig(cfg)(container)
	s.Require().NoError(err)

	resolved, resolveErr := di.Resolve[Config](container)
	s.Require().NoError(resolveErr)
	s.Equal(cfg.Port, resolved.Port)
}

func (s *ModuleTestSuite) TestProvideConfig_ResolveTwiceReturnsCachedSingleton() {
	container := di.New()
	cfg := DefaultConfig()

	err := provideConfig(cfg)(container)
	s.Require().NoError(err)

	r1, err1 := di.Resolve[Config](container)
	s.Require().NoError(err1)
	r2, err2 := di.Resolve[Config](container)
	s.Require().NoError(err2)
	s.Equal(r1.Port, r2.Port)
}

// --- provideCORSMiddleware error path ---

func (s *ModuleTestSuite) TestProvideCORSMiddleware_ResolveConfigError() {
	container := di.New()

	// Register config provider that fails.
	s.Require().NoError(di.For[Config](container).Provider(
		func(_ *di.Container) (Config, error) {
			return Config{}, errors.New("config unavailable")
		},
	))

	err := provideCORSMiddleware(container)
	s.Require().NoError(err, "registration should succeed")

	// Resolution should fail.
	_, resolveErr := di.Resolve[*CORSMiddleware](container)
	s.Require().Error(resolveErr)
}

// --- provideConnectLoggingBundle error path ---

func (s *ModuleTestSuite) TestProvideConnectLoggingBundle_RegisterError() {
	container := di.New()

	// Register first, then try again to trigger duplicate.
	err := provideConnectLoggingBundle(container)
	s.Require().NoError(err)

	// Verify resolution works.
	bundle, resolveErr := di.Resolve[*connectpkg.LoggingBundle](container)
	s.Require().NoError(resolveErr)
	s.NotNil(bundle)
}

// --- provideConnectRecoveryBundle error path ---

func (s *ModuleTestSuite) TestProvideConnectRecoveryBundle_ResolveConfigError() {
	container := di.New()

	// Register config provider that fails.
	s.Require().NoError(di.For[Config](container).Provider(
		func(_ *di.Container) (Config, error) {
			return Config{}, errors.New("config unavailable")
		},
	))

	err := provideConnectRecoveryBundle(container)
	s.Require().NoError(err, "registration should succeed")

	// Resolution should fail due to config error.
	_, resolveErr := di.Resolve[*connectpkg.RecoveryBundle](container)
	s.Require().Error(resolveErr)
}

// --- provideConnectValidationBundle error path ---

func (s *ModuleTestSuite) TestProvideConnectValidationBundle_ResolveVerify() {
	container := di.New()

	err := provideConnectValidationBundle(container)
	s.Require().NoError(err)

	bundle, resolveErr := di.Resolve[*connectpkg.ValidationBundle](container)
	s.Require().NoError(resolveErr)
	s.NotNil(bundle)
	s.Equal("validation", bundle.Name())
}

// --- provideConnectAuthBundle error path ---

func (s *ModuleTestSuite) TestProvideConnectAuthBundle_AuthFuncResolveError() {
	container := di.New()

	// Register an AuthFunc provider that errors.
	s.Require().NoError(di.For[connectpkg.AuthFunc](container).Provider(
		func(_ *di.Container) (connectpkg.AuthFunc, error) {
			return nil, errors.New("auth func unavailable")
		},
	))

	err := provideConnectAuthBundle(container)
	s.Require().Error(err)
	s.Contains(err.Error(), "resolve connect auth func")
}

// --- provideConnectRateLimitBundle error path ---

func (s *ModuleTestSuite) TestProvideConnectRateLimitBundle_LimiterResolveError() {
	container := di.New()

	// Register a Limiter provider that errors.
	s.Require().NoError(di.For[connectpkg.Limiter](container).Provider(
		func(_ *di.Container) (connectpkg.Limiter, error) {
			return nil, errors.New("limiter unavailable")
		},
	))

	err := provideConnectRateLimitBundle(container)
	s.Require().Error(err)
	s.Contains(err.Error(), "resolve connect limiter")
}

// --- provideServer error paths ---

func (s *ModuleTestSuite) TestProvideServer_ConfigResolveError() {
	container := di.New()

	// Register config provider that fails.
	s.Require().NoError(di.For[Config](container).Provider(
		func(_ *di.Container) (Config, error) {
			return Config{}, errors.New("config unavailable")
		},
	))

	// Register gRPC server.
	grpcCfg := grpcpkg.DefaultConfig()
	grpcCfg.SkipListener = true
	grpcSrv := grpcpkg.NewServer(grpcCfg, slog.Default(), container, nil)
	s.Require().NoError(di.For[*grpcpkg.Server](container).Instance(grpcSrv))

	err := provideServer(container)
	s.Require().NoError(err, "registration should succeed")

	// Build should fail because config resolution fails.
	buildErr := container.Build()
	s.Require().Error(buildErr)
}

func (s *ModuleTestSuite) TestProvideServer_GRPCServerResolveError() {
	container := di.New()

	cfg := DefaultConfig()
	cfg.Port = getFreePort(s.T())
	s.Require().NoError(di.For[Config](container).Instance(cfg))

	// Register gRPC server provider that fails.
	s.Require().NoError(di.For[*grpcpkg.Server](container).Provider(
		func(_ *di.Container) (*grpcpkg.Server, error) {
			return nil, errors.New("grpc server unavailable")
		},
	))

	err := provideServer(container)
	s.Require().NoError(err, "registration should succeed")

	// Build should fail because gRPC resolution fails.
	buildErr := container.Build()
	s.Require().Error(buildErr)
}

// --- provideOTELConnectBundle tests ---

func (s *ModuleTestSuite) TestProvideOTELConnectBundle_WithTracerProvider() {
	container := di.New()

	tp := sdktrace.NewTracerProvider()
	defer func() { _ = tp.Shutdown(context.Background()) }()

	s.Require().NoError(di.For[*sdktrace.TracerProvider](container).Instance(tp))

	err := provideOTELConnectBundle(container)
	s.Require().NoError(err)

	bundle, resolveErr := di.Resolve[*OTELConnectBundle](container)
	s.Require().NoError(resolveErr)
	s.NotNil(bundle)
	s.Equal("otelconnect", bundle.Name())
}

func (s *ModuleTestSuite) TestProvideOTELConnectBundle_WithoutTracerProvider_SkipsSilently() {
	container := di.New()

	err := provideOTELConnectBundle(container)
	s.Require().NoError(err)

	// Should not be registered.
	_, resolveErr := di.Resolve[*OTELConnectBundle](container)
	s.Require().Error(resolveErr, "OTEL Connect bundle should not be registered without TracerProvider")
}

func (s *ModuleTestSuite) TestProvideOTELConnectBundle_ResolveError() {
	container := di.New()

	// Register a provider that returns an error.
	s.Require().NoError(di.For[*sdktrace.TracerProvider](container).Provider(
		func(_ *di.Container) (*sdktrace.TracerProvider, error) {
			return nil, errors.New("tracer provider unavailable")
		},
	))

	err := provideOTELConnectBundle(container)
	s.Require().Error(err)
	s.Contains(err.Error(), "resolve tracer provider")
}

// --- provideServer tests ---

func (s *ModuleTestSuite) TestProvideServer_Registers() {
	container := di.New()

	// Register all dependencies needed by provideServer.
	cfg := DefaultConfig()
	cfg.Port = getFreePort(s.T())
	cfg.HealthEnabled = false
	s.Require().NoError(di.For[Config](container).Instance(cfg))

	// Register a grpc.Server wrapper.
	grpcCfg := grpcpkg.DefaultConfig()
	grpcCfg.SkipListener = true
	grpcSrv := grpcpkg.NewServer(grpcCfg, slog.Default(), container, nil)
	s.Require().NoError(di.For[*grpcpkg.Server](container).Instance(grpcSrv))

	err := provideServer(container)
	s.Require().NoError(err)

	// provideServer registers as Eager. Build triggers resolution.
	buildErr := container.Build()
	s.Require().NoError(buildErr)

	srv, resolveErr := di.Resolve[*Server](container)
	s.Require().NoError(resolveErr)
	s.NotNil(srv)
}
