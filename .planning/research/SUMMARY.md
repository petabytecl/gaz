# Project Research Summary

**Project:** gaz v2.0 Cleanup & Concurrency
**Domain:** Go Application Framework + Concurrency Primitives
**Researched:** 2026-01-27
**Confidence:** HIGH

## Executive Summary

gaz v2.0 adds in-process concurrency primitives (workers, worker pools, cron, eventbus) that integrate with the existing lifecycle system. The research reveals that gaz already has the core infrastructure needed—`Starter`/`Stopper` interfaces, dependency-ordered startup/shutdown, and per-hook timeouts. The main work is creating new packages that leverage these primitives, not modifying core architecture. The recommended approach is **stdlib-first for workers** (goroutines + channels are sufficient for most cases), **robfig/cron** for scheduling (only required dependency), and a **custom type-safe eventbus** using Go generics.

The key differentiator for gaz is **deep lifecycle integration**. Most Go libraries for workers/cron/eventbus are standalone—they don't integrate with DI containers or application lifecycle. gaz can provide seamless integration where workers, cron jobs, and event handlers are automatically managed by the existing lifecycle system, with proper dependency ordering and graceful shutdown.

The critical risks are goroutine leaks (workers that don't respect context cancellation), shutdown hangs (OnStop hooks blocking beyond timeout), and lifecycle ordering issues (workers depending on services that shut down first). These are well-understood pitfalls with established prevention patterns documented in PITFALLS.md. All can be mitigated through careful API design and mandatory patterns like context cancellation, done channels, and panic recovery.

## Key Findings

### Recommended Stack

The stack follows a **minimal dependency** philosophy. Only one new external dependency is required (robfig/cron), with two optional libraries for advanced use cases.

**Core technologies:**
- **Go stdlib (goroutines + channels)**: Simple workers — zero dependencies, sufficient for most cases
- **robfig/cron v3.0.1**: Scheduling — industry standard, graceful shutdown, job wrappers (only required dep)
- **alitto/pond v2.6.0**: Worker pools (optional) — context-aware, `pool.StopAndWait()` for graceful shutdown
- **Custom stdlib**: EventBus — type-safe generics, ~50 LOC, full control
- **jilio/ebu v0.10.1**: EventBus advanced (optional) — if persistence or advanced features needed

**Why NOT other options:**
- ants v2.11.0: Excellent, but `Release()`/`ReleaseTimeout()` doesn't match gaz's `OnStop(context.Context)`
- Distributed job queues (asynq, machinery, faktory): Out of scope—gaz targets in-process concurrency
- Complex eventbus libraries: Either build simple (50 LOC) or use ebu for advanced; middle ground adds deps without clear benefit

### Expected Features

**Must have (table stakes):**
- Context cancellation for graceful shutdown
- Error handling without crashing the app
- Panic recovery in all background work
- Lifecycle integration (Start/Stop with app)
- Wait for completion on shutdown
- Logging integration with existing `*slog.Logger`
- Concurrency limiting for worker pools

**Should have (differentiators):**
- DI-aware workers, jobs, handlers (inject dependencies from container)
- Automatic registration (workers implementing interface auto-start in lifecycle)
- Health check integration (worker status feeds into `/healthz`)
- Named workers/jobs for logging and debugging
- Restart policies with backoff for crashed workers
- SkipIfStillRunning/DelayIfStillRunning for cron jobs

**Defer (v2+):**
- Distributed workers (use asynq for this)
- Persistent job history
- Priority queues
- Metrics/Prometheus integration (complex, phase-specific)
- Wildcard event subscriptions

### Architecture Approach

All new concurrency components integrate via the existing `Starter`/`Stopper` interfaces. Workers spawn goroutines in `OnStart` (never block), signal and wait in `OnStop`. The dependency graph ensures proper ordering—workers depending on DB start after DB, stop before DB. New packages (`worker/`, `cron/`, `eventbus/`) wrap external libraries with lifecycle hooks.

**Major components:**
1. **worker/**: `Worker` interface with `Run(ctx)`, `WorkerManager` for tracking multiple workers, optional `Pool` for queued tasks
2. **cron/**: `Scheduler` wrapping robfig/cron, DI-aware job registration, panic recovery by default
3. **eventbus/**: Type-safe generics pub/sub, sync by default with async option, `Publish[T]`/`Subscribe[T]` API

**Data flow:**
- Workers added to container as eager singletons -> auto-discovered during Build -> started in dependency order during Run -> stopped in reverse order during Stop
- EventBus is singleton, publishers/subscribers resolve it from container, no lifecycle needed for in-memory version
- Cron scheduler starts background goroutine in OnStart, returns stop context in OnStop for graceful wait

### Critical Pitfalls

1. **Goroutine leaks (WRK-1, CRITICAL)** — Require `context.Context` in worker interface; all worker loops must check `<-ctx.Done()`
2. **Blocking OnStart (WRK-2, HIGH)** — Document that OnStart must spawn goroutine and return immediately, never run work synchronously
3. **No Done channel (WRK-3, HIGH)** — All workers must expose Done() channel; OnStop waits on done OR context timeout
4. **Panic crashes (WRK-4, CRN-2, HIGH)** — robfig/cron v3 no longer recovers panics by default; gaz cron wrapper MUST include `cron.Recover()` wrapper; workers need defer/recover
5. **Overlapping cron jobs (CRN-1, HIGH)** — Default to `cron.SkipIfStillRunning()` to prevent jobs from piling up
6. **Lifecycle ordering (INT-1, CRITICAL)** — Workers must declare dependencies explicitly; design workers to not publish during OnStop

## Implications for Roadmap

Based on research, suggested phase structure:

### Phase 0: Cleanup & DI Extraction
**Rationale:** Must clean house before adding new features. Removes deprecated code, extracts DI into `gaz/di` package.
**Delivers:** Clean foundation, `gaz/di` package
**Addresses:** Technical debt cleanup, clearer package boundaries
**Avoids:** Building on top of deprecated patterns

### Phase 1: Base Workers
**Rationale:** Foundation for all background work. No external dependencies. Establishes patterns for other phases.
**Delivers:** `Worker` interface, `WorkerManager`, lifecycle integration
**Addresses:** Context cancellation, panic recovery, done channel pattern
**Avoids:** WRK-1 (goroutine leaks), WRK-2 (blocking OnStart), WRK-3 (no done channel)

### Phase 2: Worker Pools
**Rationale:** Builds on Phase 1 infrastructure. Optional pond dependency.
**Delivers:** `Pool` with Submit/StopWait, bounded queue option
**Uses:** Go stdlib or alitto/pond v2.6.0
**Addresses:** Concurrency limiting, wait for completion
**Avoids:** WRK-5 (buffer sizing issues)

### Phase 3: Cron Scheduler
**Rationale:** Independent of workers, can parallel with Phase 2. Wraps robfig/cron.
**Delivers:** `Scheduler` with lifecycle, DI-aware jobs, panic recovery
**Uses:** robfig/cron v3.0.1
**Implements:** Job registration, SkipIfStillRunning by default, graceful stop pattern
**Avoids:** CRN-1 (overlapping jobs), CRN-2 (no panic recovery), CRN-3 (improper stop)

### Phase 4: EventBus
**Rationale:** Independent component. Type-safe generics API matching gaz style.
**Delivers:** `EventBus` with Publish[T]/Subscribe[T], sync and async modes
**Uses:** Custom stdlib implementation (or jilio/ebu if advanced features needed)
**Implements:** Bounded buffer, context propagation
**Avoids:** EVT-1 (unbounded buffer), EVT-5 (deadlock from sync publish)

### Phase 5: Integration & Polish
**Rationale:** Polish after core functionality works.
**Delivers:** Health check integration, examples, documentation
**Addresses:** Health integration for all components
**Avoids:** INT-3 (per-hook timeout sizing), INT-5 (dependency ordering)

### Phase Ordering Rationale

- **Cleanup first (Phase 0):** Can't build new features on deprecated code
- **Workers before pools (Phase 1 -> 2):** Pool uses worker patterns, not vice versa
- **Cron independent (Phase 3):** Can be parallel with Phase 2 since no dependency
- **EventBus after workers (Phase 4):** Subscribers may be workers, need worker patterns established first
- **Integration last (Phase 5):** Requires all components to exist

### Research Flags

**Phases likely needing deeper research during planning:**
- **Phase 0 (Cleanup):** Need inventory of deprecated code and migration paths
- **Phase 5 (Integration):** Metrics interface design, Prometheus integration patterns

**Phases with standard patterns (skip research-phase):**
- **Phase 1 (Workers):** Well-documented patterns from uber-go/fx, standard Go concurrency
- **Phase 2 (Worker Pools):** Established pattern from gammazero/workerpool, pond
- **Phase 3 (Cron):** robfig/cron is mature, Context7 docs comprehensive
- **Phase 4 (EventBus):** Simple pattern, jilio/ebu provides reference if needed

## Confidence Assessment

| Area | Confidence | Notes |
|------|------------|-------|
| Stack | HIGH | Context7 verified for all libraries, explicit version pins |
| Features | HIGH | Analyzed multiple established Go frameworks, clear table stakes |
| Architecture | HIGH | Verified integration with existing gaz Starter/Stopper, uber-go/fx patterns |
| Pitfalls | HIGH | Context7 docs, official robfig/cron changelog, Watermill troubleshooting |

**Overall confidence:** HIGH

### Gaps to Address

- **Worker restart policy:** Should crashed workers auto-restart? Needs design decision during Phase 1 planning
- **Metrics interface:** How to expose worker queue depth, processing time? Defer to Phase 5 or v2.1
- **Distributed cron:** Leader election for multi-instance deployment is out of scope for v2.0—document as limitation
- **EventBus cleanup:** Should EventBus have OnStop to wait for async handlers? Clarify during Phase 4

## Sources

### Primary (HIGH confidence)
- Context7: `/robfig/cron` — job wrappers, graceful stop, panic recovery change
- Context7: `/alitto/pond` — context-aware API, StopAndWait pattern
- Context7: `/uber-go/fx` — lifecycle hooks, "must not block" semantics
- Context7: `/threedotslabs/watermill` — router patterns, CloseTimeout, deadlock troubleshooting
- Context7: `/jilio/ebu` — type-safe generics eventbus API

### Secondary (MEDIUM confidence)
- GitHub: gammazero/workerpool — worker pool patterns, Submit/StopWait
- GitHub: panjf2000/ants — evaluated but rejected for API mismatch

### Tertiary (LOW confidence)
- Worker restart policies — needs design decision, no clear consensus

---
*Research completed: 2026-01-27*
*Ready for roadmap: yes*
