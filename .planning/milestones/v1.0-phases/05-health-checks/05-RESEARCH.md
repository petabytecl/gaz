# Phase 05: Health Checks - Research

**Researched:** 2026-01-26
**Domain:** Application Health & Observability
**Confidence:** HIGH

## Summary

This phase implements production-ready health endpoints using the `alexliesenfeld/health` library, which provides the necessary concurrency, caching, and timeout logic. While the library's default output is close to the IETF standard, a custom `ResultWriter` is required to achieve strict compliance with the IETF Health Check draft as requested.

The implementation distinguishes strictly between **Liveness** (restart required) and **Readiness** (stop traffic), with distinct status code mappings (Liveness failures -> 200 OK, Readiness failures -> 503). A "Shutdown Hook" pattern ensures readiness probes fail immediately upon application shutdown signals, preventing traffic from being routed to terminating pods.

**Primary recommendation:** Use `alexliesenfeld/health` with a custom `ResultWriter` for IETF compliance and specific shutdown-aware readiness checks.

## Standard Stack

### Core
| Library | Version | Purpose | Why Standard |
|---------|---------|---------|--------------|
| `github.com/alexliesenfeld/health` | `v0.8.0`+ | Health Check Engine | Industry standard, supports concurrency, timeouts, and custom status codes. |

### Supporting
| Library | Version | Purpose | When to Use |
|---------|---------|---------|-------------|
| `context` | stdlib | Timeout propagation | Mandatory for all checks. |
| `os/signal` | stdlib | Shutdown detection | For the Shutdown Readiness Checker. |

**Installation:**
```bash
go get github.com/alexliesenfeld/health
```

## Architecture Patterns

### 1. DI Auto-Discovery (Multibinding)
Checks should be registered via Dependency Injection using a "Multibinding" or "Group" pattern. This allows decentralized registration of checks from different modules.

**Example (Conceptual DI):**
```go
// In infrastructure module
func ProvideDBCheck() HealthChecker { ... }

// In health module
type Params struct {
    fx.In
    Checkers []HealthChecker `group:"health_checkers"`
}

func NewHealthHandler(p Params) http.Handler {
    // Register all p.Checkers
}
```

### 2. Strict IETF Response Adapter
The default JSON output of `alexliesenfeld/health` uses a simplified `details` object map. The IETF draft specifies a `checks` object where values are *arrays* of objects (to support multiple nodes per component).

**Pattern:** Implement `health.WithResultWriter` to transform the `health.CheckerResult` into the strict IETF structure.

```go
// Strict IETF Structure
type IETFResponse struct {
    Status string              `json:"status"` // "pass", "fail", "warn"
    Checks map[string][]Check  `json:"checks,omitempty"`
    // ... other IETF fields
}

// Adapter
func IETFResultWriter(w http.ResponseWriter, r *http.Request, status int, result *health.CheckerResult) {
    // Transform result -> IETFResponse
    // Write JSON
}
```

### 3. Shutdown-Aware Readiness
Readiness probes must fail *immediately* when the app receives a SIGTERM, even while the app is still draining requests.

**Pattern:**
1. Create a `ShutdownReadinessChecker`.
2. Register an `OnStop` lifecycle hook (or signal handler) to flip its state to `false`.
3. Register this checker as a **Readiness** check.

```go
type ShutdownChecker struct {
    shuttingDown atomic.Bool
}

func (c *ShutdownChecker) Check(ctx context.Context) error {
    if c.shuttingDown.Load() {
        return errors.New("application is shutting down")
    }
    return nil
}

// In Lifecycle OnStop:
c.shuttingDown.Store(true)
```

## Don't Hand-Roll

| Problem | Don't Build | Use Instead | Why |
|---------|-------------|-------------|-----|
| Check Execution | Custom goroutines & WaitGroups | `alexliesenfeld/health` | Handles concurrency, timeouts, and error aggregation correctly. |
| HTTP Handler | `http.HandleFunc("/health", ...)` | `health.NewHandler()` | Provides standard headers, method handling, and caching out of the box. |

**Key insight:** Concurrency + Timeouts + Error aggregation is harder than it looks to get right. Use the library's engine.

## Common Pitfalls

### Pitfall 1: Liveness Dependency Checking
**What goes wrong:** Liveness probe checks external dependencies (e.g., Database).
**Why it happens:** Developer copy-pastes readiness checks to liveness.
**Result:** DB goes down -> App restarts -> App still can't connect -> App restarts loop.
**Prevention:** Liveness checks should *only* check if the Go runtime is stuck (deadlock) or internal invariants are broken.

### Pitfall 2: Default Status Codes
**What goes wrong:** Readiness returns 200 with "status: down" body.
**Why it happens:** Default configuration often returns 200 for everything.
**Result:** Load balancers (Kubernetes) don't see the failure and keep sending traffic.
**Prevention:** Explicitly configure `WithStatusCodeDown(503)` for Readiness.

### Pitfall 3: Blocking Checks
**What goes wrong:** A check hangs forever (e.g., DB lock).
**Why it happens:** Ignoring `context.Context` in the check function.
**Result:** Health endpoint hangs, orchestrator times out and kills the pod aggressively.
**Prevention:** ALWAYS respect `ctx` in checks.

## Code Examples

### 1. IETF-Compliant Setup
```go
// Source: alexliesenfeld/health docs & IETF draft
checker := health.NewChecker(
    
    // Global Timeout
    health.WithTimeout(10 * time.Second),

    // Check Configuration
    health.WithCheck(health.Check{
        Name: "database",
        Check: func(ctx context.Context) error {
            return db.Ping(ctx) // Respect context!
        },
    }),
    
    // Verify IETF Output
    health.WithResultWriter(NewIETFResultWriter()),
)

// Readiness Handler (503 on failure)
readinessHandler := health.NewHandler(checker,
    health.WithStatusCodeUp(http.StatusOK),
    health.WithStatusCodeDown(http.StatusServiceUnavailable), // 503
)

// Liveness Handler (200 on failure, unless dead)
livenessHandler := health.NewHandler(livenessChecker,
    health.WithStatusCodeUp(http.StatusOK),
    health.WithStatusCodeDown(http.StatusOK), // 200 (Degraded != Dead)
)
```

## State of the Art

| Old Approach | Current Approach | Impact |
|--------------|------------------|--------|
| Simple 200 OK | Rich JSON (IETF) | Standardized observability across tools. |
| Shared Checks | Split Liveness/Readiness | Prevents cascading restart loops. |
| Synchronous | Concurrent Execution | Faster probes, less impact on main loop. |

## Open Questions

1. **DI Auto-Discovery Implementation**
   - We assume the project's custom DI supports `group` or multibinding. If not, manual registration in a central `HealthModule` will be required.

## Sources

### Primary (HIGH confidence)
- Context7 ID `/alexliesenfeld/health` - Checked features, custom status codes, and concurrency support.
- IETF Draft `draft-inadarei-api-health-check-06` - Verified JSON schema.
- Kubernetes Docs `configure-liveness-readiness-startup-probes` - Verified status code behavior.

### Secondary (MEDIUM confidence)
- GitHub Issues (alexliesenfeld/health#41) - Confirmed concurrency support added in recent versions.

## Metadata

**Confidence breakdown:**
- Standard stack: HIGH - `alexliesenfeld/health` is the clear leader.
- Architecture: HIGH - Kubernetes patterns are well-documented.
- Pitfalls: HIGH - Common industry knowledge.

**Research date:** 2026-01-26
**Valid until:** 2027-01-26
