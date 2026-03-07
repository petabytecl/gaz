// Package vanguard provides a unified server that serves gRPC, Connect,
// gRPC-Web, and REST protocols on a single port via Vanguard transcoder
// with h2c support.
//
// # Overview
//
// This package implements the core v5.0 server that composes multiple protocol
// handlers into a single HTTP endpoint. It uses connectrpc.com/vanguard to
// transcode between gRPC, Connect, gRPC-Web, and REST (via google.api.http
// annotations) without any code generation.
//
// Services are auto-discovered from the DI container:
//   - Connect services implement [connect.Registrar] and are resolved via
//     di.ResolveAll[connect.Registrar].
//   - gRPC services are bridged through the gRPC server's raw *grpc.Server
//     via vanguardgrpc.NewTranscoder.
//
// # Quick Start
//
// Use the module to register the Vanguard server with your application:
//
//	app := gaz.New()
//	app.Use(grpc.NewModule())      // gRPC services (skip-listener mode)
//	app.Use(vanguard.NewModule())  // Vanguard unified server
//
// # Health Endpoints
//
// When a health.Manager is present in the DI container, the server
// automatically mounts health endpoints on the unknown handler:
//   - /healthz — readiness probe
//   - /readyz  — readiness probe
//   - /livez   — liveness probe
//
// # Reflection
//
// gRPC reflection (v1 and v1alpha) is enabled by default for grpcurl
// compatibility. Reflection handlers are registered as Connect-style
// services in the Vanguard transcoder.
//
// # Configuration
//
// Configuration uses the "server" namespace:
//
//	server:
//	  port: 8080
//	  read_header_timeout: 5s
//	  idle_timeout: 120s
//	  reflection: true
//	  health_enabled: true
package vanguard
