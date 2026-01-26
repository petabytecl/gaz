# Technology Stack

**Project:** gaz - Go Application Framework
**Researched:** 2026-01-26
**Mode:** Ecosystem Research - Stack Dimension

---

## Executive Summary

This research establishes the recommended 2026 Go stack for building a unified application framework consolidating DI, lifecycle management, config, and logging. The Go ecosystem has matured significantly with:

- **stdlib slog (Go 1.21+)** now the standard for structured logging
- **koanf** emerging as the cleaner alternative to viper for config
- **samber/do v2** representing the latest generics-based DI approach
- **cobra** remaining the undisputed CLI standard

For a framework that values convention-over-configuration and clean APIs, the recommendation is to use stdlib-first where possible (slog), leverage modern generics-based patterns (custom DI inspired by do/fx), and integrate with battle-tested ecosystem tools (cobra, koanf).

---

## Recommended Stack

### Core Runtime

| Technology | Version | Purpose | Confidence | Why |
|------------|---------|---------|------------|-----|
| Go | 1.21+ | Runtime | HIGH | Required for slog, mature generics |
| log/slog | stdlib | Structured logging | HIGH | Stdlib, zero dependencies, widely adopted |

**Go 1.21+ Rationale:** This version introduced `log/slog` in stdlib and has mature generics support. The framework should require 1.21+ minimum, with 1.22+ recommended for enhanced loop semantics.

### CLI Framework

| Technology | Version | Purpose | Confidence | Why |
|------------|---------|---------|------------|-----|
| spf13/cobra | v1.9.1 | CLI framework | HIGH | Industry standard, excellent ecosystem |
| spf13/pflag | v1.0.5 | POSIX-compliant flags | HIGH | Required by cobra |

**Cobra Rationale (Context7 verified):**
- Undisputed standard for Go CLIs (kubectl, docker, hugo all use it)
- Excellent subcommand support
- Built-in help generation, shell completions
- Integrates seamlessly with viper/koanf for config
- Persistent and local flag scoping

### Configuration

| Technology | Version | Purpose | Confidence | Why |
|------------|---------|---------|------------|-----|
| knadh/koanf | v2.x | Configuration loading | HIGH | Cleaner than viper, better abstraction |
| knadh/koanf/parsers/yaml | v2.x | YAML parsing | HIGH | Common config format |
| knadh/koanf/parsers/json | v2.x | JSON parsing | HIGH | API/config interop |
| knadh/koanf/providers/env | v2.x | Environment variables | HIGH | 12-factor compliance |
| knadh/koanf/providers/file | v2.x | File loading | HIGH | Config file support |
| knadh/koanf/providers/posflag | v2.x | pflag integration | HIGH | CLI flag binding |

**Koanf over Viper Rationale (Context7 verified):**
- **Cleaner API:** No global state, explicit providers/parsers
- **Better abstractions:** Separates provider (where data comes from) from parser (how to decode)
- **Lighter weight:** Fewer dependencies, faster startup
- **Modern design:** Built for composition, not magic
- **Full feature parity:** env vars, files, CLI flags, remote config

```go
// Koanf's explicit, composable design
k := koanf.New(".")
k.Load(file.Provider("config.yaml"), yaml.Parser())
k.Load(env.Provider("APP_", ".", transformKey), nil)
k.Load(posflag.Provider(flags, ".", k), nil)
```

### Dependency Injection

| Technology | Version | Purpose | Confidence | Why |
|------------|---------|---------|------------|-----|
| Custom (inspired by samber/do + fx) | n/a | DI container | MEDIUM | Framework-specific needs |

**DI Landscape Analysis:**

| Library | Approach | Strengths | Weaknesses | Fit for gaz |
|---------|----------|-----------|------------|-------------|
| **samber/do v2** | Runtime, generics | Clean API, health checks, shutdown, scopes | Young v2, some complexity | Good inspiration |
| **uber-go/fx** | Runtime, reflection | Battle-tested at scale, lifecycle hooks | Heavy, Uber-specific patterns | Good patterns |
| **uber-go/dig** | Runtime, reflection | Simple, flexible | Lower-level than fx | Underlying engine |
| **google/wire** | Compile-time codegen | Zero runtime overhead | No runtime flexibility, code generation | Not recommended |

**Recommendation: Custom DI inspired by do/fx**

Given the reference implementations in `tmp/dibx/` and `tmp/gazx/`, the framework already has a custom DI solution. The recommendation is to:

1. Keep the custom DI approach for full control
2. Study samber/do v2 for API patterns (generics-based registration, health checks)
3. Study fx for lifecycle patterns (OnStart/OnStop hooks, graceful shutdown)
4. Avoid wire's codegen approach (limits flexibility, adds build complexity)

**Key features to ensure (from samber/do v2):**
- Generics-based registration: `Register[T](provider)`
- Health check interface integration
- Graceful shutdown with context
- Service introspection for debugging

### Logging

| Technology | Version | Purpose | Confidence | Why |
|------------|---------|---------|------------|-----|
| log/slog | stdlib (Go 1.21+) | Structured logging API | HIGH | Stdlib standard, zero deps |
| (optional) slog handler adapters | varies | Backend flexibility | MEDIUM | For zerolog/zap backends if needed |

**slog as Primary Rationale (Official Go documentation verified):**

Since Go 1.21, `log/slog` is the standard structured logging package. Key advantages:

- **Stdlib:** Zero dependencies, guaranteed long-term support
- **Unified API:** Handler interface allows plugging any backend
- **Performance:** Designed for efficiency with zero-allocation patterns
- **Ecosystem adoption:** 59,515+ packages import slog (pkg.go.dev)

```go
// slog's clean, context-aware API
slog.InfoContext(ctx, "request complete",
    slog.String("method", r.Method),
    slog.Int("status", 200),
    slog.Duration("latency", elapsed),
)
```

**When to consider zerolog/zap handlers:**
- Ultra-high-throughput logging (millions of logs/sec)
- Need specific features (diode writer for non-blocking)
- Legacy integration requirements

### Health Checks

| Pattern | Source | Purpose | Confidence | Why |
|---------|--------|---------|------------|-----|
| Readiness/Liveness pattern | Kubernetes | Health check model | HIGH | Industry standard |
| Interface-based checks | samber/do | Check registration | HIGH | Composable pattern |

**Health Check Patterns (from samber/do and gazx):**

```go
// Interface-based health checks (recommended pattern)
type HealthChecker interface {
    HealthCheck(ctx context.Context) error
}

// Typed check registration
type HealthCheckerType int
const (
    LivenessHealthType HealthCheckerType = iota
    ReadinessHealthType
)
```

The existing `gazx` implementation already follows best practices:
- Separate liveness (is process alive?) and readiness (can accept traffic?)
- Context-aware with timeouts
- Concurrent check execution
- Health status events for observability

### Supporting Libraries

| Library | Version | Purpose | When to Use |
|---------|---------|---------|-------------|
| uber-go/automaxprocs | latest | GOMAXPROCS tuning | Containerized deployments |
| (optional) uber-go/memlimit | latest | Memory limit awareness | Container memory limits |
| errors (stdlib) | stdlib | Error handling | Always |
| context (stdlib) | stdlib | Cancellation/timeouts | Always |

### Dev Dependencies

| Tool | Version | Purpose |
|------|---------|---------|
| golangci-lint | v1.x | Linting |
| testify | v1.x | Test assertions |
| mockery | v2.x | Mock generation |

---

## Alternatives Considered

### DI Libraries

| Recommended | Alternative | Why Not Alternative |
|-------------|-------------|---------------------|
| Custom DI | uber-go/fx | Too opinionated, heavy for framework embedding |
| Custom DI | google/wire | Compile-time codegen limits runtime flexibility |
| Custom DI | samber/do | Good patterns but v2 still young, want full control |

### Config Libraries

| Recommended | Alternative | Why Not Alternative |
|-------------|-------------|---------------------|
| knadh/koanf | spf13/viper | Global state, heavier, more magic |
| knadh/koanf | kelseyhightower/envconfig | Too simple, env-only |
| knadh/koanf | caarlos0/env | Struct tags only, limited layering |

### Logging Libraries

| Recommended | Alternative | Why Not Alternative |
|-------------|-------------|---------------------|
| log/slog (stdlib) | rs/zerolog | External dep, slog is now stdlib |
| log/slog (stdlib) | uber-go/zap | External dep, slog is now stdlib |

**Note:** zerolog and zap remain excellent choices and can be used as slog backends via handler adapters when maximum performance is critical.

### CLI Libraries

| Recommended | Alternative | Why Not Alternative |
|-------------|-------------|---------------------|
| spf13/cobra | urfave/cli | Less popular, fewer features |
| spf13/cobra | alecthomas/kong | Smaller ecosystem |

---

## What NOT to Use

| Technology | Why Avoid |
|------------|-----------|
| **google/wire for DI** | Code generation adds build complexity, limits runtime flexibility for testing/overrides |
| **spf13/viper** | Global state, heavy, koanf is cleaner modern alternative |
| **logrus** | Deprecated in favor of slog, no longer maintained actively |
| **global singletons** | Makes testing difficult, prefer explicit injection |
| **init() for registration** | Implicit ordering issues, prefer explicit registration |

---

## Installation

```bash
# Core dependencies
go get github.com/spf13/cobra@v1.9.1
go get github.com/spf13/pflag@v1.0.5
go get github.com/knadh/koanf/v2@latest
go get github.com/knadh/koanf/parsers/yaml@latest
go get github.com/knadh/koanf/parsers/json@latest
go get github.com/knadh/koanf/providers/file@latest
go get github.com/knadh/koanf/providers/env/v2@latest
go get github.com/knadh/koanf/providers/posflag@latest

# Runtime optimization (optional)
go get go.uber.org/automaxprocs@latest

# Dev dependencies
go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
```

---

## Version Compatibility Matrix

| Component | Minimum Go | Tested With | Notes |
|-----------|------------|-------------|-------|
| gaz framework | Go 1.21 | Go 1.22, 1.23 | slog requires 1.21 |
| cobra v1.9.1 | Go 1.15 | Go 1.22+ | Very compatible |
| koanf v2.x | Go 1.18 | Go 1.22+ | Generics for providers |
| slog | Go 1.21 | Go 1.22+ | Stdlib, no version issues |

---

## Sources

### HIGH Confidence (Context7 / Official Docs)

- **cobra v1.9.1:** Context7 `/spf13/cobra/v1.9.1` - Persistent flags, viper integration
- **koanf:** Context7 `/knadh/koanf` - Providers, parsers, pflag integration
- **uber-go/fx v1.24.0:** Context7 `/uber-go/fx` - Lifecycle hooks, dependency injection patterns
- **samber/do v2.0.0:** Context7 `/samber/do/v2_0_0` - Health checks, shutdown, scopes
- **log/slog:** Official Go documentation pkg.go.dev - Handlers, levels, groups
- **zerolog:** Context7 `/rs/zerolog` - Performance benchmarks, structured logging
- **zap:** Context7 `/uber-go/zap` - Performance, sugared/core logger patterns

### MEDIUM Confidence (GitHub Releases, Verified)

- **samber/do v2.0.0 release:** https://github.com/samber/do/releases/tag/v2.0.0 (2024-09-21)
- **uber-go/fx v1.24.0 release:** https://github.com/uber-go/fx/releases/tag/v1.24.0 (2025-05-13)
- **cobra v1.9.1 release:** Context7 verified

### Reference Implementations Analyzed

- `/home/coto/dev/petabyte/gaz/tmp/dibx/` - Custom DI container with generics, service types
- `/home/coto/dev/petabyte/gaz/tmp/gazx/` - App framework with lifecycle, health checks, events

---

## Implications for Roadmap

Based on this stack research:

1. **Phase 1 (Core DI):** Refine custom DI based on dibx patterns + do/fx learnings
2. **Phase 2 (Lifecycle):** Implement Start/Stop hooks following fx patterns
3. **Phase 3 (Config):** Integrate koanf with clear provider ordering
4. **Phase 4 (Logging):** slog integration with handler flexibility
5. **Phase 5 (Health):** Readiness/liveness pattern from gazx
6. **Phase 6 (CLI):** Cobra integration with config binding

**Key architectural decisions this research supports:**
- Stdlib-first (slog) reduces dependencies
- koanf's explicit providers enable predictable config layering
- Custom DI gives full control over API and behavior
- Cobra is non-negotiable for CLI applications
