package vanguard

import (
	"fmt"
	"log/slog"

	sdktrace "go.opentelemetry.io/otel/sdk/trace"

	"github.com/petabytecl/gaz"
	connectpkg "github.com/petabytecl/gaz/server/connect"
	grpcpkg "github.com/petabytecl/gaz/server/grpc"
)

// resolveLogger attempts to resolve a logger from the container, falling back to slog.Default().
func resolveLogger(c *gaz.Container) *slog.Logger {
	if resolved, err := gaz.Resolve[*slog.Logger](c); err == nil {
		return resolved
	}
	return slog.Default()
}

// provideConfig creates a Config provider function.
func provideConfig(defaultCfg Config) func(*gaz.Container) error {
	return func(c *gaz.Container) error {
		return gaz.For[Config](c).Provider(func(c *gaz.Container) (Config, error) {
			cfg := defaultCfg

			if pv, err := gaz.Resolve[*gaz.ProviderValues](c); err == nil {
				if unmarshalErr := pv.UnmarshalKey(defaultCfg.Namespace(), &cfg); unmarshalErr != nil {
					_ = unmarshalErr
				}
			}

			if err := cfg.Validate(); err != nil {
				return Config{}, fmt.Errorf("vanguard config validate: %w", err)
			}

			return cfg, nil
		})
	}
}

// provideCORSMiddleware registers a CORSMiddleware in the DI container.
// The CORS middleware is always registered and uses the Config's CORS settings and DevMode flag.
func provideCORSMiddleware(c *gaz.Container) error {
	if err := gaz.For[*CORSMiddleware](c).Provider(func(c *gaz.Container) (*CORSMiddleware, error) {
		cfg, err := gaz.Resolve[Config](c)
		if err != nil {
			return nil, fmt.Errorf("resolve vanguard config: %w", err)
		}
		return NewCORSMiddleware(cfg.CORS, cfg.DevMode), nil
	}); err != nil {
		return fmt.Errorf("register cors middleware: %w", err)
	}
	return nil
}

// provideOTELMiddleware registers an OTELMiddleware in the DI container.
// The middleware is only registered if a TracerProvider is available in DI.
func provideOTELMiddleware(c *gaz.Container) error {
	if !gaz.Has[*sdktrace.TracerProvider](c) {
		// No TracerProvider — skip OTEL middleware silently.
		return nil
	}

	tp, resolveErr := gaz.Resolve[*sdktrace.TracerProvider](c)
	if resolveErr != nil {
		return fmt.Errorf("resolve tracer provider: %w", resolveErr)
	}

	if regErr := gaz.For[*OTELMiddleware](c).Provider(func(_ *gaz.Container) (*OTELMiddleware, error) {
		return NewOTELMiddleware(tp), nil
	}); regErr != nil {
		return fmt.Errorf("register otel middleware: %w", regErr)
	}
	return nil
}

// provideOTELConnectBundle registers an OTELConnectBundle in the DI container.
// The bundle is only registered if a TracerProvider is available in DI.
func provideOTELConnectBundle(c *gaz.Container) error {
	if !gaz.Has[*sdktrace.TracerProvider](c) {
		// No TracerProvider — skip OTEL Connect bundle silently.
		return nil
	}

	tp, resolveErr := gaz.Resolve[*sdktrace.TracerProvider](c)
	if resolveErr != nil {
		return fmt.Errorf("resolve tracer provider: %w", resolveErr)
	}

	if regErr := gaz.For[*OTELConnectBundle](c).Provider(func(c *gaz.Container) (*OTELConnectBundle, error) {
		return NewOTELConnectBundle(tp, resolveLogger(c)), nil
	}); regErr != nil {
		return fmt.Errorf("register otel connect bundle: %w", regErr)
	}
	return nil
}

// provideConnectLoggingBundle registers a connect.LoggingBundle in the DI container.
func provideConnectLoggingBundle(c *gaz.Container) error {
	if err := gaz.For[*connectpkg.LoggingBundle](c).Provider(func(c *gaz.Container) (*connectpkg.LoggingBundle, error) {
		return connectpkg.NewLoggingBundle(resolveLogger(c)), nil
	}); err != nil {
		return fmt.Errorf("register connect logging bundle: %w", err)
	}
	return nil
}

// provideConnectRecoveryBundle registers a connect.RecoveryBundle in the DI container.
func provideConnectRecoveryBundle(c *gaz.Container) error {
	if err := gaz.For[*connectpkg.RecoveryBundle](c).Provider(func(c *gaz.Container) (*connectpkg.RecoveryBundle, error) {
		cfg, cfgErr := gaz.Resolve[Config](c)
		if cfgErr != nil {
			return nil, fmt.Errorf("resolve vanguard config: %w", cfgErr)
		}
		return connectpkg.NewRecoveryBundle(resolveLogger(c), cfg.DevMode), nil
	}); err != nil {
		return fmt.Errorf("register connect recovery bundle: %w", err)
	}
	return nil
}

// provideConnectValidationBundle registers a connect.ValidationBundle in the DI container.
func provideConnectValidationBundle(c *gaz.Container) error {
	if err := gaz.For[*connectpkg.ValidationBundle](c).Provider(func(_ *gaz.Container) (*connectpkg.ValidationBundle, error) {
		return connectpkg.NewValidationBundle(), nil
	}); err != nil {
		return fmt.Errorf("register connect validation bundle: %w", err)
	}
	return nil
}

// provideConnectAuthBundle registers a connect.AuthBundle in the DI container.
// The bundle is only registered if a ConnectAuthFunc is available in DI.
// This makes authentication opt-in — services without ConnectAuthFunc skip auth.
func provideConnectAuthBundle(c *gaz.Container) error {
	if !gaz.Has[connectpkg.ConnectAuthFunc](c) {
		// No ConnectAuthFunc registered — skip auth interceptor silently.
		return nil
	}

	authFunc, resolveErr := gaz.Resolve[connectpkg.ConnectAuthFunc](c)
	if resolveErr != nil {
		return fmt.Errorf("resolve connect auth func: %w", resolveErr)
	}

	if regErr := gaz.For[*connectpkg.AuthBundle](c).Provider(func(_ *gaz.Container) (*connectpkg.AuthBundle, error) {
		return connectpkg.NewAuthBundle(authFunc), nil
	}); regErr != nil {
		return fmt.Errorf("register connect auth bundle: %w", regErr)
	}
	return nil
}

// provideConnectRateLimitBundle registers a connect.RateLimitBundle in the DI container.
// If a ConnectLimiter is registered in DI, it uses that limiter.
// Otherwise, it registers a bundle with AlwaysPassLimiter (allows all requests).
func provideConnectRateLimitBundle(c *gaz.Container) error {
	var limiter connectpkg.ConnectLimiter
	if gaz.Has[connectpkg.ConnectLimiter](c) {
		resolved, resolveErr := gaz.Resolve[connectpkg.ConnectLimiter](c)
		if resolveErr != nil {
			return fmt.Errorf("resolve connect limiter: %w", resolveErr)
		}
		limiter = resolved
	}
	// limiter is nil if not registered, NewRateLimitBundle handles this.

	if regErr := gaz.For[*connectpkg.RateLimitBundle](c).Provider(func(_ *gaz.Container) (*connectpkg.RateLimitBundle, error) {
		return connectpkg.NewRateLimitBundle(limiter), nil
	}); regErr != nil {
		return fmt.Errorf("register connect ratelimit bundle: %w", regErr)
	}
	return nil
}

// provideServer creates a Server provider function.
// The server is registered as Eager so it starts with the application.
func provideServer(c *gaz.Container) error {
	if err := gaz.For[*Server](c).
		Eager().
		Provider(func(c *gaz.Container) (*Server, error) {
			cfg, err := gaz.Resolve[Config](c)
			if err != nil {
				return nil, fmt.Errorf("resolve vanguard config: %w", err)
			}

			// Resolve the gRPC server wrapper to get the raw *grpc.Server.
			grpcSrv, err := gaz.Resolve[*grpcpkg.Server](c)
			if err != nil {
				return nil, fmt.Errorf("resolve grpc server: %w", err)
			}

			return NewServer(cfg, resolveLogger(c), c, grpcSrv.GRPCServer()), nil
		}); err != nil {
		return fmt.Errorf("register vanguard server: %w", err)
	}
	return nil
}

// NewModule creates a Vanguard module.
// Returns a gaz.Module that registers Vanguard server components.
//
// Components registered:
//   - vanguard.Config (loaded from flags/config)
//   - *vanguard.CORSMiddleware (transport middleware, always registered)
//   - *vanguard.OTELMiddleware (transport middleware, only if TracerProvider registered)
//   - *vanguard.OTELConnectBundle (connect interceptor bundle, only if TracerProvider registered)
//   - *connect.LoggingBundle (connect logging interceptor, always registered)
//   - *connect.RecoveryBundle (connect panic recovery interceptor, always registered)
//   - *connect.ValidationBundle (connect protovalidate interceptor, always registered)
//   - *connect.AuthBundle (connect auth interceptor, only if ConnectAuthFunc registered)
//   - *connect.RateLimitBundle (connect rate limit interceptor, uses AlwaysPassLimiter unless ConnectLimiter registered)
//   - *vanguard.Server (eager, starts on app start)
//
// The module depends on grpc.NewModule() being registered first, as it
// resolves *grpc.Server from the DI container to bridge gRPC services
// into the Vanguard transcoder.
//
// Example:
//
//	app := gaz.New()
//	app.Use(grpc.NewModule())      // Must come first
//	app.Use(vanguard.NewModule())  // Vanguard unified server
func NewModule() gaz.Module {
	defaultCfg := DefaultConfig()

	return gaz.NewModule("vanguard").
		Flags(defaultCfg.Flags).
		Provide(provideConfig(defaultCfg)).
		Provide(provideCORSMiddleware).
		Provide(provideOTELMiddleware).
		Provide(provideOTELConnectBundle).
		Provide(provideConnectLoggingBundle).
		Provide(provideConnectRecoveryBundle).
		Provide(provideConnectValidationBundle).
		Provide(provideConnectAuthBundle).
		Provide(provideConnectRateLimitBundle).
		Provide(provideServer).
		Build()
}
