# Project Research Summary

**Project:** Security & Hardening v1.1
**Domain:** Application Robustness (Configuration & Lifecycle)
**Researched:** Mon Jan 26 2026
**Confidence:** HIGH

## Executive Summary

This project focuses on enhancing application robustness through configuration validation and hardened lifecycle management. Research indicates that a production-ready Go application must fail fast on invalid configuration and guarantee graceful, bounded shutdown to prevent data loss or "zombie" processes. The industry standard is to treat configuration validity as a precondition for startup and shutdown limits as a hard guarantee.

The recommended approach introduces two "Hardening Gates": a startup gate using `go-playground/validator` to enforce structural configuration integrity before dependency injection, and a shutdown guard using `context` and `os/signal` to enforce a strict timeout (default 30s) on application stops. This moves validation from runtime checks to startup enforcement and transforms shutdown logic from "best effort" to "guaranteed exit."

Key risks include "zero-value ambiguity" where missing config is mistaken for default values, and blocked shutdowns where services hang indefinitely. These are mitigated by using pointer types for optional fields, explicit `required` tags, and wrapping the lifecycle manager in a timeout context that forces `os.Exit(1)` if graceful shutdown fails.

## Key Findings

### Recommended Stack

Research strongly favors integrating validation directly with the existing struct-based config loading.

**Core technologies:**
- **`go-playground/validator` (v10):** Struct-based validation — The de-facto standard for Go; integrates seamlessly with `koanf` struct tags to allow declarative validation rules (e.g., `required`, `min`, `max`).
- **`context` (Stdlib):** Timeout management — Essential for enforcing bounded execution times during shutdown.
- **`os/signal` (Stdlib):** Signal handling — Standard mechanism to intercept `SIGTERM`/`SIGINT` for graceful stops.

### Expected Features

**Must have (table stakes):**
- **Struct Tag Validation:** Declarative rules on config structs (e.g., `validate:"required"`).
- **Fail-Fast Config:** Application must exit immediately if validation fails at startup.
- **Graceful Shutdown:** `SIGINT`/`SIGTERM` triggers a coordinated stop of all services.
- **Shutdown Timeout:** Hard limit (default 30s) on how long shutdown can take.

**Should have (competitive):**
- **Force Kill on Timeout:** Guarantee process exit even if goroutines are stuck.
- **Cross-Field Validation:** Rules that depend on multiple fields (e.g., "TLS cert required if HTTPS enabled").
- **Double-Interrupt Exit:** Developer convenience to force exit by pressing Ctrl+C twice.

**Defer (v2+):**
- **Component Blaming:** Complex logic to identify exactly which service stalled the shutdown.
- **Custom Validators:** User-defined validation logic (stick to standard tags for v1.1).

### Architecture Approach

The architecture inserts control gates at the edges of the application lifecycle.

**Major components:**
1.  **Startup Gate (Validator):** Validates the `Config` struct immediately after unmarshalling. Prevents the DI container from ever seeing an invalid config.
2.  **Shutdown Guard (Timeout):** Wraps the `Lifecycle.Stop()` method. Races the graceful stop logic against a `context.WithTimeout`.
3.  **Strict Config Object:** A pattern where consumers (services) receive a config object that is guaranteed to be valid, removing the need for defensive checks inside business logic.

### Critical Pitfalls

1.  **Zero-Value Ambiguity:** Missing fields defaulting to `0` or `""`.
    *   *Avoid by:* Using pointer types (`*int`) for optional fields or `required` tags.
2.  **Blocked Shutdown:** `OnStop` hooks hanging indefinitely.
    *   *Avoid by:* The "Shutdown Guard" pattern—always enforcing a `os.Exit(1)` fallback after the timeout.
3.  **Unfriendly Validation Errors:** Cryptic validator output causing user frustration.
    *   *Avoid by:* Translating validation errors into human-readable messages before exiting.

## Implications for Roadmap

Based on research, the implementation should follow a dependency-based order: Validation -> Lifecycle -> Integration.

### Phase 1: Validation Engine
**Rationale:** Configuration validity is a dependency for all other components. We must ensure the `Config` object is correct before passing it to the Lifecycle engine or other services.
**Delivers:** Integration of `go-playground/validator` into the config loader, struct tags on config structs, and friendly error reporting.
**Addresses:** "Struct Tag Validation" and "Fail-Fast Config" features.
**Avoids:** "Zero-Value Ambiguity" pitfall.

### Phase 2: Hardened Lifecycle
**Rationale:** Once we can start safely (Phase 1), we focus on stopping safely. This involves the complex concurrency logic of signals and timeouts.
**Delivers:** `ShutdownGuard` implementation, `context`-propagation for `Stop()` methods, and signal handling.
**Uses:** `context`, `os/signal`.
**Implements:** "Shutdown Guard" architecture component.
**Avoids:** "Blocked Shutdown" and "Just Kill It" anti-patterns.

### Phase 3: CLI & Entrypoint Integration
**Rationale:** Wires the new engines into the main application entry point (`main.go` or CLI command).
**Delivers:** Updates to `app.Run()` to use the new Validation and Lifecycle logic, ensuring the "Gates" are active.
**Addresses:** "Force Kill on Timeout" and overall wiring.

### Phase Ordering Rationale

- **Dependencies:** You cannot reliably run a lifecycle engine with invalid configuration, so Validation comes first.
- **Risk Control:** Lifecycle logic (Phase 2) is harder to test and debug than Validation (Phase 1). Separating them allows focused testing on the concurrency aspects without noise from config errors.
- **Architecture:** Phase 1 solidifies the "Input" gate, Phase 2 solidifies the "Output" gate, and Phase 3 connects them to the user interface (CLI).

### Research Flags

**Standard patterns (skip research-phase):**
- **Phase 1 (Validation):** `go-playground/validator` is extremely well-documented and standard. No deep research needed.
- **Phase 2 (Lifecycle):** The `context.WithTimeout` + `select` pattern is standard Go concurrency.

## Confidence Assessment

| Area | Confidence | Notes |
|------|------------|-------|
| Stack | HIGH | `validator` is the clear industry standard. |
| Features | HIGH | Table stakes are well-defined for production Go apps. |
| Architecture | HIGH | The "Gates" pattern is a simplified version of standard frameworks like Uber fx. |
| Pitfalls | HIGH | Zero-value issues are a known Go specific pain point. |

**Overall confidence:** HIGH

### Gaps to Address

- **Validation Error UX:** While we know we *need* friendly errors, the specific format/style isn't defined. Needs a quick design decision during Phase 1.
- **Timeout Tuning:** The default 30s is a guess. We may need to make this configurable via environment variable immediately if deployment targets vary wildly.

## Sources

### Primary (HIGH confidence)
- **Go Validator:** `go-playground/validator` (GitHub/Godoc) — Standard library for struct validation.
- **Uber Go Style Guide:** Patterns for zero-value safety and error handling.
- **Standard Library:** `context` and `os/signal` documentation.

### Secondary (MEDIUM confidence)
- **Internal:** `lifecycle_engine.go` — Analyzed existing LIFO logic to ensure compatibility.
