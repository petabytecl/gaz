---
phase: 29-documentation-examples
plan: 04
subsystem: documentation
tags: [examples, workers, microservice, eventbus, health, lifecycle]

# Dependency graph
requires:
  - phase: 28-testing-infrastructure
    provides: Testing infrastructure and patterns for verifying examples

provides:
  - Background workers tutorial example app
  - Microservice tutorial example app with health, workers, and eventbus

affects: [29-05-documentation-finalization]

# Tech tracking
tech-stack:
  added: []
  patterns: [worker.Worker interface pattern, eventbus pub/sub pattern, health module integration]

key-files:
  created:
    - examples/background-workers/main.go
    - examples/background-workers/README.md
    - examples/microservice/main.go
    - examples/microservice/README.md
    - examples/microservice/config.yaml
  modified: []

key-decisions:
  - "Background workers example uses two concurrent workers (EmailWorker, NotificationWorker) to demonstrate multi-worker pattern"
  - "Microservice example uses event-driven architecture with OrderCreatedEvent and OrderProcessedEvent"
  - "Health module configured on port 9090 to avoid conflict with app server"

patterns-established:
  - "Worker.OnStart must be non-blocking - spawn internal goroutines"
  - "Worker.OnStop should wait for goroutines to exit gracefully"
  - "Eventbus subscribers use typed handlers with context awareness"

# Metrics
duration: 4min
completed: 2026-02-01
---

# Phase 29 Plan 04: Tutorial Example Apps Summary

**Two runnable tutorial apps demonstrating v3 patterns: background workers (179 lines) and microservice with health/workers/eventbus (292 lines)**

## Performance

- **Duration:** 4 min
- **Started:** 2026-02-01T03:58:36Z
- **Completed:** 2026-02-01T04:02:03Z
- **Tasks:** 2
- **Files created:** 5

## Accomplishments

- Created `examples/background-workers/` demonstrating worker.Worker interface implementation
- Created `examples/microservice/` demonstrating full microservice with health, workers, and eventbus
- Both examples compile and build successfully
- Examples use v3 patterns exclusively (no deprecated APIs)
- Well-commented code explains each section's purpose

## Task Commits

Each task was committed atomically:

1. **Task 1: Create background-workers example** - `21bde16` (feat)
2. **Task 2: Create microservice example** - `1fcd29b` (feat)

## Files Created

- `examples/background-workers/main.go` - Background worker tutorial (179 lines)
- `examples/background-workers/README.md` - Worker example documentation
- `examples/microservice/main.go` - Microservice tutorial (292 lines)
- `examples/microservice/README.md` - Microservice example documentation
- `examples/microservice/config.yaml` - Example configuration file

## Decisions Made

1. **Multi-worker pattern:** Background workers example uses two distinct worker types (EmailWorker, NotificationWorker) to demonstrate concurrent worker management
2. **Event-driven microservice:** Microservice example uses OrderCreatedEvent/OrderProcessedEvent to demonstrate eventbus pub/sub patterns
3. **Health port separation:** Health endpoints on port 9090 to avoid conflict with potential application server

## Deviations from Plan

None - plan executed exactly as written.

## Issues Encountered

None

## Next Phase Readiness

- Both example apps are complete and verified
- Ready for Phase 29 Plan 05: Documentation finalization
- Examples demonstrate all major v3 patterns:
  - worker.Worker interface with OnStart/OnStop
  - Eventbus pub/sub with typed events
  - Health module integration
  - Eager registration for auto-start

---

*Phase: 29-documentation-examples*
*Completed: 2026-02-01*
