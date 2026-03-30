---
phase: 48-server-module-gateway-removal
plan: 01
subsystem: server
tags: [vanguard, grpc, gateway-removal, di, module-composition]

# Dependency graph
requires:
  - phase: 47-vanguard-server-wiring
    provides: Vanguard server module, Connect interceptors, transport middleware
provides:
  - Unified server module composing gRPC + Vanguard with SkipListener auto-set
  - Gateway package fully removed from codebase
  - Clean linter config without gateway references
affects: [48-02-examples-docs-cleanup]

# Tech tracking
tech-stack:
  added: []
  patterns: [forceSkipListener config override via Replace() in parent module]

key-files:
  created: []
  modified:
    - server/module.go
    - server/module_test.go
    - server/doc.go
    - server/otel/doc.go
    - .golangci.yml
    - go.mod
    - go.sum

key-decisions:
  - "Used Replace() provider to override gRPC Config with SkipListener=true in parent server module"
  - "grpc-gateway/v2 remains as indirect dependency (pulled by OTEL exporter) — cannot be removed without dropping OTEL"

patterns-established:
  - "Parent module config override: Use Replace().Provider() to mutate child module config in composite modules"

requirements-completed: [SMOD-01, SMOD-02, SMOD-03]

# Metrics
duration: 5min
completed: 2026-03-06
---

# Phase 48 Plan 01: Server Module Gateway Removal Summary

**Unified server module now bundles gRPC + Vanguard with automatic SkipListener, gateway package fully deleted**

## Performance

- **Duration:** 5 min
- **Started:** 2026-03-06T23:15:17Z
- **Completed:** 2026-03-06T23:20:26Z
- **Tasks:** 2
- **Files modified:** 27 (including 24 deleted)

## Accomplishments
- server.NewModule() now composes grpc.NewModule() + vanguard.NewModule() with SkipListener=true auto-set
- server/gateway/ package fully deleted (13 files)
- examples/grpc-gateway/ deleted (blocking go mod tidy)
- .golangci.yml cleaned of all gateway references (ireturn paths, depguard allow lists)
- All server tests pass, entire project compiles cleanly

## Task Commits

Each task was committed atomically:

1. **Task 1: Update server module to bundle Vanguard with SkipListener auto-set** - `f97309d` (feat)
2. **Task 2: Delete gateway package and clean linter config and dependencies** - `48ba820` (chore)

## Files Created/Modified
- `server/module.go` - Replaced gateway with vanguard, added forceSkipListener provider
- `server/module_test.go` - Asserts vanguard.Server registration and SkipListener=true
- `server/doc.go` - Rewritten to reference Vanguard architecture exclusively
- `server/otel/doc.go` - Updated gateway reference to vanguard
- `.golangci.yml` - Removed gateway from ireturn exclusions and depguard allow lists
- `go.mod` / `go.sum` - Cleaned via go mod tidy
- `server/gateway/` - Deleted (13 files)
- `examples/grpc-gateway/` - Deleted (11 files, deviation Rule 3)

## Decisions Made
- **SkipListener implementation:** Used `gaz.For[grpc.Config](c).Replace().Provider(...)` to override the gRPC config in the parent server module's Provide() function. This runs after child modules are applied, ensuring the gRPC config provider is already registered before being replaced. The replacement provider re-creates the config from defaults, applies flag/config values, forces `SkipListener=true`, then validates.
- **grpc-gateway/v2 as indirect:** The dependency remains in go.mod as `// indirect` because `go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc` transitively depends on it. This is correct Go module behavior — no code in gaz imports it directly.

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 3 - Blocking] Deleted examples/grpc-gateway/ to unblock go mod tidy**
- **Found during:** Task 2 (go mod tidy)
- **Issue:** examples/grpc-gateway/ imports server/gateway and grpc-gateway/v2, preventing go mod tidy from succeeding after server/gateway deletion
- **Fix:** Deleted examples/grpc-gateway/ (originally planned for Plan 02)
- **Files modified:** examples/grpc-gateway/ (11 files deleted)
- **Verification:** go mod tidy succeeds, go build ./... succeeds
- **Committed in:** 48ba820 (Task 2 commit)

**2. [Rule 1 - Bug] Fixed stale gateway reference in server/otel/doc.go**
- **Found during:** Task 2 (gateway reference audit)
- **Issue:** server/otel/doc.go referenced "grpc, gateway" instead of "grpc, vanguard"
- **Fix:** Updated doc comment to reference vanguard
- **Files modified:** server/otel/doc.go
- **Verification:** rg "gateway" --glob '*.go' returns zero results
- **Committed in:** 48ba820 (Task 2 commit)

---

**Total deviations:** 2 auto-fixed (1 blocking, 1 bug)
**Impact on plan:** Both essential — examples/grpc-gateway blocked go mod tidy, otel doc reference was stale. No scope creep.

## Issues Encountered
- grpc-gateway/v2 remains as indirect dependency in go.mod (pulled by OTEL exporter). This is expected Go module behavior and cannot be changed without removing OTEL support. Direct imports are confirmed removed.
- Pre-existing lint issues in server/connect/interceptors.go (7 issues: nonamedreturns, perfsprint, revive, wrapcheck) — NOT caused by this plan's changes, out of scope.

## User Setup Required

None - no external service configuration required.

## Next Phase Readiness
- Ready for 48-02 (examples and docs cleanup)
- Note: examples/grpc-gateway/ already deleted in this plan (was originally planned for 48-02)
- server/http package is untouched and continues working independently (SMOD-03 satisfied)

---
*Phase: 48-server-module-gateway-removal*
*Completed: 2026-03-06*
