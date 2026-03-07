package vanguard

import (
	"context"
	"log/slog"
	"net/http"
	"testing"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/suite"

	"github.com/petabytecl/gaz/di"
	connectpkg "github.com/petabytecl/gaz/server/connect"
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
