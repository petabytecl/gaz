# Roadmap: gaz AEGIS Remediation

## Overview

Systematic remediation of 45+ findings from the AEGIS diagnostic audit, organized into 4 phases matching the AEGIS execution plan tiers. Each phase has a verification gate. Total estimated effort: 5-7 days.

## Current Milestone

**v5.2 AEGIS Remediation** (v5.2.0)
Status: In progress
Phases: 0 of 4 complete

## Phases

| Phase | Name | Plans | Status | Completed |
|-------|------|-------|--------|-----------|
| 1 | Immediate Security & Safety | 2 | Complete | 2026-04-03 |
| 2 | Core Reliability & Benchmarks | 5 | Planning | - |
| 3 | Shutdown & Graph Integrity | 4 | Not started | - |
| 4 | Documentation & Governance | 4 | Not started | - |

## Phase Details

### Phase 1: Immediate Security & Safety

**Goal:** Fix the CRITICAL CVE and add DI container safety guards. Zero-risk, high-value changes.
**Depends on:** Nothing (first phase)
**Research:** Unlikely (all fixes are prescribed by AEGIS playbooks)
**AEGIS tier:** Tier 0
**Effort:** 2-3 hours

**Scope:**
- Bump grpc v1.79.1 → v1.79.3 (Playbook 01)
- Add `built` guard to Container.Register() (Playbook 04)
- Add checked type assertion in ResolveAll[T] (Playbook 04)
- Add nil guard in instanceServiceAny.ServiceType() (Playbook 04)

**Plans:**
- [x] 01-01: grpc CVE fix + govulncheck verification
- [x] 01-02: DI container safety guards (Register, ResolveAll, ServiceType)

**Verification gate:**
- `make test && make lint` passes
- `govulncheck ./...` clean
- Register/ReplaceService return errors after Build()

### Phase 2: Core Reliability & Benchmarks

**Goal:** Fix the top HIGH-severity findings: EventBus lock pattern, singleton deadlock risk, Cobra/Run divergence. Establish benchmarks first.
**Depends on:** Phase 1 (DI guard is prerequisite)
**Research:** Unlikely (all fixes are prescribed)
**AEGIS tier:** Tier 1
**Effort:** 2-3 days

**Scope:**
- Test isolation fixes + benchmarks (Playbook 09)
- EventBus snapshot-then-deliver pattern (Playbook 03)
- Singleton atomic fast-path (Playbook 05, Steps 1-2)
- Cobra Start()/Run() unification (Playbook 02)

**Plans:**
- [ ] 02-01: Test infrastructure — t.Setenv, port 0, time.Sleep removal
- [ ] 02-02: Benchmarks for DI resolution and EventBus (MUST do before 02-03/02-04)
- [ ] 02-03: EventBus synchronization fix (snapshot-then-deliver, unsubscribe lock fix)
- [ ] 02-04: Singleton atomic fast-path + Cobra/Run unification
- [ ] 02-05: Benchmark comparison — verify no regressions

**Verification gate:**
- Benchmark comparison shows no regressions
- `go test -race -count=3 ./...` passes
- Cobra integration: worker supervision confirmed

### Phase 3: Shutdown & Graph Integrity

**Goal:** Fix shutdown reliability issues and dependency graph correctness.
**Depends on:** Phase 2 (cached startup order from Cobra fix)
**Research:** Unlikely
**AEGIS tier:** Tier 2
**Effort:** 1-2 days

**Scope:**
- Cron context-bounded OnStop (Playbook 06)
- Health server synchronous bind (Playbook 06)
- stopOnce context fix (Playbook 06)
- Dependency graph edge dedup (Playbook 07)
- Conditional: ResolveByName fast-path (Playbook 05.3, if benchmarks justify)
- Conditional: Explicit graph building (Playbook 07.2, if lazy semantics acceptable)

**Plans:**
- [ ] 03-01: Cron + health shutdown fixes
- [ ] 03-02: stopOnce context fix + graph edge dedup
- [ ] 03-03: Conditional optimizations (05.3, 07.2) — based on Phase 2 benchmarks
- [ ] 03-04: Shutdown integration verification

**Verification gate:**
- Cron OnStop returns within deadline
- Health server reports bind failures
- No duplicate graph edges after Build
- `make test && make lint` passes

### Phase 4: Documentation & Governance

**Goal:** Fix doc/reality gaps, secure defaults, supply chain governance. Documentation must reflect behavior from Phases 1-3.
**Depends on:** Phase 3 (docs must match final behavior)
**Research:** Unlikely
**AEGIS tier:** Tier 3
**Effort:** 1 day

**Scope:**
- Fix troubleshooting docs, env var docs, README (Playbook 08)
- gRPC reflection default to false (Playbook 08)
- ErrDuplicate enforcement (Playbook 08)
- SHA-pin GitHub Actions + Dependabot (Playbook 10)
- SECURITY.md (Playbooks 08 + 10)

**Plans:**
- [ ] 04-01: Documentation fixes (troubleshooting, env vars, README, dead error types)
- [ ] 04-02: gRPC reflection default + ErrDuplicate enforcement
- [ ] 04-03: Supply chain governance (SHA pinning, Dependabot, SECURITY.md)
- [ ] 04-04: Final verification — all docs match code, CI green, govulncheck clean

**Verification gate (FINAL):**
- All documentation reviewed for accuracy
- CI green with SHA-pinned actions
- `govulncheck ./...` clean
- No orphaned sentinel errors
- SECURITY.md visible on GitHub

---
*Roadmap created: 2026-04-02*
*Source: AEGIS execution plan (.aegis/execution/execution-plan.md)*
