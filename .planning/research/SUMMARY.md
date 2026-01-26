# Project Research Summary

**Project:** gaz - Unified Go Application Framework
**Domain:** Go Application Framework (DI, Lifecycle, Config, Logging)
**Researched:** 2026-01-26
**Confidence:** HIGH

## Executive Summary

Building a unified Go application framework requires consolidating several proven patterns: generics-based dependency injection for compile-time type safety, hook-based lifecycle management for predictable startup/shutdown, and stdlib-first choices (slog for logging) with ecosystem integration (Cobra for CLI, Koanf for config). The Go ecosystem has matured significantly since 2021 with `log/slog` becoming the standard, `koanf` emerging as the cleaner viper alternative, and generics enabling type-safe DI without code generation.

The recommended approach is a **layered architecture with a core package and optional subpackages**. The DI container forms the foundation, followed by lifecycle management, then the App builder with Cobra integration. Health checks, config, and logging are optional subpackages that enhance but don't require the core. This design maximizes flexibility while maintaining a clean API surface. Convention-over-configuration is key: sensible defaults that work immediately, with explicit overrides when needed.

The primary risks are **over-abstraction** (hiding too much, making debugging impossible), **scope complexity** (too many scope types confusing users), and **non-deterministic lifecycle hooks** (unpredictable startup/shutdown order causing production bugs). All three are mitigated by following Go idioms: explicit > magic, minimal API surface, LIFO shutdown order. The custom DI approach (inspired by samber/do + fx patterns) provides full control over API design while avoiding the code generation overhead of Wire.

## Key Findings

### Recommended Stack

The stack prioritizes stdlib-first choices with mature ecosystem dependencies. Go 1.21+ is required for `log/slog` and mature generics support.

**Core technologies:**
- **Go 1.21+**: Runtime requirement for slog, mature generics (HIGH confidence)
- **log/slog (stdlib)**: Structured logging - stdlib standard, zero dependencies, handler-based extensibility
- **Cobra v1.9.1**: CLI framework - undisputed standard (kubectl, docker, hugo use it), excellent subcommand support
- **Koanf v2.x**: Configuration - cleaner than Viper, explicit providers/parsers, no global state
- **Custom DI (samber/do + fx patterns)**: Full control, generics-based, health check integration

**Explicitly avoided:**
- google/wire: Code generation adds friction, limits runtime flexibility
- spf13/viper: Global state, heavier, koanf is modern alternative
- logrus: Deprecated, slog is now stdlib standard

### Expected Features

**Must have (table stakes):**
- Dependency injection with constructor registration
- Lazy instantiation (performance, don't build what isn't used)
- Singleton scoping (most services should be singletons)
- Error propagation from constructors `(T, error)` pattern
- Lifecycle hooks (OnStart/OnStop)
- Graceful shutdown with signal handling (SIGTERM/SIGINT)
- Shutdown timeouts
- Context propagation for cancellation
- Module/provider system for code organization
- Testing support (test builders, isolated containers)
- Logging integration with slog

**Should have (differentiators):**
- Type-safe generics DI (compile-time safety, no `interface{}` casts)
- Convention-over-configuration (zero-config works, override when needed)
- Cobra CLI integration in App builder
- Health check subsystem (liveness/readiness for Kubernetes)
- Deterministic startup/shutdown order (LIFO for stop)
- Named services (multiple implementations of same type)
- Service override for testing
- Struct tag injection (`gaz:"name:primary"`)

**Defer (v2+):**
- Hierarchical scopes (complex, niche use case)
- Event bus (nice-to-have, not essential)
- Worker groups (can use goroutines directly)
- Decorator pattern (advanced)
- Value groups (fx feature, advanced)

### Architecture Approach

A 4-layer architecture with optional subpackages: Container (DI) forms Layer 1 providing type-safe dependency injection with generics. Lifecycle Manager (Layer 2) orchestrates startup/shutdown with hook-based callbacks. App Builder (Layer 3) provides the fluent user-facing API with Cobra integration. Provider System (Layer 4) encapsulates related services, flags, and lifecycle components. Optional subpackages (health/, config/, log/) depend on core but core works standalone.

**Major components:**
1. **Container** — Type-safe DI with generics, service registry, lazy/eager/transient resolution
2. **Lifecycle Manager** — OnStart/OnStop hooks, ordered execution, shutdown timeout
3. **App Builder** — Fluent API (`gaz.New(cmd).With(...).Run()`), signal handling
4. **Provider System** — Module composition, dependency declaration, flags integration
5. **health/** — Health checks, liveness/readiness probes (optional)
6. **config/** — Koanf integration, Cobra flag binding (optional)
7. **log/** — slog setup, structured logging handlers (optional)

### Critical Pitfalls

Research identified 15+ pitfalls. Top 5 to actively prevent:

1. **Over-abstraction / "Magic"** — Keep behavior explicit and traceable. If users can't understand what happens by reading main.go, the API has gone too far. Error messages must include full dependency chain context.

2. **Reflection performance in hot paths** — Use generics for type-safe paths, cache reflection at registration time. Target: <100 ns/op, 0 allocs/op for cached singleton resolution. Never reflect on every resolution.

3. **Scope complexity explosion** — Only provide: Singleton (default), Transient (new each request), and explicit child scopes. No PerRequest, PerGraph, Custom scopes. Users should never need to ask "which scope should I use?"

4. **Lifecycle hook ordering confusion** — Document and enforce: Stop hooks run in REVERSE of Start. If A depends on B: Start order is B→A, Stop order is A→B. Timeout ALL lifecycle operations.

5. **Global container singleton** — Never provide package-level container. Pass injector explicitly. Tests must be able to run in parallel with isolated containers.

## Implications for Roadmap

Based on research, suggested phase structure:

### Phase 1: Core DI Container
**Rationale:** Foundation layer; everything else depends on it. STACK.md confirms custom DI inspired by samber/do + fx patterns is the approach.
**Delivers:** Type-safe DI with generics, registration API, resolution, singleton/transient scoping
**Addresses:** DI, Constructor Registration, Lazy Instantiation, Singleton Scoping, Error Handling, Named Services
**Avoids:** Global singleton, stringly-typed deps, reflection performance trap, allocation per resolve

### Phase 2: Lifecycle Management
**Rationale:** Must exist before App builder; hooks are registered during provider calls. ARCHITECTURE.md places this as Layer 2.
**Delivers:** OnStart/OnStop hooks, ordered execution, graceful shutdown, shutdown timeout
**Addresses:** Lifecycle Hooks, Graceful Shutdown, Shutdown Timeouts, Context Propagation
**Avoids:** Hook ordering confusion, cleanup/shutdown bugs, non-LIFO stop order

### Phase 3: App Builder + Cobra Integration
**Rationale:** User-facing API that combines container + lifecycle. Cobra is non-negotiable per STACK.md.
**Delivers:** Fluent builder API, signal handling, Cobra command integration, module/provider composition
**Addresses:** Module System, Testing Support, Convention-over-Configuration
**Avoids:** Too many ways to do same thing, God provider anti-pattern

### Phase 4: Config System
**Rationale:** Independent of lifecycle but needs container for registration. Koanf integration per STACK.md.
**Delivers:** Multi-source config (env, files, flags), type-safe config structs, Cobra flag binding
**Addresses:** Multi-Source Config (FEATURES.md differentiator)
**Avoids:** Over-configuration, magic string keys

### Phase 5: Health Checks
**Rationale:** Production-ready feature that requires lifecycle (starts/stops with app). FEATURES.md marks as differentiator.
**Delivers:** HealthManager, liveness/readiness probes, K8s-native patterns, concurrent check execution
**Addresses:** Health Check Subsystem (differentiator from fx)
**Avoids:** No health check integration debt

### Phase 6: Logging (slog Integration)
**Rationale:** Can be done in parallel with Phase 5. slog is stdlib per STACK.md.
**Delivers:** Default logger setup, context propagation, framework event logging, handler flexibility
**Addresses:** Logging Integration, slog Context Propagation
**Avoids:** Silent failures (structured error logging)

### Phase Ordering Rationale

- **Phases 1-2-3 are sequential:** Container → Lifecycle → App Builder follows natural dependency order from ARCHITECTURE.md
- **Phases 4-5-6 can parallelize:** Config, Health, and Logging are optional subpackages with no inter-dependencies
- **Core before optional:** Phases 1-3 establish the foundation; 4-6 enhance it
- **This order prevents:** Building on unstable foundation, lifecycle bugs bleeding into all components, API churn from late DI changes

### Research Flags

Phases likely needing deeper research during planning:
- **Phase 1 (Core DI):** Performance benchmarking needed — must validate <100ns resolution before moving on
- **Phase 4 (Config):** Koanf API patterns — verify provider ordering and pflag integration work as documented

Phases with standard patterns (skip research-phase):
- **Phase 2 (Lifecycle):** Well-documented fx.Lifecycle patterns, gazx reference implementation exists
- **Phase 3 (App Builder):** Fluent builder is straightforward, Cobra integration is standard
- **Phase 5 (Health):** K8s liveness/readiness is industry standard, samber/do provides patterns
- **Phase 6 (Logging):** slog is stdlib with official documentation, handler pattern is clear

## Confidence Assessment

| Area | Confidence | Notes |
|------|------------|-------|
| Stack | HIGH | Context7 verified: Cobra, Koanf, fx, do, slog official docs |
| Features | HIGH | Cross-referenced fx/wire/go-micro + internal dibx/gazx |
| Architecture | HIGH | Verified patterns from fx + local reference implementations |
| Pitfalls | HIGH | Context7 + official docs + internal experience with dibx/gazx |

**Overall confidence:** HIGH

### Gaps to Address

- **Custom DI performance:** Must benchmark early in Phase 1 — no verified performance data for generics approach yet
- **Koanf + Cobra integration:** Need to validate pflag provider works as expected (medium confidence from docs)
- **samber/do v2 patterns:** v2 is relatively young (released 2024-09-21) — verify patterns in implementation
- **Hierarchical scopes (deferred):** If needed later, design work required — current recommendation is flat scope only

## Sources

### Primary (HIGH confidence)
- Context7 `/spf13/cobra/v1.9.1` — CLI patterns, persistent flags, viper integration
- Context7 `/knadh/koanf` — Providers, parsers, pflag integration
- Context7 `/uber-go/fx` — Lifecycle hooks, modules, dependency injection patterns
- Context7 `/samber/do/v2_0_0` — Health checks, shutdown, scopes, generics DI
- Context7 `/rs/zerolog` — Performance benchmarks (for comparison)
- Official Go documentation pkg.go.dev — log/slog handlers, levels, groups

### Secondary (MEDIUM confidence)
- GitHub uber-go/fx v1.24.0 release (2025-05-13)
- GitHub samber/do v2.0.0 release (2024-09-21)
- Internal dibx source code (`tmp/dibx/`)
- Internal gazx source code (`tmp/gazx/`)

### Tertiary (LOW confidence)
- General ecosystem patterns — validate during implementation

---
*Research completed: 2026-01-26*
*Ready for roadmap: yes*
