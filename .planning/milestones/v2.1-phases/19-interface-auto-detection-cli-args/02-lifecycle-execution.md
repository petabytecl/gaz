---
id: 02-lifecycle-execution.md
wave: 2
depends_on: [01-interface-detection.md]
files_modified:
  - di/service.go
  - di/lazy_singleton.go
  - di/eager_singleton.go
  - di/lifecycle_test.go
autonomous: true
---

# Plan 02: Lifecycle Execution & Precedence

**Goal:** Ensure `OnStart`/`OnStop` are called on detected services, handling pointer receivers correctly, and respecting the rule that explicit hooks take precedence.

## Context
With detection working (Plan 01), we now need to ensure `Start()` and `Stop()` execution paths:
1. Pass the correct instance (address of instance if receiver is pointer but T is value) to the lifecycle runner.
2. In `baseService`, implement the precedence: Explicit Hook > Interface Method.

## Tasks

<task>
  <description>Update baseService execution logic</description>
  <instructions>
    In `di/service.go`, modify `runStartLifecycle(ctx context.Context, instance any) error`:
    1. Check if `len(s.startHooks) > 0`.
    2. If YES: Run hooks and RETURN. Do NOT check interface.
    3. If NO: Check `instance.(Starter)`. If match, call `OnStart`.
    
    Do the same for `runStopLifecycle` and `Stopper`.
  </instructions>
  <files>
    <file>di/service.go</file>
  </files>
</task>

<task>
  <description>Handle pointer receivers in LazySingleton</description>
  <instructions>
    In `di/lazy_singleton.go`, update `Start(ctx context.Context)`:
    1. Identify the correct instance to pass to `runStartLifecycle`.
    2. `s.instance` is type `T`.
    3. If `T` implements `Starter`, use `s.instance`.
    4. If `T` does NOT implement `Starter` but `*T` does, use `&s.instance`.
    5. Pass the determined object to `s.runStartLifecycle`.
    
    Apply similar logic to `Stop(ctx context.Context)`.
  </instructions>
  <files>
    <file>di/lazy_singleton.go</file>
  </files>
</task>

<task>
  <description>Handle pointer receivers in EagerSingleton</description>
  <instructions>
    In `di/eager_singleton.go`:
    1. Apply the same pointer-handling logic to `Start` and `Stop` as done for LazySingleton.
  </instructions>
  <files>
    <file>di/eager_singleton.go</file>
  </files>
</task>

<task>
  <description>Verify execution and precedence</description>
  <instructions>
    Update `di/lifecycle_test.go`:
    1. Add test: `Starter` (value) runs.
    2. Add test: `Starter` (pointer) runs on value type `T`.
    3. Add test: Explicit `OnStart` runs, implicit `Starter.OnStart` does NOT run (precedence).
  </instructions>
  <files>
    <file>di/lifecycle_test.go</file>
  </files>
</task>

## Verification
- Run `go test ./di/...`
- Confirm all lifecycle permutations execute correctly.
- Confirm precedence rule holds.
