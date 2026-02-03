package gateway

import (
	"fmt"
	"log/slog"

	"github.com/spf13/pflag"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"

	"github.com/petabytecl/gaz/di"
)

// ModuleOption configures the Gateway module.
type ModuleOption func(*moduleConfig)

type moduleConfig struct {
	port       int
	grpcTarget string
	cors       *CORSConfig
	devMode    bool
}

func defaultModuleConfig() *moduleConfig {
	return &moduleConfig{
		port:       DefaultPort,
		grpcTarget: "",  // Will use DefaultGRPCTarget if empty.
		cors:       nil, // Will use DefaultCORSConfig based on devMode.
		devMode:    false,
	}
}

// WithPort sets the Gateway HTTP port. Default is 8080.
func WithPort(port int) ModuleOption {
	return func(c *moduleConfig) {
		c.port = port
	}
}

// WithGRPCTarget sets the gRPC server target for loopback connections.
// Default is "localhost:50051".
func WithGRPCTarget(target string) ModuleOption {
	return func(c *moduleConfig) {
		c.grpcTarget = target
	}
}

// WithDevMode enables development mode.
// In dev mode:
//   - CORS is permissive (allows all origins and headers)
//   - Error responses include detailed gRPC error messages
//
// Default is false.
func WithDevMode(enabled bool) ModuleOption {
	return func(c *moduleConfig) {
		c.devMode = enabled
	}
}

// WithCORS sets custom CORS configuration.
// If not set, DefaultCORSConfig is used based on devMode.
func WithCORS(cfg CORSConfig) ModuleOption {
	return func(c *moduleConfig) {
		c.cors = &cfg
	}
}

// NewModule creates a Gateway module with the given options.
// Returns a di.Module that registers Gateway components.
//
// Components registered:
//   - gateway.Config (from options or defaults)
//   - *gateway.Gateway (eager, initializes on app start)
//
// Example:
//
//	app := gaz.New()
//	app.Use(gateway.NewModule())                           // defaults
//	app.Use(gateway.NewModule(gateway.WithPort(9000)))     // custom port
//	app.Use(gateway.NewModule(gateway.WithDevMode(true)))  // dev mode
func NewModule(opts ...ModuleOption) di.Module {
	cfg := defaultModuleConfig()
	for _, opt := range opts {
		opt(cfg)
	}

	return di.NewModuleFunc("gateway", func(c *di.Container) error {
		// Determine CORS config.
		corsConfig := cfg.cors
		if corsConfig == nil {
			defaultCORS := DefaultCORSConfig(cfg.devMode)
			corsConfig = &defaultCORS
		}

		// Build Gateway config.
		gatewayCfg := Config{
			Port:       cfg.port,
			GRPCTarget: cfg.grpcTarget,
			CORS:       *corsConfig,
		}

		// Register Config.
		if err := di.For[Config](c).Instance(gatewayCfg); err != nil {
			return fmt.Errorf("register gateway config: %w", err)
		}

		// Delegate to Module for component registration.
		return Module(c, cfg.devMode)
	})
}

// NewModuleWithFlags creates a Gateway module with CLI flag support.
// The flags are registered with the provided FlagSet and their values
// are read when the module registers components.
//
// Flags registered:
//   - --gateway-port: Gateway HTTP port (default from options or 8080)
//   - --gateway-grpc-target: gRPC server target (default "localhost:50051")
//   - --gateway-dev-mode: Enable development mode (default from options or false)
//
// Example:
//
//	rootCmd := &cobra.Command{}
//	app := gaz.New()
//	app.Use(gateway.NewModuleWithFlags(rootCmd.Flags()))
//	// Now --gateway-port, --gateway-grpc-target, --gateway-dev-mode are available.
func NewModuleWithFlags(fs *pflag.FlagSet, opts ...ModuleOption) di.Module {
	cfg := defaultModuleConfig()
	for _, opt := range opts {
		opt(cfg)
	}

	// Define flags with initial values from options.
	portFlag := fs.Int("gateway-port", cfg.port, "Gateway HTTP port")
	grpcTargetFlag := fs.String("gateway-grpc-target", "", "gRPC server target (default: localhost:<grpc-port>)")
	devModeFlag := fs.Bool("gateway-dev-mode", cfg.devMode, "Enable development mode")

	return di.NewModuleFunc("gateway", func(c *di.Container) error {
		// Read flag values (deferred evaluation).
		port := *portFlag
		grpcTarget := *grpcTargetFlag
		devMode := *devModeFlag

		// Use grpcTarget from flags if set, otherwise from options.
		if grpcTarget == "" {
			grpcTarget = cfg.grpcTarget
		}

		// Determine CORS config.
		corsConfig := cfg.cors
		if corsConfig == nil {
			defaultCORS := DefaultCORSConfig(devMode)
			corsConfig = &defaultCORS
		}

		// Build Gateway config.
		gatewayCfg := Config{
			Port:       port,
			GRPCTarget: grpcTarget,
			CORS:       *corsConfig,
		}

		// Register Config.
		if err := di.For[Config](c).Instance(gatewayCfg); err != nil {
			return fmt.Errorf("register gateway config: %w", err)
		}

		// Delegate to Module for component registration.
		return Module(c, devMode)
	})
}

// Module registers the Gateway module components.
// It provides:
//   - *Gateway (eager, initializes on app start)
//
// It assumes that gateway.Config has been registered in the container
// (e.g., via NewModule or manual registration).
//
// The devMode parameter controls whether detailed error messages are exposed
// and whether CORS debugging is enabled.
//
// If a *sdktrace.TracerProvider is registered in the container, the gateway
// will be instrumented with OpenTelemetry tracing.
func Module(c *di.Container, devMode bool) error {
	// Register Gateway (implements di.Starter and di.Stopper).
	if err := di.For[*Gateway](c).
		Eager().
		Provider(func(c *di.Container) (*Gateway, error) {
			cfg, err := di.Resolve[Config](c)
			if err != nil {
				return nil, fmt.Errorf("resolve gateway config: %w", err)
			}

			logger, err := di.Resolve[*slog.Logger](c)
			if err != nil {
				return nil, fmt.Errorf("resolve logger: %w", err)
			}

			// Try to resolve TracerProvider (optional).
			// If not found or nil, OTEL tracing is disabled.
			var tp *sdktrace.TracerProvider
			if resolved, resolveErr := di.Resolve[*sdktrace.TracerProvider](c); resolveErr == nil {
				tp = resolved
			}

			return NewGateway(cfg, logger, c, devMode, tp), nil
		}); err != nil {
		return fmt.Errorf("register gateway: %w", err)
	}

	return nil
}
