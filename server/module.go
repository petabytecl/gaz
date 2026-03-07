package server

import (
	"fmt"

	"github.com/petabytecl/gaz"
	"github.com/petabytecl/gaz/server/grpc"
	"github.com/petabytecl/gaz/server/vanguard"
)

// forceSkipListener overrides the gRPC Config to set SkipListener=true.
// This ensures gRPC registers services and interceptors but does not bind
// its own listener — Vanguard handles all connections on a single port.
func forceSkipListener(c *gaz.Container) error {
	if err := gaz.For[grpc.Config](c).Replace().Provider(func(c *gaz.Container) (grpc.Config, error) {
		cfg := grpc.DefaultConfig()

		if pv, err := gaz.Resolve[*gaz.ProviderValues](c); err == nil {
			if unmarshalErr := pv.UnmarshalKey(cfg.Namespace(), &cfg); unmarshalErr != nil {
				_ = unmarshalErr
			}
		}

		cfg.SkipListener = true

		if err := cfg.Validate(); err != nil {
			return grpc.Config{}, fmt.Errorf("grpc config validate: %w", err)
		}

		return cfg, nil
	}); err != nil {
		return fmt.Errorf("override grpc config: %w", err)
	}
	return nil
}

// NewModule creates a unified server module.
// Returns a gaz.Module that bundles gRPC and Vanguard modules with gRPC
// SkipListener automatically set to true.
//
// The module composes two child modules:
//   - grpc.NewModule(): gRPC server with interceptors, reflection, health
//   - vanguard.NewModule(): Vanguard unified server (gRPC, Connect, gRPC-Web, REST)
//
// Startup order:
//   - gRPC registers services and interceptors (without binding a listener)
//   - Vanguard builds the transcoder and serves all protocols on a single h2c port
//
// Shutdown order:
//   - Vanguard stops first (drains HTTP connections)
//   - gRPC stops second (closes service registrations)
//
// Configuration:
//   - gRPC: "grpc-port", "grpc-reflection", "grpc-dev-mode" flags (port unused with SkipListener)
//   - Vanguard: "vanguard-address", "vanguard-dev-mode", CORS and timeout flags
//
// Example:
//
//	app := gaz.New()
//	app.Use(server.NewModule())
func NewModule() gaz.Module {
	return gaz.NewModule("server").
		Use(grpc.NewModule()).
		Use(vanguard.NewModule()).
		Provide(forceSkipListener).
		Build()
}
