# Feature Research: v1.1 Security & Hardening

**Domain:** Application Robustness (Configuration & Lifecycle)
**Researched:** Mon Jan 26 2026
**Confidence:** HIGH

## Feature Landscape

### Table Stakes (Users Expect These)

Features users assume exist in a robust application framework. Missing these = "not production ready".

| Feature | Why Expected | Complexity | Notes |
|---------|--------------|------------|-------|
| **Struct Tag Validation** | Standard Go pattern. Reduces boilerplate `if` checks. | LOW | Use `go-playground/validator/v10`. Tags: `required`, `min`, `max`, `email`, `url`. |
| **Fail-Fast Config** | Invalid config must prevent startup. Silent errors cause runtime outages. | LOW | Panic or `os.Exit(1)` with clear error message immediately after load. |
| **Graceful Shutdown** | Prevent data loss/corruption on deploy/restart. | MEDIUM | Standard `context` propagation to all services. |
| **Signal Handling** | Standard Unix behavior (`SIGINT`, `SIGTERM`). | LOW | Use `os/signal` + `NotifyContext`. |
| **Shutdown Timeout** | Prevent stalled deployments (e.g., K8s `terminationGracePeriodSeconds`). | LOW | Default 30s. |

### Differentiators (Competitive Advantage)

Features that set the robustness apart from basic implementations.

| Feature | Value Proposition | Complexity | Notes |
|---------|-------------------|------------|-------|
| **Force Kill on Timeout** | Guarantees process exit even if goroutines are stuck. Prevents "zombie" processes. | MEDIUM | Framework must enforce `os.Exit(1)` if `Stop()` exceeds timeout. |
| **Double-Interrupt Exit** | Developer convenience: Ctrl+C twice forces immediate exit during shutdown. | LOW | Watch signal channel during shutdown phase. |
| **Cross-Field Validation** | Enforces business logic constraints (e.g., `CertFile` required if `TLS=true`). | MEDIUM | `validator` supports `required_with`, `required_if`. |
| **Custom Validators** | Domain-specific checks (e.g., "valid AWS region", "valid CIDR"). | MEDIUM | Allow users to register custom validation functions. |
| **Component Blaming** | Identifies *which* service stalled the shutdown. Critical for debugging hangs. | HIGH | Wrap `OnStop` calls with individual timers/logging. |

### Anti-Features (Commonly Requested, Often Problematic)

Features that seem good but create problems.

| Feature | Why Requested | Why Problematic | Alternative |
|---------|---------------|-----------------|-------------|
| **Silent Validation Failure** | "Keep running if possible" | App runs in undefined state; hard to debug later. | **Fail Fast:** Error & Exit on startup. |
| **Indefinite Wait** | "Finish all work no matter what" | Blocks deployments; requires manual kill. | **Timeout:** Always enforce a hard deadline (e.g., 30s). |
| **Partial Config Loading** | "Load what works" | Inconsistent application state. | **Atomic Load:** All valid or nothing starts. |
| **Global State Validation** | Simplicity | Hard to test; hidden dependencies. | **Scoped Validation:** Validate the struct instance only. |

## Feature Dependencies

```
[Config Loading (Viper)]
    └──requires──> [Struct Definition]
                       └──enhances──> [Config Validation (Validator v10)]
                                          └──enables──> [Safe App Startup]

[Safe App Startup]
    └──requires──> [Lifecycle Engine]
                       └──enables──> [Hardened Shutdown]
                                          └──requires──> [Signal Handling]
                                          └──requires──> [Timeout Enforcement]
```

### Dependency Notes

- **Config Validation requires Config Loading:** Validation happens *after* unmarshaling but *before* the config is used.
- **Hardened Shutdown requires Lifecycle Engine:** The engine must orchestrate the stop order (LIFO) and enforce the timeout.

## MVP Definition

### Launch With (v1.1)

Minimum features to achieve "Security & Hardening" goal.

- [ ] **Struct Tag Validation** — Integrate `go-playground/validator/v10` into `ConfigManager`. Support `required`, `min`, `max`, `email`, `url`.
- [ ] **Fail-Fast Behavior** — `app.Run()` returns error immediately if validation fails.
- [ ] **Shutdown Timeout Enforcement** — Ensure `Stop()` respects the timeout context.
- [ ] **Force Exit** — If shutdown times out, log error and `os.Exit(1)`.

### Add After Validation (v1.2)

- [ ] **Cross-Field Validation** — When complex dependency configurations arise.
- [ ] **Custom Validator Registration** — When users need domain-specific rules.

### Future Consideration (v2+)

- [ ] **Component Blaming** — If debugging shutdown hangs becomes a common support issue.

## Feature Prioritization Matrix

| Feature | User Value | Implementation Cost | Priority |
|---------|------------|---------------------|----------|
| **Struct Tag Validation** | HIGH | LOW | P1 |
| **Fail-Fast Config** | HIGH | LOW | P1 |
| **Shutdown Timeout** | HIGH | LOW | P1 |
| **Force Kill** | HIGH | MEDIUM | P1 |
| **Double-Interrupt Exit** | MEDIUM | LOW | P2 |
| **Custom Validators** | MEDIUM | MEDIUM | P2 |
| **Cross-Field Validation** | MEDIUM | MEDIUM | P2 |
| **Component Blaming** | MEDIUM | HIGH | P3 |

## Competitor Feature Analysis

| Feature | Uber fx | Google Wire | Our Approach (v1.1) |
|---------|---------|-------------|---------------------|
| **Validation** | Manual `fx.Option` or `OnStart` checks | Compile-time checks (limited) | **Struct Tags (Validator v10)** |
| **Shutdown** | `fx.StopTimeout`, context propagation | Manual context handling | **Managed Timeout + Force Kill** |
| **Config** | External (usually Viper) | External | **Integrated Viper + Validation** |

## Sources

- **Go Validator:** `go-playground/validator` (Standard)
- **Config:** `spf13/viper` (Current)
- **Pattern:** [Uber Go Style Guide](https://github.com/uber-go/guide) (Zero-value safety, error handling)
- **Internal:** `lifecycle_engine.go` (Existing LIFO logic)

---
*Feature research for: v1.1 Security & Hardening*
*Researched: Mon Jan 26 2026*
