package server

import (
	"fmt"

	"github.com/petabytecl/gaz"
	"github.com/petabytecl/gaz/internal/configuredmodule"
	"github.com/petabytecl/gaz/server/grpc"
	"github.com/petabytecl/gaz/server/vanguard"
)

// unifiedServerBridge owns the single-port transport composition rule.
//
// Standalone gRPC and Vanguard modules remain usable on their own. The bridge
// adds only the policy needed when callers choose server.NewModule(): gRPC
// registers services without binding a listener, and Vanguard owns the public
// listener for all protocols.
type unifiedServerBridge struct {
	grpcModule     gaz.Module
	vanguardModule gaz.Module
}

func newUnifiedServerBridge() unifiedServerBridge {
	return unifiedServerBridge{
		grpcModule:     grpc.NewModule(),
		vanguardModule: vanguard.NewModule(),
	}
}

func (b unifiedServerBridge) configure(builder *gaz.ModuleBuilder) *gaz.ModuleBuilder {
	return builder.
		Use(b.grpcModule).
		Use(b.vanguardModule).
		Provide(forceGRPCServiceRegistrationOnly)
}

func forceGRPCServiceRegistrationOnly(c *gaz.Container) error {
	if err := gaz.For[grpc.Config](c).Replace().Provider(grpcBridgeConfig); err != nil {
		return fmt.Errorf("override grpc config: %w", err)
	}
	return nil
}

func grpcBridgeConfig(c *gaz.Container) (grpc.Config, error) {
	cfg := grpc.DefaultConfig()
	if err := overlayGRPCProviderValues(c, &cfg); err != nil {
		return grpc.Config{}, err
	}

	cfg.SkipListener = true

	if err := cfg.Validate(); err != nil {
		return grpc.Config{}, fmt.Errorf("grpc config validate: %w", err)
	}
	return cfg, nil
}

func overlayGRPCProviderValues(c *gaz.Container, cfg *grpc.Config) error {
	if err := configuredmodule.OverlayProviderValues(c, cfg); err != nil {
		return fmt.Errorf("overlay grpc provider values: %w", err)
	}
	return nil
}
