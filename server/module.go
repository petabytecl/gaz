package server

import (
	"github.com/petabytecl/gaz"
	"github.com/petabytecl/gaz/server/gateway"
	"github.com/petabytecl/gaz/server/grpc"
)

// NewModule creates a unified server module.
// Returns a gaz.Module that bundles gRPC and Gateway modules.
//
// This module ensures correct startup order:
//   - gRPC server starts first (port binding and service registration)
//   - Gateway server starts second (depends on gRPC being available)
//
// Shutdown occurs in reverse order (Gateway first, then gRPC).
//
// Configuration:
//   - gRPC: "grpc-port", "grpc-reflection", "grpc-dev-mode" flags
//   - Gateway: "gateway-port", "gateway-grpc-target", "gateway-dev-mode" flags
//   - HTTP: "http-port", timeouts via flags (provided by gateway module's use of http module)
//
// Example:
//
//	app := gaz.New()
//	app.Use(server.NewModule())
func NewModule() gaz.Module {
	return gaz.NewModule("server").
		Use(grpc.NewModule()).
		Use(gateway.NewModule()).
		Build()
}
