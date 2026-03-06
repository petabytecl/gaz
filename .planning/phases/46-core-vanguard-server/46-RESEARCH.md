# Phase 46: Core Vanguard Server - Research

**Researched:** 2026-03-06
**Domain:** Go unified HTTP/gRPC/Connect server with Vanguard transcoder
**Confidence:** HIGH

## Summary

Phase 46 delivers a single-port Vanguard server that serves gRPC, Connect, gRPC-Web, and REST (via `google.api.http` annotations) on one `http.Server` using h2c. The implementation creates two new packages — `server/connect/` for the `ConnectRegistrar` interface and `server/vanguard/` for the Vanguard server — while modifying `server/grpc/` to support a "skip listener" mode where it still creates and configures `*grpc.Server` but doesn't bind a port.

The architecture is well-constrained by existing patterns. The gRPC server's `Registrar` interface, `di.ResolveAll` auto-discovery, `Config` struct with `Flags()/SetDefaults()/Validate()/Namespace()`, and `OnStart`/`OnStop` lifecycle are all directly cloneable templates. The key new element is Vanguard's `NewTranscoder` composing gRPC services (via `vanguardgrpc.NewTranscoder`) and Connect services (via `vanguard.NewService`) into a single `http.Handler`, then serving it through Go 1.26's native h2c support via `http.Protocols.SetUnencryptedHTTP2(true)`.

Critical technical considerations: (1) Vanguard's transcoder is one-shot — all services must be registered before `vanguard.NewTranscoder` is called, so construction must happen in `OnStart` after DI resolution, not in the provider. (2) Server timeouts must be streaming-safe — `ReadTimeout=0` and `WriteTimeout=0` with `ReadHeaderTimeout=5s` for slowloris protection. (3) The gRPC server's skip-listener mode must still call `GracefulStop()` on shutdown even though no listener is involved. (4) Health endpoints mount via Vanguard's unknown handler mechanism, not via a separate mux on the transcoded routes.

**Primary recommendation:** Follow the existing `server/grpc/` patterns exactly for package structure, DI registration, config, module, and tests — then compose Vanguard transcoder from resolved gRPC and Connect services in `OnStart`.

<user_constraints>
## User Constraints (from CONTEXT.md)

### Locked Decisions
- **ConnectRegistrar Interface:** Single method `RegisterConnect() (string, http.Handler)` returning service path and handler. Lives in `server/connect/` package. Auto-discovered via `di.ResolveAll[ConnectRegistrar]` in Vanguard server's `OnStart`. gRPC reflection implemented as Connect handler using `connectrpc.com/grpcreflect`, registered as built-in ConnectRegistrar when reflection enabled.
- **Vanguard Server Architecture:** New `server/vanguard/` package owns the Vanguard transcoder and h2c `http.Server`. Creates its OWN `http.Server` — does NOT reuse `server/http` package. Wraps existing `*grpc.Server` via `vanguardgrpc.NewTranscoder`. Vanguard transcoder built during `OnStart` (one-shot construction). h2c via Go 1.26+ native `http.Protocols`.
- **gRPC Server Skip-Listener Mode:** Config flag `SkipListener bool` on gRPC `Config`. When true, `OnStart` discovers registrars, registers services, enables reflection, wires health — but skips `net.Listen` and `server.Serve()`. `OnStop` with `SkipListener` calls `GracefulStop()` directly. CLI flag: `--grpc-skip-listener` (default: false).
- **Non-RPC Handler Mounting:** Single fallback handler via Vanguard's unknown handler — NOT a full HTTP mux. Health endpoints auto-registered if `health.Manager` present. `SetUnknownHandler(h http.Handler)` method on Vanguard server.
- **Config & CLI Flags:** Flag prefix `server-` (NOT `vanguard-`). Default port 8080. Streaming-safe defaults: `ReadTimeout=0`, `WriteTimeout=0`, `ReadHeaderTimeout=5s`, `IdleTimeout=120s`. `--server-dev-mode` flag.

### Claude's Discretion
- Internal package structure within `server/vanguard/` (single file vs multiple files)
- Exact error messages and log field names (follow existing conventions)
- Whether to expose Vanguard-specific options (like service codecs) via config or keep them internal
- Test helper design in `gaztest` for Vanguard server testing

### Deferred Ideas (OUT OF SCOPE)
- CORS middleware for browser clients — Phase 47 (MDDL-01)
- Connect interceptor bundles (auth, logging, validation) — Phase 47 (CONN-02, CONN-03)
- OTEL instrumentation for Connect RPC layer — Phase 47 (MDDL-03)
- Proto constraint validation interceptor — Phase 47 (MDDL-04)
- Updated `server.NewModule()` bundling — Phase 48 (SMOD-01)
- Gateway package removal — Phase 48 (SMOD-02)
- Migration guide and examples — Deferred (MIGR-01, MIGR-02, EXMP-01)
</user_constraints>

<phase_requirements>
## Phase Requirements

| ID | Description | Research Support |
|----|-------------|-----------------|
| USRV-01 | Vanguard server serving gRPC, Connect, gRPC-Web, and REST on single http.Handler | Vanguard `NewTranscoder` composes all services into one `http.Handler`; `vanguardgrpc.NewTranscoder` bridges gRPC; `vanguard.NewService` bridges Connect handlers |
| USRV-02 | All protocols on single port using h2c via Go native `http.Protocols` | Go 1.26 `http.Protocols.SetUnencryptedHTTP2(true)` enables h2c natively; verified in Connect-Go official examples |
| USRV-03 | REST endpoints from proto `google.api.http` annotations without codegen | Vanguard transcoder handles REST transcoding automatically from proto annotations — no grpc-gateway codegen needed |
| USRV-04 | Browser clients via gRPC-Web without external proxy | Vanguard natively handles gRPC-Web protocol transcoding — free with the transcoder |
| USRV-05 | Custom HTTP handlers for non-RPC routes via unknown handler | `vanguard.WithUnknownHandler(handler)` option on `NewTranscoder` — verified in Vanguard docs |
| USRV-06 | Server address, timeouts, and options via CLI flags and config struct | Config struct with `Flags()/SetDefaults()/Validate()/Namespace()` — mirrors existing `server/grpc/Config` pattern |
| CONN-01 | ConnectRegistrar with auto-discovery through `di.ResolveAll` | `RegisterConnect() (string, http.Handler)` mirrors gRPC `Registrar` pattern; `di.ResolveAll[ConnectRegistrar]` in `OnStart` |
| CONN-04 | gRPC reflection for Connect services via `connectrpc.com/grpcreflect` | `grpcreflect.NewHandlerV1` and `NewHandlerV1Alpha` return `(string, http.Handler)` — register as built-in ConnectRegistrars |
| MDDL-05 | Health checks wired into unified Vanguard server | Health endpoints mount via unknown handler mux; auto-resolved from `health.Manager` in DI container |
</phase_requirements>

## Standard Stack

### Core
| Library | Version | Purpose | Why Standard |
|---------|---------|---------|--------------|
| `connectrpc.com/vanguard` | v0.4.0 | Unified transcoder for gRPC/Connect/gRPC-Web/REST | Only library that transcodes all four protocols from proto annotations without codegen |
| `connectrpc.com/connect` | v1.19.1 | Connect protocol handlers and interceptors | Official Connect-Go library, 4,556 importers, v1.x stable API |
| `connectrpc.com/grpcreflect` | latest | gRPC reflection for Connect services | Official reflection package from Buf; returns `(string, http.Handler)` tuple compatible with ConnectRegistrar |
| Go stdlib `net/http` | Go 1.26 | h2c-enabled HTTP server | Native `http.Protocols.SetUnencryptedHTTP2(true)` eliminates need for `golang.org/x/net/http2/h2c` |
| `google.golang.org/grpc` | v1.79.1 | gRPC server (already in go.mod) | Existing dependency; Vanguard wraps `*grpc.Server` via `vanguardgrpc.NewTranscoder` |

### Supporting
| Library | Version | Purpose | When to Use |
|---------|---------|---------|-------------|
| `connectrpc.com/vanguard/vanguardgrpc` | v0.4.0 (subpackage) | Bridge `*grpc.Server` into Vanguard transcoder | Always — enables existing gRPC Registrar services to be served through Vanguard |
| `github.com/petabytecl/gaz/health` | (internal) | Health check manager | Auto-wire health endpoints when `health.Manager` present in DI |
| `github.com/stretchr/testify` | v1.11.1 | Test assertions/suites | All tests follow existing suite pattern |

### Alternatives Considered
| Instead of | Could Use | Tradeoff |
|------------|-----------|----------|
| Vanguard | cmux (TCP multiplexing) | cmux does byte-sniffing which is fragile; Vanguard does HTTP-level dispatch which is correct |
| `http.Protocols` h2c | `golang.org/x/net/http2/h2c` | Old approach; Go 1.26 makes it unnecessary; native is simpler and better maintained |
| Vanguard unknown handler | Separate HTTP mux port | Defeats single-port goal; unknown handler is the correct abstraction |

### Installation

```bash
go get connectrpc.com/vanguard@v0.4.0
go get connectrpc.com/connect@v1.19.1
go get connectrpc.com/grpcreflect@latest
```

## Architecture Patterns

### Recommended Package Structure
```
server/
├── connect/
│   ├── doc.go              # Package doc
│   ├── registrar.go        # ConnectRegistrar interface
│   └── registrar_test.go   # Interface compliance tests
├── vanguard/
│   ├── doc.go              # Package doc
│   ├── config.go           # Config struct with Flags/SetDefaults/Validate/Namespace
│   ├── config_test.go      # Config validation tests
│   ├── server.go           # Server struct with OnStart/OnStop
│   ├── server_test.go      # Server lifecycle and protocol tests
│   ├── health.go           # Health endpoint mounting on unknown handler
│   ├── health_test.go      # Health endpoint tests
│   ├── module.go           # NewModule() with DI registration
│   └── module_test.go      # Module wiring tests
├── grpc/
│   ├── config.go           # MODIFIED: add SkipListener field
│   ├── server.go           # MODIFIED: skip-listener mode in OnStart/OnStop
│   └── ...                 # Existing files unchanged
└── ...
```

### Pattern 1: ConnectRegistrar Interface (mirrors gRPC Registrar)

**What:** Single-method interface returning `(string, http.Handler)` for Vanguard service registration.
**When to use:** Every Connect-Go service that should be auto-discovered.

```go
// Source: Mirrors server/grpc/server.go:32-34 (Registrar pattern)
package connect

import "net/http"

// ConnectRegistrar is implemented by Connect-Go services that want to be
// auto-discovered and registered with the Vanguard server.
//
// Implementations return the service path and HTTP handler:
//
//     type GreeterService struct {
//         greetv1connect.UnimplementedGreeterServiceHandler
//     }
//
//     func (s *GreeterService) RegisterConnect() (string, http.Handler) {
//         return greetv1connect.NewGreeterServiceHandler(s)
//     }
type ConnectRegistrar interface {
    RegisterConnect() (string, http.Handler)
}
```

**Why this works:** Connect-Go generated code produces `NewXxxServiceHandler(impl) (string, http.Handler)` — the ConnectRegistrar signature matches exactly. The `(string, http.Handler)` tuple maps directly to `vanguard.NewService(path, handler)`.

### Pattern 2: Vanguard Transcoder Construction in OnStart

**What:** Build the Vanguard transcoder in `OnStart` after all DI resolution, not in the provider function.
**When to use:** Always — Vanguard transcoder is one-shot (immutable after construction).

```go
// Source: Verified from Vanguard README and existing server/grpc/server.go OnStart pattern
func (s *Server) OnStart(ctx context.Context) error {
    // 1. Discover Connect services
    connectRegistrars, err := di.ResolveAll[connect.ConnectRegistrar](s.container)
    if err != nil {
        return fmt.Errorf("vanguard: discover connect services: %w", err)
    }

    // 2. Build Vanguard services from Connect registrars
    var services []*vanguard.Service
    for _, r := range connectRegistrars {
        path, handler := r.RegisterConnect()
        services = append(services, vanguard.NewService(path, handler))
    }

    // 3. Bridge gRPC server into Vanguard
    grpcServices := vanguardgrpc.NewTranscoder(s.grpcServer.GRPCServer())
    services = append(services, grpcServices...)

    // 4. Build transcoder options
    opts := []vanguard.TranscoderOption{}
    if s.unknownHandler != nil {
        opts = append(opts, vanguard.WithUnknownHandler(s.unknownHandler))
    }

    // 5. Create the one-shot transcoder
    transcoder, err := vanguard.NewTranscoder(services, opts...)
    if err != nil {
        return fmt.Errorf("vanguard: create transcoder: %w", err)
    }

    // 6. Configure h2c-enabled http.Server
    p := new(http.Protocols)
    p.SetHTTP1(true)
    p.SetUnencryptedHTTP2(true)

    s.httpServer = &http.Server{
        Addr:              fmt.Sprintf(":%d", s.config.Port),
        Handler:           transcoder,
        Protocols:         p,
        ReadTimeout:       s.config.ReadTimeout,
        WriteTimeout:      s.config.WriteTimeout,
        ReadHeaderTimeout: s.config.ReadHeaderTimeout,
        IdleTimeout:       s.config.IdleTimeout,
    }

    // 7. Start serving
    go func() {
        if err := s.httpServer.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
            s.logger.Error("vanguard server error", slog.Any("error", err))
        }
    }()

    return nil
}
```

### Pattern 3: gRPC Server Skip-Listener Mode

**What:** Modified gRPC server `OnStart`/`OnStop` that registers services but doesn't bind a port.
**When to use:** When Vanguard handles all incoming connections.

```go
// Source: Derived from existing server/grpc/server.go:107-162
func (s *Server) OnStart(ctx context.Context) error {
    if !s.config.SkipListener {
        // Existing behavior: bind port, serve, etc.
        // ... (unchanged)
    }

    // Skip-listener mode: still discover and register services
    registrars, err := di.ResolveAll[Registrar](s.container)
    if err != nil {
        return fmt.Errorf("grpc: discover services: %w", err)
    }
    for _, r := range registrars {
        r.RegisterService(s.server)
    }

    // Still register health
    if s.config.HealthEnabled {
        if manager, resolveErr := di.Resolve[*health.Manager](s.container); resolveErr == nil {
            s.healthAdapter = newHealthAdapter(manager, s.config.HealthCheckInterval, s.logger)
            s.healthAdapter.Register(s.server)
            s.healthAdapter.Start(ctx)
        }
    }

    // Still enable reflection
    if s.config.Reflection {
        reflection.Register(s.server)
    }

    s.logger.InfoContext(ctx, "gRPC server started (skip-listener mode)",
        slog.Bool("reflection", s.config.Reflection),
        slog.Int("services", len(registrars)),
    )

    return nil // No listener, no goroutine
}

func (s *Server) OnStop(ctx context.Context) error {
    if s.config.SkipListener {
        // No listener to close; just stop the server
        if s.healthAdapter != nil {
            _ = s.healthAdapter.Stop(ctx)
        }
        s.server.GracefulStop()
        return nil
    }
    // ... existing shutdown logic
}
```

### Pattern 4: Health Endpoint Mounting via Unknown Handler

**What:** Mount health HTTP endpoints as the Vanguard unknown handler.
**When to use:** When `health.Manager` is present in DI container.

```go
// Source: Mirrors health/server.go:30-33 and health/handlers.go
func (s *Server) buildUnknownHandler() http.Handler {
    mux := http.NewServeMux()

    // Auto-mount health endpoints if health.Manager is available
    if s.healthManager != nil {
        mux.Handle("/healthz", s.healthManager.NewReadinessHandler())
        mux.Handle("/readyz", s.healthManager.NewReadinessHandler())
        mux.Handle("/livez", s.healthManager.NewLivenessHandler())
    }

    // Allow user-defined routes
    if s.userUnknownHandler != nil {
        mux.Handle("/", s.userUnknownHandler)
    }

    return mux
}
```

### Pattern 5: Config Struct (follows existing convention exactly)

**What:** Configuration struct with flags, defaults, validation, and namespace.
**When to use:** Always — every server package has this pattern.

```go
// Source: Mirrors server/grpc/config.go and server/http/config.go
type Config struct {
    Port              int           `json:"port" yaml:"port" mapstructure:"port" gaz:"port"`
    ReadTimeout       time.Duration `json:"read_timeout" yaml:"read_timeout" mapstructure:"read_timeout" gaz:"read_timeout"`
    WriteTimeout      time.Duration `json:"write_timeout" yaml:"write_timeout" mapstructure:"write_timeout" gaz:"write_timeout"`
    IdleTimeout       time.Duration `json:"idle_timeout" yaml:"idle_timeout" mapstructure:"idle_timeout" gaz:"idle_timeout"`
    ReadHeaderTimeout time.Duration `json:"read_header_timeout" yaml:"read_header_timeout" mapstructure:"read_header_timeout" gaz:"read_header_timeout"`
    Reflection        bool          `json:"reflection" yaml:"reflection" mapstructure:"reflection" gaz:"reflection"`
    DevMode           bool          `json:"dev_mode" yaml:"dev_mode" mapstructure:"dev_mode" gaz:"dev_mode"`
}

func (c *Config) Namespace() string { return "server" }

func (c *Config) Flags(fs *pflag.FlagSet) {
    fs.IntVar(&c.Port, "server-port", c.Port, "Server port")
    fs.DurationVar(&c.ReadHeaderTimeout, "server-read-header-timeout", c.ReadHeaderTimeout, "Read header timeout")
    fs.DurationVar(&c.IdleTimeout, "server-idle-timeout", c.IdleTimeout, "Idle timeout")
    fs.BoolVar(&c.Reflection, "server-reflection", c.Reflection, "Enable gRPC/Connect reflection")
    fs.BoolVar(&c.DevMode, "server-dev-mode", c.DevMode, "Enable development mode")
}
```

**Note on timeout validation:** Unlike `server/http/Config.Validate()` which requires positive timeouts, the Vanguard config must accept `ReadTimeout=0` and `WriteTimeout=0` as valid (streaming-safe). Only `ReadHeaderTimeout` and `IdleTimeout` need positive validation.

### Pattern 6: Module Registration (follows existing convention)

**What:** `NewModule()` function using `gaz.NewModule` builder.
**When to use:** Always — standard DI module pattern.

```go
// Source: Mirrors server/grpc/module.go:175-188
func NewModule() gaz.Module {
    defaultCfg := DefaultConfig()
    return gaz.NewModule("vanguard").
        Flags(defaultCfg.Flags).
        Provide(provideConfig(defaultCfg)).
        Provide(provideServer).
        Build()
}
```

### Anti-Patterns to Avoid

- **Reusing `server/http` package for Vanguard:** The HTTP server package has `ReadTimeout`/`WriteTimeout` defaults that break streaming. Vanguard server needs its own `http.Server` with streaming-safe zero timeouts. Decision explicitly says "does NOT reuse `server/http` package."
- **Building transcoder in provider function:** Vanguard transcoder is one-shot. If built in a provider, DI resolution order may not guarantee all services are registered. Build in `OnStart` after `di.ResolveAll`.
- **Adding routes to the Vanguard transcoder:** Vanguard transcodes RPC protocols. Non-RPC routes (health, metrics) go through the unknown handler fallback, not as Vanguard services.
- **Using `ReadTimeout`/`WriteTimeout` for DoS protection:** These apply per-connection and kill streaming RPCs. Use `ReadHeaderTimeout` for slowloris protection instead.
- **Registering Connect reflection handlers conditionally:** Both `grpcreflect.NewHandlerV1` and `grpcreflect.NewHandlerV1Alpha` should be registered — many tools (including `grpcurl`) still use v1alpha.

## Don't Hand-Roll

| Problem | Don't Build | Use Instead | Why |
|---------|-------------|-------------|-----|
| Protocol multiplexing | Custom Content-Type router | `vanguard.NewTranscoder` | Handles gRPC, Connect, gRPC-Web, REST protocol detection and transcoding correctly |
| REST transcoding | Custom HTTP-to-gRPC mapping | Vanguard + proto `google.api.http` annotations | Codegen-free, handles path params, query params, body mapping, FieldMask |
| gRPC-Web support | Custom frame encoding | Vanguard built-in | gRPC-Web framing is complex (base64, binary modes) |
| h2c (HTTP/2 cleartext) | `golang.org/x/net/http2/h2c` wrapper | `http.Protocols.SetUnencryptedHTTP2(true)` | Go 1.26 native; cleaner, no external dependency |
| gRPC reflection for Connect | Custom reflection handler | `connectrpc.com/grpcreflect` | Handles both v1 and v1alpha reflection APIs |
| Health check HTTP handlers | Custom health endpoint handlers | `health.Manager.NewReadinessHandler()` etc. | Already implemented with IETF `application/health+json` format |

**Key insight:** The entire value of Phase 46 is composition — wiring existing components (gRPC server, health manager, Connect handlers) through Vanguard's transcoder. Almost nothing should be built from scratch.

## Common Pitfalls

### Pitfall 1: Vanguard Transcoder is Immutable After Construction
**What goes wrong:** Attempting to add services to the transcoder after `NewTranscoder()` returns.
**Why it happens:** Unlike `grpc.Server` which accepts `RegisterService` calls, Vanguard takes all services at construction time.
**How to avoid:** Collect ALL services (Connect registrars + gRPC transcoder) before calling `vanguard.NewTranscoder()`. Do this in `OnStart`, not in the provider.
**Warning signs:** Services not appearing in Vanguard despite being registered in DI.

### Pitfall 2: Server Timeouts Kill Streaming RPCs
**What goes wrong:** `http.Server.ReadTimeout` or `WriteTimeout` set to non-zero values terminate long-running streaming RPCs.
**Why it happens:** These timeouts apply per-connection, not per-request. A server-streaming RPC that sends data over 30 seconds gets killed.
**How to avoid:** Set `ReadTimeout=0` and `WriteTimeout=0`. Use `ReadHeaderTimeout=5s` for slowloris protection. Handle per-request timeouts via context deadlines.
**Warning signs:** Streaming RPCs working in tests but failing in production with timeout errors.

### Pitfall 3: gRPC Skip-Listener Mode Forgets GracefulStop
**What goes wrong:** gRPC server in skip-listener mode doesn't call `GracefulStop()` on shutdown, leaving in-flight RPCs in undefined state.
**Why it happens:** The existing `OnStop` uses `GracefulStop()` in a goroutine with listener close — but skip-listener mode has no listener.
**How to avoid:** In skip-listener `OnStop`, call `s.server.GracefulStop()` directly (synchronous, no listener to close).
**Warning signs:** gRPC health adapter or interceptors not cleaning up on shutdown.

### Pitfall 4: Missing Reflection v1alpha Handler
**What goes wrong:** Tools like `grpcurl` can't discover services because they use the older v1alpha reflection API.
**Why it happens:** Only registering `grpcreflect.NewHandlerV1` without `NewHandlerV1Alpha`.
**How to avoid:** Register BOTH v1 and v1alpha reflection handlers as ConnectRegistrars.
**Warning signs:** `grpcurl` fails to list services even though reflection is enabled.

### Pitfall 5: Health Endpoints Conflicting with RPC Routes
**What goes wrong:** Health check paths like `/healthz` might shadow RPC routes or vice versa.
**Why it happens:** If health endpoints are registered as Vanguard services instead of via the unknown handler.
**How to avoid:** Health endpoints go through `vanguard.WithUnknownHandler()` — they are NOT Vanguard services. Vanguard checks its service routes first, then falls through to the unknown handler.
**Warning signs:** Health checks returning gRPC errors or proto responses instead of JSON.

### Pitfall 6: Config Validation Rejects Zero Timeouts
**What goes wrong:** Validation copied from `server/http/Config.Validate()` rejects `ReadTimeout=0` and `WriteTimeout=0`.
**Why it happens:** The HTTP config requires all timeouts to be positive — but Vanguard intentionally uses zero for streaming safety.
**How to avoid:** Vanguard config validation only requires `ReadHeaderTimeout > 0` and `IdleTimeout > 0`. `ReadTimeout` and `WriteTimeout` should accept zero values.
**Warning signs:** Server fails to start with "invalid read_timeout" errors.

### Pitfall 7: DI Resolution Order Between gRPC and Vanguard Servers
**What goes wrong:** Vanguard server resolves `*grpc.Server` before the gRPC module has registered it.
**Why it happens:** Module registration order affects DI resolution order.
**How to avoid:** Vanguard server resolves `*grpc.Server` from the gRPC `Server` wrapper (via `gaz.Resolve[*grpc.Server]`), which is already registered by the gRPC module. The gRPC module should be a dependency (`Use`) of the Vanguard module. Both are Eager, so DI resolution order matters — gRPC module must come first.
**Warning signs:** "service not found" errors for `*grpc.Server` at startup.

## Code Examples

Verified patterns from official sources and existing codebase:

### Connect-Go Handler Registration (from official docs)
```go
// Source: Connect-Go official example and Vanguard README
// Generated code produces: func NewMyServiceHandler(impl) (string, http.Handler)

// The ConnectRegistrar wraps this pattern:
type GreeterService struct {
    greetv1connect.UnimplementedGreeterServiceHandler
}

func (s *GreeterService) Greet(ctx context.Context, req *connect.Request[greetv1.GreetRequest]) (*connect.Response[greetv1.GreetResponse], error) {
    return connect.NewResponse(&greetv1.GreetResponse{
        Greeting: "Hello, " + req.Msg.GetName(),
    }), nil
}

// RegisterConnect implements connect.ConnectRegistrar
func (s *GreeterService) RegisterConnect() (string, http.Handler) {
    return greetv1connect.NewGreeterServiceHandler(s)
}
```

### Vanguard Service Composition (from Vanguard README)
```go
// Source: connectrpc.com/vanguard README.md
// Connect service → vanguard.NewService (path, handler directly)
myService := vanguard.NewService(
    myservicev1connect.NewMyServiceHandler(&myServiceImpl{}),
)

// gRPC server → vanguardgrpc.NewTranscoder (returns []*vanguard.Service)
grpcServices := vanguardgrpc.NewTranscoder(grpcServer)

// Compose all services into one transcoder
transcoder, err := vanguard.NewTranscoder(
    append(grpcServices, myService),
    vanguard.WithUnknownHandler(healthMux),
)
// transcoder implements http.Handler
```

### h2c Configuration (from Connect-Go official example)
```go
// Source: Connect-Go official example (verified in Context7)
p := new(http.Protocols)
p.SetHTTP1(true)
p.SetUnencryptedHTTP2(true)

server := &http.Server{
    Addr:              ":8080",
    Handler:           transcoder,
    Protocols:         p,
    ReadHeaderTimeout: 5 * time.Second,
    IdleTimeout:       120 * time.Second,
    // ReadTimeout and WriteTimeout intentionally 0 for streaming
}
```

### gRPC Reflection via Connect (from connectrpc.com docs)
```go
// Source: connectrpc.com/grpcreflect documentation
// Must mount BOTH v1 and v1alpha for grpcurl compatibility

reflector := grpcreflect.NewStaticReflector(serviceNames...)

// Each returns (string, http.Handler) — matches ConnectRegistrar signature
v1Path, v1Handler := grpcreflect.NewHandlerV1(reflector)
v1AlphaPath, v1AlphaHandler := grpcreflect.NewHandlerV1Alpha(reflector)
```

### Existing gRPC Module Pattern (template for Vanguard module)
```go
// Source: server/grpc/module.go:175-188
func NewModule() gaz.Module {
    defaultCfg := DefaultConfig()
    return gaz.NewModule("grpc").
        Flags(defaultCfg.Flags).
        Provide(provideConfig(defaultCfg)).
        Provide(provideLoggingBundle).
        Provide(provideServer).
        Build()
}
```

### Test Pattern (template for Vanguard tests)
```go
// Source: server/grpc/server_test.go (test suite pattern)
type VanguardServerTestSuite struct {
    suite.Suite
}

func TestVanguardServerTestSuite(t *testing.T) {
    suite.Run(t, new(VanguardServerTestSuite))
}

func (s *VanguardServerTestSuite) TestStartStop() {
    cfg := DefaultConfig()
    cfg.Port = getFreePort(s.T())
    // ... setup container with mock registrars
    // ... test OnStart/OnStop lifecycle
}
```

## State of the Art

| Old Approach | Current Approach | When Changed | Impact |
|--------------|------------------|--------------|--------|
| `golang.org/x/net/http2/h2c` wrapper | `http.Protocols.SetUnencryptedHTTP2(true)` | Go 1.25 (stdlib) | Eliminates external dependency for h2c |
| grpc-gateway (codegen + loopback) | Vanguard (transcoding, no codegen) | Vanguard v0.1.0 (2024) | No generated gateway code, no loopback connection, single port |
| Multi-port serving (gRPC :50051, HTTP :8080) | Single-port via Vanguard | Vanguard + h2c | Simpler deployment, single load balancer target |
| `grpc.Dial` (deprecated) | `grpc.NewClient` | grpc-go v1.65 | Non-blocking client creation |
| Native gRPC reflection | `connectrpc.com/grpcreflect` Connect handler | Connect ecosystem | Works through Vanguard transcoder; gRPC native reflection only works on native gRPC server |

**Deprecated/outdated:**
- `grpc.Dial()`: Deprecated in favor of `grpc.NewClient()`. Already fixed in existing codebase.
- `golang.org/x/net/http2/h2c.NewHandler()`: Replaced by Go 1.26 native `http.Protocols`. Do not use.
- grpc-gateway codegen: Replaced by Vanguard transcoding. Gateway package preserved in Phase 46 but deprecated in Phase 48.

## Open Questions

1. **`vanguardgrpc.NewTranscoder` Return Type**
   - What we know: Returns services that can be passed to `vanguard.NewTranscoder`. The README shows `grpcServices := vanguardgrpc.NewTranscoder(grpcServer)`.
   - What's unclear: Exact return type — `[]*vanguard.Service` or `*vanguard.Service` (single). Need to verify.
   - Recommendation: Check the actual Go package API at build time. If it returns a single `*vanguard.Service`, wrap in slice. LOW risk — trivial to adapt.

2. **Vanguard Service Names for Reflection**
   - What we know: `grpcreflect.NewStaticReflector(serviceNames...)` needs the list of service names.
   - What's unclear: Whether service names should include gRPC services bridged via `vanguardgrpc`, only Connect services, or both.
   - Recommendation: Pass ALL service names (both gRPC and Connect) to the reflector. The reflector should reflect everything the server serves. Can get gRPC service names from `grpc.Server.GetServiceInfo()`.

3. **Vanguard-Specific Config Options**
   - What we know: Vanguard supports `WithCodec`, `WithTargetProtocols`, `WithTargetCodecs`.
   - What's unclear: Whether these should be user-configurable or hardcoded.
   - Recommendation: Keep internal for Phase 46 (Claude's Discretion area). Expose only if a concrete use case emerges.

## Sources

### Primary (HIGH confidence)
- Context7 `/connectrpc/vanguard-go` — transcoder API, service registration, unknown handler option, gRPC bridging
- Context7 `/connectrpc/connect-go` — handler registration pattern `(string, http.Handler)`, interceptors, h2c setup
- Context7 `/websites/connectrpc` — grpcreflect setup (v1 + v1alpha handlers), gRPC compatibility docs
- Context7 `/golang/go/go1.26.0` — `http.Protocols` API, server lifecycle, shutdown
- Existing codebase `server/grpc/server.go` — Registrar interface, OnStart/OnStop lifecycle, DI resolution pattern
- Existing codebase `server/grpc/config.go` — Config struct with Flags/SetDefaults/Validate/Namespace
- Existing codebase `server/grpc/module.go` — Module builder pattern with Flags/Provide/Build
- Existing codebase `server/grpc/health_adapter.go` — Health integration pattern
- Existing codebase `server/http/server.go` — HTTP server lifecycle (template for Vanguard, but NOT reused)
- Existing codebase `health/handlers.go` — `NewReadinessHandler()`, `NewLivenessHandler()` returning `http.Handler`

### Secondary (MEDIUM confidence)
- Vanguard README.md — `NewTranscoder` API, `vanguardgrpc.NewTranscoder`, `WithUnknownHandler`
- Connect-Go official deployment docs — h2c configuration, grpcreflect handler setup
- `.planning/research/SUMMARY.md` — Stack selection, architecture decisions, pitfall catalog

### Tertiary (LOW confidence)
- h2c behavior with non-Go gRPC clients — needs empirical validation in integration tests
- `vanguardgrpc.NewTranscoder` exact return type — verified from README examples but not from Go API docs

## Metadata

**Confidence breakdown:**
- Standard stack: HIGH — all packages verified on Context7 with exact versions; Connect-Go v1.x stable
- Architecture: HIGH — patterns directly derived from existing codebase (`server/grpc/`, `server/http/`) with verified Vanguard API
- Pitfalls: HIGH — pitfalls sourced from official docs, existing code analysis, and v5.0 research summary
- Code examples: HIGH — all examples verified against Context7 and existing codebase patterns

**Research date:** 2026-03-06
**Valid until:** 2026-04-06 (30 days — Vanguard API stable within minor version)
