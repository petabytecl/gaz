package vanguard

import (
	"fmt"
	"log/slog"

	"github.com/petabytecl/gaz"
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
		Provide(provideServer).
		Build()
}
