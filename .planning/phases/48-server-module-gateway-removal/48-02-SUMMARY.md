---
phase: 48-server-module-gateway-removal
plan: 02
subsystem: server, examples
tags: vanguard, grpc, connect, grpc-web, rest, protobuf, buf

# Dependency graph
requires:
  - phase: 48-server-module-gateway-removal
    provides: "Plan 01 removed server/gateway package, deleted examples/grpc-gateway, cleaned go.mod"
provides:
  - "examples/vanguard/ demonstrating all four protocols on single port"
  - "Updated README.md with vanguard references replacing grpc-gateway"
  - "Connect adapter pattern for bridging gRPC and Connect handler signatures"
affects: []

# Tech tracking
tech-stack:
  added: []
  patterns:
    - "Connect adapter pattern: separate adapter struct bridging gRPC and Connect handler signatures"
    - "Interface-based auto-discovery via di.ResolveAll for grpc.Registrar and connect.Registrar"

key-files:
  created:
    - examples/vanguard/main.go
    - examples/vanguard/service.go
    - examples/vanguard/proto/hello.proto
    - examples/vanguard/proto/hello.pb.go
    - examples/vanguard/proto/hello_grpc.pb.go
    - examples/vanguard/proto/helloconnect/hello.connect.go
    - examples/vanguard/buf.yaml
    - examples/vanguard/buf.gen.yaml
    - examples/vanguard/README.md
  modified:
    - README.md
    - go.mod

key-decisions:
  - "Used adapter pattern for Connect handler: GreeterService keeps gRPC-style SayHello, greeterConnectAdapter wraps it for Connect's generic Request/Response types"
  - "No .As() registration needed: di.ResolveAll uses reflection-based interface matching, so registering concrete *GreeterService is sufficient for grpc.Registrar and connect.Registrar discovery"
  - "buf.gen.yaml uses out: . (not out: proto) to avoid nested proto/proto/ directory with paths=source_relative"

patterns-established:
  - "Connect adapter pattern: when a service implements gRPC-style methods, create a thin adapter struct that wraps the gRPC handler and converts between gRPC and Connect request/response types"

requirements-completed: [SMOD-01, SMOD-02]

# Metrics
duration: 7min
completed: 2026-03-06
---

# Phase 48 Plan 02: Vanguard Example & README Cleanup Summary

**New examples/vanguard/ demonstrating gRPC, Connect, gRPC-Web, and REST on single port via server.NewModule() with Connect adapter pattern**

## Performance

- **Duration:** 7 min
- **Started:** 2026-03-06T23:24:02Z
- **Completed:** 2026-03-06T23:31:05Z
- **Tasks:** 2
- **Files modified:** 13 (10 created, 3 modified)

## Accomplishments
- Created `examples/vanguard/` with complete proto definitions, generated gRPC + Connect stubs, service implementation, and entry point
- Solved gRPC/Connect dual-interface problem with adapter pattern (greeterConnectAdapter bridges signature mismatch)
- Updated project README.md: replaced gRPC-Gateway references with Vanguard
- Verified zero gateway references remain in production code
- All tests pass, `go vet` clean

## Task Commits

Each task was committed atomically:

1. **Task 1: Create Vanguard unified server example** - `4805de3` (feat)
2. **Task 2: Update README, verify clean state, run tests/lint** - `e09550f` (docs)

## Files Created/Modified
- `examples/vanguard/proto/hello.proto` - Protobuf definition with google.api.http annotations for REST transcoding
- `examples/vanguard/proto/hello.pb.go` - Generated protobuf types
- `examples/vanguard/proto/hello_grpc.pb.go` - Generated gRPC server/client stubs
- `examples/vanguard/proto/helloconnect/hello.connect.go` - Generated Connect handler/client stubs
- `examples/vanguard/buf.yaml` - Buf module config with googleapis dependency
- `examples/vanguard/buf.gen.yaml` - Buf code generation config (protoc-gen-go, grpc, connect)
- `examples/vanguard/buf.lock` - Buf dependency lock file
- `examples/vanguard/service.go` - GreeterService with gRPC impl + Connect adapter
- `examples/vanguard/main.go` - Entry point using server.NewModule() and gaz.For registration
- `examples/vanguard/README.md` - Example documentation with protocol test commands
- `README.md` - Updated features and examples sections
- `go.mod` - protobuf/googleapis promoted from indirect to direct

## Decisions Made
- **Connect adapter pattern:** The gRPC `GreeterServer` interface has `SayHello(ctx, *HelloRequest) (*HelloReply, error)` while Connect's `GreeterHandler` has `SayHello(ctx, *connect.Request[HelloRequest]) (*connect.Response[HelloReply], error)`. Since Go doesn't allow same-name methods with different signatures, we use a thin `greeterConnectAdapter` struct that wraps the gRPC service and converts between types in `RegisterConnect()`.
- **No `.As()` needed:** The DI container's `ResolveAll` uses `reflect.Type.AssignableTo` to find all registered types implementing an interface. Registering `*GreeterService` as its concrete type is sufficient — `ResolveAll[grpc.Registrar]` and `ResolveAll[connect.Registrar]` will discover it automatically.
- **`buf.gen.yaml` output path:** Using `out: .` instead of `out: proto` because with `paths=source_relative` and proto files at `proto/hello.proto`, `out: proto` would create a nested `proto/proto/` structure.

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 1 - Bug] Fixed Connect handler signature conflict**
- **Found during:** Task 1 (service.go creation)
- **Issue:** Plan embedded both `UnimplementedGreeterServer` and `UnimplementedGreeterHandler` in the same struct, but both define `SayHello` with incompatible signatures. Go cannot have two methods with the same name and different parameter types on one struct.
- **Fix:** Removed `UnimplementedGreeterHandler` embedding. Created `greeterConnectAdapter` struct that wraps `*GreeterService` and implements `helloconnect.GreeterHandler` by delegating to the gRPC-style `SayHello` with type conversion.
- **Files modified:** `examples/vanguard/service.go`
- **Verification:** `go build ./examples/vanguard/...` succeeds, `go vet` clean
- **Committed in:** `4805de3`

**2. [Rule 3 - Blocking] Skipped examples/grpc-gateway deletion (already done)**
- **Found during:** Task 1 pre-check
- **Issue:** Plan specified deleting `examples/grpc-gateway/` but it was already deleted in Plan 01 as a Rule 3 deviation
- **Fix:** Skipped deletion step, proceeded to create `examples/vanguard/`
- **Files modified:** None
- **Committed in:** N/A

---

**Total deviations:** 2 (1 bug fix, 1 already-handled)
**Impact on plan:** Adapter pattern is the correct solution for dual-interface services. No scope creep.

## Issues Encountered
- Pre-existing lint errors in `server/connect/interceptors.go` (7 issues: stutter, nonamedreturns, wrapcheck, perfsprint). Documented in deferred-items.md — unrelated to this plan's changes.

## User Setup Required
None - no external service configuration required.

## Next Phase Readiness
- Phase 48 is complete: server/gateway removed, vanguard example created, README updated
- All four protocols (gRPC, Connect, gRPC-Web, REST) demonstrated in a single example
- Pre-existing lint issues in server/connect should be addressed in a future cleanup

## Self-Check: PASSED

- All 10 created files verified present
- Commit `4805de3` (Task 1) verified in git log
- Commit `e09550f` (Task 2) verified in git log

---
*Phase: 48-server-module-gateway-removal*
*Completed: 2026-03-06*
