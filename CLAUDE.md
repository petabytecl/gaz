# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

`github.com/petabytecl/gaz` is a Go 1.26 type-safe dependency injection framework with lifecycle management. No code generation, minimal reflection in public API. Built on generics for compile-time safety.

## Common Commands

```bash
make test              # Run all tests with race detector
make cover             # Tests + coverage (90% threshold, excludes examples/)
make lint              # golangci-lint (v2, 50+ linters, strict)
make fmt               # gofmt + goimports
make fmt-check         # Verify formatting without modifying

# Single test
go test -race -run TestName ./path/to/package/

# Single package
go test -race ./di/
```

## Architecture

### Core Flow

**Build phase** (`App.Build()`): Collect ConfigProviders -> load config (files/env/flags) -> register providers -> build DI container -> compute dependency graph -> topological sort for startup order (Kahn's algorithm in `lifecycle_engine.go`).

**Run phase** (`App.Run()`): Start services in dependency order (parallel per layer) -> run workers/cron/eventbus -> wait for signal -> graceful shutdown in reverse order.

### Key Packages

- **`di/`** - Generic DI container. Fluent API: `di.For[T](c).Provider(fn)`. Scopes: Singleton (default), Transient, Eager, Instance. Named services. Struct injection via `gaz:"inject"` tags. Cycle detection per-goroutine via `goid`.

- **`config/`** - Configuration with `Backend` interface (Viper implementation). Manager runs `Defaulter.Default()` -> validate with `go-playground/validator` -> `Validator.Validate()`. Strict mode rejects unknown keys.

- **`worker/`** - Long-running background tasks. Supervisor wraps workers with panic recovery, exponential backoff (1s-5m, 2x, jitter), and circuit breaker. Options: `WithPoolSize`, `WithCritical`, `WithMaxRestarts`.

- **`cron/`** - Scheduled jobs via `robfig/cron/v3`. Implements `CronJob` interface (Name/Schedule/Timeout/Run). SkipIfStillRunning by default. Return empty `Schedule()` to disable.

- **`health/`** - Kubernetes-aligned probes (liveness/readiness/startup) on a dedicated management port (default 9090). `ShutdownCheck` auto-fails readiness during shutdown.

- **`eventbus/`** - Type-safe pub/sub. `Subscribe[T](bus, handler)` / `Publish[T](bus, event, topic)`. Async with configurable buffer (default 100).

- **`server/`** - Unified transport: gRPC + Connect + gRPC-Web + REST via Vanguard on single h2c port. gRPC registers services but skips its own listener; Vanguard handles all connections.

- **`logger/`** - slog-based with context propagation (trace ID, request ID). `ContextHandler` wraps any `slog.Handler`. HTTP middleware for X-Request-ID.

- **`gaztest/`** - Test framework with builder pattern: `gaztest.New(t).WithModules(...).Build()`. Per-subsystem test helpers in each package's `testing.go` (MockWorker, MockJob, MapBackend, etc.). Use port 0 for random available ports.

### Key Patterns

- **Module pattern**: `app.Use(module)` or `app.Module("name", registrations...)`
- **ConfigProvider**: Services declare their config namespace and flags; framework auto-discovers during Build
- **Error convention**: `di:` prefix format, sentinel errors as `di.ErrNotFound`, re-exported as `gaz.ErrDINotFound`
- **Lifecycle interfaces**: Types implementing `Starter`/`Stopper` are auto-discovered and ordered by dependency graph

## Linting Notes

The `.golangci.yml` is strict: gocognit limit 20, funlen 100 lines/50 statements, depguard with allowlists, sloglint enforces context-based logging (no global loggers). Test files and examples have relaxed rules.

## Security RULES (not enforced by golangci-lint)

## Rule 1: No Lock-During-Blocking-IO

- **Prevents:** F-03-006, F-08-004, F-08-007, F-07-004
- **Rule:** Never hold a mutex (sync.Mutex or sync.RWMutex) while performing a blocking operation (channel send, channel receive, network I/O, context wait, or sync.WaitGroup.Wait). Snapshot the data under lock, release the lock, then perform the blocking operation outside the lock. Use `defer` only when the critical section does not contain blocking calls.
- **Rationale:** The EventBus held RLock during channel sends, causing cascading backpressure that could hang shutdown. The cron scheduler held its mutex while waiting for jobs to drain. This pattern appears in 3+ subsystems and is the root cause of the "lock-during-blocking-IO" cross-cutting finding.
- **Invalidation:** This rule is permanent. It reflects a fundamental Go concurrency best practice.

## Rule 2: Single Startup Path

- **Prevents:** F-01-007, F-RG-011, F-RG-012, F-11-007
- **Rule:** All application startup paths (Run, Start, Cobra commands) MUST delegate to a single shared private method for service lifecycle management. Never duplicate worker filtering, startup ordering, or parallel-layer startup logic. If a new entry point is added, it must call the shared method — never re-implement startup inline.
- **Rationale:** The Start() method in cobra.go diverged from Run() in app_run.go, silently losing worker supervision, parallel startup, and panic recovery for all Cobra-based applications. This was the strongest finding in the audit.
- **Invalidation:** Remove if the framework removes the Cobra integration or consolidates to a single entry point.

## Rule 3: Context Propagation in OnStop

- **Prevents:** F-07-004, F-07-005, F-07-001
- **Rule:** Every OnStop(ctx context.Context) implementation MUST respect the context deadline. Use `select { case <-done: case <-ctx.Done(): }` when waiting for goroutines, channel drains, or external operations to complete. Never block indefinitely in a shutdown path. Log a warning when the deadline is exceeded and return ctx.Err().
- **Rationale:** Multiple subsystems (cron, eventbus, health) ignored the shutdown context, blocking until the global force-exit timer fired os.Exit(1). This defeats graceful shutdown and can cause data loss.
- **Invalidation:** This rule is permanent. Context deadline respect is a Go convention.

## Rule 4: DI Registration After Build Guard

- **Prevents:** F-02-001, F-09-001, F-RG-001, F-RG-002
- **Rule:** Container.Register() and Container.ReplaceService() MUST return an error when called after Build(). The ErrAlreadyBuilt sentinel error exists for this purpose. Never add a registration method that bypasses the built flag check. Test this guard in the DI test suite.
- **Rationale:** The container defined ErrAlreadyBuilt but never checked the flag in Register(), creating a phantom contract. Post-Build registration silently succeeds but produces services with incorrect lifecycle ordering and missing dependency graph edges.
- **Invalidation:** Remove if the container is redesigned to support dynamic registration (e.g., plugin hot-loading).

## Rule 5: Documentation Must Match Implementation

- **Prevents:** F-RG-003, F-RG-005, F-RG-006, F-RG-008
- **Rule:** When changing runtime behavior (error returns, default values, naming conventions), update the corresponding documentation in the same commit. When adding sentinel errors or typed error structs to the public API, add at least one code path that returns them. Run `grep` for the error name across docs/ to verify consistency. Do not merge documentation that references behavior the code does not implement.
- **Rationale:** The audit found 4 documentation-reality gaps: troubleshooting docs describing impossible errors, README claiming "no reflection" with 46+ reflect calls, docs claiming ErrDuplicate is returned when it never is, and env var docs showing the wrong naming convention. These erode user trust.
- **Invalidation:** This rule is permanent. It is a documentation hygiene practice.

## Rule 6: Test Isolation for Environment and Ports

- **Prevents:** F-06-001, F-06-003, F-06-007
- **Rule:** In test files, use t.Setenv() (never os.Setenv), use port 0 for all server bindings (never hardcoded ports), and use require.Eventually() or channel synchronization (never time.Sleep) for waiting on async conditions. These are enforced by golangci-lint custom rules where available.
- **Rationale:** ~20 os.Setenv call sites make parallel test execution unsafe. Hardcoded ports cause CI flakiness. time.Sleep synchronization is fragile under load and in CI environments with variable CPU availability.
- **Invalidation:** This rule is permanent. It reflects Go testing best practices.

## Rule 7: gRPC Reflection Off by Default

- **Prevents:** F-04-005
- **Rule:** gRPC server reflection MUST default to false in all config structs (server/vanguard/config.go, server/grpc/config.go). Enable reflection only via explicit configuration, CLI flag, or DevMode. Document this in the config struct's field comments.
- **Rationale:** gRPC reflection exposes the full service schema to any client. In production, this enables reconnaissance by attackers. The framework defaulted to reflection=true, meaning every application built with gaz exposed its API surface unless explicitly disabled.
- **Invalidation:** Remove if the framework adds a DevMode auto-detection that safely toggles reflection based on environment.

## Rule 8: GitHub Actions SHA Pinning

- **Prevents:** F-05-001
- **Rule:** All GitHub Actions in .github/workflows/ MUST be pinned to full SHA digests, not mutable tags (v4, v5, etc.). Include the human-readable version as a comment after the SHA (e.g., `@abc123 # v4.1.1`). Use Dependabot's github-actions ecosystem to receive update PRs when new versions are released.
- **Rationale:** Mutable tags can be force-pushed by action maintainers (or attackers who compromise the repository). SHA pinning ensures reproducible CI and prevents supply chain attacks. This is an industry standard for security-conscious open-source projects.
- **Invalidation:** This rule is permanent for public repositories. May be relaxed for private repos with trusted action sources.
