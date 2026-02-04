package gateway

import (
	"fmt"
	"log/slog"
	"net/http"

	sdktrace "go.opentelemetry.io/otel/sdk/trace"

	"github.com/petabytecl/gaz"
	servergrpc "github.com/petabytecl/gaz/server/grpc"
	serverhttp "github.com/petabytecl/gaz/server/http"
)

// NewModule creates a Gateway module.
// Returns a gaz.Module that registers Gateway components.
//
// Components registered:
//   - gateway.Config (loaded from flags/config)
//   - *gateway.Gateway (eager, initializes on app start)
//   - http.Handler (via Gateway.Handler())
//   - Includes server/http module to provide the HTTP server
//
// Example:
//
//	app := gaz.New()
//	app.Use(gateway.NewModule())
func NewModule() gaz.Module {
	defaultCfg := DefaultConfig()
	return gaz.NewModule("gateway").
		Flags(defaultCfg.Flags).
		Use(serverhttp.NewModule()). // Use http module to provide the server
		Provide(func(c *gaz.Container) error {
			// Register Config provider
			if err := gaz.For[Config](c).Provider(func(c *gaz.Container) (Config, error) {
				// Start with the default configuration which has flags bound to it
				cfg := defaultCfg

				// Resolve ProviderValues to load config
				if pv, err := gaz.Resolve[*gaz.ProviderValues](c); err == nil {
					if unmarshalErr := pv.UnmarshalKey(cfg.Namespace(), &cfg); unmarshalErr != nil {
						// ignore error, use defaults
						_ = unmarshalErr
					}
				}

				// Re-apply CORS defaults if not set, respecting the loaded DevMode
				if len(cfg.CORS.AllowedOrigins) == 0 {
					cfg.CORS = DefaultCORSConfig(cfg.DevMode)
				}

				if err := cfg.Validate(); err != nil {
					return Config{}, fmt.Errorf("gateway config validate: %w", err)
				}

				return cfg, nil
			}); err != nil {
				return fmt.Errorf("register config: %w", err)
			}
			return nil
		}).
		Provide(provideGateway).
		Provide(provideHandler).
		Build()
}

func provideGateway(c *gaz.Container) error {
	// Register Gateway
	if err := gaz.For[*Gateway](c).
		Eager().
		Provider(newGatewayProvider); err != nil {
		return fmt.Errorf("register gateway: %w", err)
	}
	return nil
}

func newGatewayProvider(c *gaz.Container) (*Gateway, error) {
	cfg, err := gaz.Resolve[Config](c)
	if err != nil {
		return nil, fmt.Errorf("resolve gateway config: %w", err)
	}

	// Auto-configure GRPCTarget from grpc.Config if available and using defaults
	if cfg.GRPCTarget == DefaultGRPCTarget {
		if grpcCfg, resolveErr := gaz.Resolve[servergrpc.Config](c); resolveErr == nil {
			// If gRPC config is available in the same container, point to it.
			// This handles the case where --grpc-port is changed but gateway target isn't.
			cfg.GRPCTarget = fmt.Sprintf("localhost:%d", grpcCfg.Port)
		}
	}

	logger := slog.Default()
	if resolved, resolveErr := gaz.Resolve[*slog.Logger](c); resolveErr == nil {
		logger = resolved
	}

	// Try to resolve TracerProvider (optional).
	var tp *sdktrace.TracerProvider
	if resolved, resolveErr := gaz.Resolve[*sdktrace.TracerProvider](c); resolveErr == nil {
		tp = resolved
	}

	return NewGateway(cfg, logger, c, cfg.DevMode, tp), nil
}

func provideHandler(c *gaz.Container) error {
	// Register Gateway as http.Handler so http.Server can use it
	if err := gaz.For[http.Handler](c).Provider(func(c *gaz.Container) (http.Handler, error) {
		gw, err := gaz.Resolve[*Gateway](c)
		if err != nil {
			return nil, fmt.Errorf("resolve gateway for handler: %w", err)
		}
		return gw.Handler(), nil
	}); err != nil {
		return fmt.Errorf("register gateway handler: %w", err)
	}
	return nil
}
