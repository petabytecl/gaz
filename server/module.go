package server

import (
	"github.com/petabytecl/gaz"
)

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
	return newUnifiedServerBridge().
		configure(gaz.NewModule(ModuleName)).
		Build()
}
