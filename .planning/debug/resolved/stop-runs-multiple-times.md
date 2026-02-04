---
status: verifying
trigger: "grpc-gateway example's stop function runs multiple times when pressing Ctrl+C, and help text prints after shutdown"
created: 2026-02-03T00:00:00Z
updated: 2026-02-03T00:00:00Z
---

## Current Focus

hypothesis: Stop() is called TWICE - once by waitForShutdownSignal() when Ctrl+C is pressed, and once by PersistentPostRunE after RunE returns
test: Verify the call chain and add idempotency check to Stop()
expecting: Adding a stopped flag will prevent duplicate shutdown
next_action: Verify hypothesis by tracing the exact flow, then implement fix

## Symptoms

expected: Clean shutdown with one stop log - server stops gracefully with a single stop message
actual: Stop function runs 2+ times, error messages appear, help text possibly prints after shutdown
errors: Yes, there are errors (debugger to discover what they are)
reproduction: `go run ./examples/grpc-gateway serve`, then press Ctrl+C to stop
started: Never worked correctly - this has always been broken

## Eliminated

## Evidence

- timestamp: 2026-02-03T00:01:00Z
  checked: examples/grpc-gateway/main.go
  found: Uses app.WithCobra(serveCmd) to attach Cobra integration
  implication: Need to investigate WithCobra implementation for signal handling

- timestamp: 2026-02-03T00:02:00Z
  checked: cobra.go WithCobra() implementation
  found: |
    1. PersistentPreRunE calls bootstrap() -> Build() + Start(), sets running=true
    2. RunE (injected if not set) calls waitForShutdownSignal()
    3. PersistentPostRunE calls Stop() unconditionally
  implication: When Ctrl+C is pressed, both waitForShutdownSignal AND PersistentPostRunE call Stop()

- timestamp: 2026-02-03T00:03:00Z
  checked: app.go waitForShutdownSignal() and handleSignalShutdown()
  found: |
    - waitForShutdownSignal sets up signal.Notify for SIGINT/SIGTERM
    - On signal, calls handleSignalShutdown() which calls a.Stop(shutdownCtx)
    - Returns the error from Stop()
    - RunE returns, then Cobra executes PersistentPostRunE
  implication: Stop() is definitely called twice - once from signal handler, once from post-run hook

- timestamp: 2026-02-03T00:04:00Z
  checked: app.go Stop() function (line 854-933)
  found: |
    - No idempotency guard - Stop() can be called multiple times
    - Only guards stopCh close with select/default pattern (line 920-925)
    - Logs "stopping workers" on each call
    - Computes shutdown order on each call
    - running flag is only set to false in PersistentPostRunE (cobra.go:103-105), AFTER Stop() completes
  implication: This is the root cause - Stop() needs idempotency protection

## Resolution

root_cause: |
  Stop() is called twice when using WithCobra integration:
  1. First call: waitForShutdownSignal() receives SIGINT → handleSignalShutdown() → a.Stop()
  2. Second call: After RunE returns, Cobra executes PersistentPostRunE → a.Stop()
  
  Stop() lacks idempotency protection, causing all services to stop twice. On the second stop,
  the Gateway fails trying to close an already-closed gRPC connection. The error propagates up
  and causes Cobra to print usage text.

fix: Added sync.Once based idempotency to Stop() - new stopOnce and stopErr fields, Stop() delegates to doStop() via sync.Once
verification: |
  1. Manual test: "stopping workers" now appears only once
  2. All services stop cleanly without errors
  3. No Cobra help text appears after shutdown
  4. All tests pass (go test -race ./...)
files_changed:
  - app.go
