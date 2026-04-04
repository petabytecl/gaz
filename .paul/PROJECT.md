# gaz — AEGIS Remediation

## What This Is

Systematic remediation of findings from the AEGIS diagnostic audit of the gaz DI framework. 10 playbooks addressing 45+ findings across security, reliability, correctness, performance, documentation, and governance domains. Driven by the AEGIS execution plan with 4 phased implementation tiers.

## Core Value

Ship a safer, more reliable gaz framework by fixing the 1 CRITICAL CVE, 10 HIGH-severity findings, and 3 cross-cutting anti-patterns identified by the 12-agent AEGIS audit.

## Current State

| Attribute | Value |
|-----------|-------|
| Type | Application |
| Version | 5.1.0 |
| Status | Remediation |
| Last Updated | 2026-04-02 |

## Requirements

### Core Features

- Fix CVE-2026-33186 (grpc v1.79.1 → v1.79.3)
- Unify Cobra Start() and Run() startup paths (strongest audit finding)
- Fix EventBus lock-during-blocking-IO pattern
- Add DI container safety guards (built guard, nil guard, checked assertion)
- Add atomic fast-path for singleton resolution
- Fix shutdown reliability (cron, health, stopOnce)
- Fix dependency graph correctness (dedup edges, explicit building)
- Fix documentation/reality gaps (sentinel errors, env vars, reflection)
- Improve test infrastructure (t.Setenv, port 0, benchmarks)
- Add supply chain governance (SHA-pinned actions, Dependabot, SECURITY.md)

### Validated (Shipped)
None yet.

### Active (In Progress)
None yet.

### Planned (Next)
Phase 1: Immediate Security and Safety (Tier 0)

### Out of Scope

- health/checks/ packages (blind spot — separate follow-up audit)
- cron/internal/ forked parser (blind spot — separate follow-up audit)
- OTEL provider lifecycle (blind spot)
- backoff package correctness (blind spot)
- Connect interceptor ordering (blind spot)
- App struct extraction / god object refactoring (deferred — Tier 3 strategic, not in this milestone)

## Constraints

### Technical Constraints

- Playbook 09 (benchmarks) MUST complete before Playbooks 03 and 05
- Playbooks 02, 05, 07 have HIGH risk dimensions — require extra review
- Playbook 04 is prerequisite for Playbook 08 Step 4
- 2 conditional items (05.3, 07.2) depend on benchmark evidence

### Business Constraints

- Bus factor = 1 (single maintainer)
- No breaking public API changes without major version bump
- ~5-7 days estimated total effort

## Key Decisions

| Decision | Rationale | Date | Status |
|----------|-----------|------|--------|
| Import from AEGIS audit | 12-agent audit produced prioritized playbooks | 2026-04-02 | Active |
| Benchmarks before perf fixes | Need baselines to measure improvement | 2026-04-02 | Active |
| Defer god object extraction | Stabilize current design first | 2026-04-02 | Active |
| Defer 07.2 (eager graph) | May break lazy singleton semantics | 2026-04-02 | Active |

## Success Metrics

| Metric | Target | Current | Status |
|--------|--------|---------|--------|
| CRITICAL findings | 0 | 1 | Not started |
| HIGH findings resolved | 10 | 0 | Not started |
| Playbooks completed | 10 | 0 | Not started |
| Tests passing | 100% | 100% | On track |
| Coverage threshold | 90% | 90%+ | On track |

## Tech Stack / Tools

| Layer | Technology | Notes |
|-------|------------|-------|
| Language | Go 1.26 | DI framework |
| Audit source | AEGIS | 12-agent diagnostic audit |
| Execution plan | .aegis/execution/ | 4-phase sequenced plan |
| Playbooks | .aegis/remediation/playbooks/ | 10 remediation playbooks |

## Links

| Resource | URL |
|----------|-----|
| AEGIS Report | .aegis/report/AEGIS-REPORT.md |
| Execution Plan | .aegis/execution/execution-plan.md |
| Risk Scores | .aegis/execution/risk-scores.md |
| Guardrails | .aegis/remediation/guardrails/claude-rules.md |

---
*Created: 2026-04-02*
