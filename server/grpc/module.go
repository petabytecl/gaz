package grpc

import (
	"fmt"
	"log/slog"

	sdktrace "go.opentelemetry.io/otel/sdk/trace"

	"github.com/petabytecl/gaz"
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
				return Config{}, fmt.Errorf("grpc config validate: %w", err)
			}

			return cfg, nil
		})
	}
}

// provideLoggingBundle creates a LoggingBundle provider function.
func provideLoggingBundle(c *gaz.Container) error {
	if err := gaz.For[*LoggingBundle](c).Provider(func(c *gaz.Container) (*LoggingBundle, error) {
		return NewLoggingBundle(resolveLogger(c)), nil
	}); err != nil {
		return fmt.Errorf("register logging bundle: %w", err)
	}
	return nil
}

// provideRecoveryBundle creates a RecoveryBundle provider function.
func provideRecoveryBundle(c *gaz.Container) error {
	if err := gaz.For[*RecoveryBundle](c).Provider(func(c *gaz.Container) (*RecoveryBundle, error) {
		cfg, err := gaz.Resolve[Config](c)
		if err != nil {
			return nil, fmt.Errorf("resolve grpc config: %w", err)
		}
		return NewRecoveryBundle(resolveLogger(c), cfg.DevMode), nil
	}); err != nil {
		return fmt.Errorf("register recovery bundle: %w", err)
	}
	return nil
}

// provideValidationBundle creates a ValidationBundle provider function.
func provideValidationBundle(c *gaz.Container) error {
	if err := gaz.For[*ValidationBundle](c).Provider(func(c *gaz.Container) (*ValidationBundle, error) {
		return NewValidationBundle()
	}); err != nil {
		return fmt.Errorf("register validation bundle: %w", err)
	}
	return nil
}

// provideServer creates a Server provider function.
func provideServer(c *gaz.Container) error {
	if err := gaz.For[*Server](c).
		Eager().
		Provider(func(c *gaz.Container) (*Server, error) {
			cfg, err := gaz.Resolve[Config](c)
			if err != nil {
				return nil, fmt.Errorf("resolve grpc config: %w", err)
			}

			var tp *sdktrace.TracerProvider
			if resolved, resolveErr := gaz.Resolve[*sdktrace.TracerProvider](c); resolveErr == nil {
				tp = resolved
			}

			return NewServer(cfg, resolveLogger(c), c, tp), nil
		}); err != nil {
		return fmt.Errorf("register server: %w", err)
	}
	return nil
}

// NewModule creates a gRPC module.
// Returns a gaz.Module that registers gRPC server components.
//
// Components registered:
//   - grpc.Config (loaded from flags/config)
//   - *grpc.LoggingBundle (logging interceptor)
//   - *grpc.ValidationBundle (protovalidate interceptor)
//   - *grpc.RecoveryBundle (panic recovery interceptor)
//   - *grpc.Server (eager, starts on app start)
//
// Custom interceptors can be added by registering implementations of
// InterceptorBundle in the DI container. They will be auto-discovered
// and chained based on their Priority().
//
// Example:
//
//	app := gaz.New()
//	app.Use(grpc.NewModule())
//
// Adding custom interceptors:
//
//	// Register your interceptor bundle
//	gaz.For[*MyInterceptor](c).Provider(NewMyInterceptor)
//
//	// MyInterceptor implements grpc.InterceptorBundle
//	type MyInterceptor struct{}
//	func (m *MyInterceptor) Name() string { return "my-interceptor" }
//	func (m *MyInterceptor) Priority() int { return 500 } // Between validation (100) and recovery (1000)
//	func (m *MyInterceptor) Interceptors() (grpc.UnaryServerInterceptor, grpc.StreamServerInterceptor) {
//	    return myUnaryInterceptor, myStreamInterceptor
//	}
func NewModule() gaz.Module {
	defaultCfg := DefaultConfig()

	return gaz.NewModule("grpc").
		Flags(defaultCfg.Flags).
		Provide(provideConfig(defaultCfg)).
		Provide(provideLoggingBundle).
		Provide(provideValidationBundle).
		Provide(provideRecoveryBundle).
		Provide(provideServer).
		Build()
}
