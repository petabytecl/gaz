# Plan: Interface Auto-Detection

**Phase:** 19
**Wave:** 1
**Depends On:** None
**Files Modified:**
- `di/service.go`
- `di/singleton.go`
- `di/lifecycle_test.go` (new)

## Context
Currently, `gaz` services only execute `OnStart`/`OnStop` if they are explicitly registered via `.OnStart(...)` or `.OnStop(...)` hooks. The `HasLifecycle()` method returns false otherwise, causing the App to skip these services during startup/shutdown.

We need to enable "convention over configuration" where implementing `Starter` (`OnStart(context.Context) error`) or `Stopper` (`OnStop(context.Context) error`) is sufficient to participate in the lifecycle.

## Goals
1.  Modify `HasLifecycle()` to return `true` if the service type `T` or `*T` implements lifecycle interfaces.
2.  Update `Start()`/`Stop()` execution flow to invoke these interface methods on the correct receiver (value or pointer).
3.  Ensure explicit hooks take precedence over interface implementations (if a hook is registered, the interface method is ignored).

## Tasks

<task>
<id>1</id>
<description>Create lifecycle reproduction test</description>
<steps>
    <step>Create `di/lifecycle_test.go`</step>
    <step>Define a `struct` service with `OnStart` (pointer receiver)</step>
    <step>Define a `struct` service with `OnStart` (value receiver)</step>
    <step>Register them in a container without explicit hooks</step>
    <step>Assert that `HasLifecycle()` currently returns false (or start fails/does nothing)</step>
</steps>
</task>

<task>
<id>2</id>
<description>Implement Interface Detection in HasLifecycle</description>
<steps>
    <step>Modify `lazySingleton[T].HasLifecycle()` and `eagerSingleton[T].HasLifecycle()`</step>
    <step>Add logic to check if `T` implements `Starter` or `Stopper`</step>
    <step>Add logic to check if `*T` implements `Starter` or `Stopper` (using `new(T)` check)</step>
    <step>Keep existing check: `if s.baseService.HasLifecycle() { return true }` (explicit hooks)</step>
</steps>
</task>

<task>
<id>3</id>
<description>Update Execution Logic for Pointer Receivers</description>
<steps>
    <step>Modify `lazySingleton[T].Start()` and `eagerSingleton[T].Start()`</step>
    <step>Before calling `runStartLifecycle`, check if `s.instance` implements `Starter`</step>
    <step>If not, check if `&s.instance` implements `Starter`</step>
    <step>Pass the correct subject (value or pointer) to `runStartLifecycle`</step>
    <step>Repeat for `Stop()`</step>
</steps>
</task>

<task>
<id>4</id>
<description>Enforce Precedence in BaseService</description>
<steps>
    <step>Modify `baseService.runStartLifecycle`</step>
    <step>Verify logic: If `len(s.startHooks) > 0`, run hooks and RETURN (do not run interface method)</step>
    <step>If no hooks, cast `instance` to `Starter` and call `OnStart`</step>
    <step>Repeat for `Stop()`</step>
</steps>
</task>

<task>
<id>5</id>
<description>Verify Lifecycle Behavior</description>
<steps>
    <step>Run `di/lifecycle_test.go`</step>
    <step>Verify `OnStart` is called for pointer receivers</step>
    <step>Verify `OnStart` is called for value receivers</step>
    <step>Verify explicit hooks override interface methods (create a test case for this)</step>
</steps>
</task>

## Verification Criteria
- [ ] `HasLifecycle()` returns true for a struct with `func (*S) OnStart(ctx)`
- [ ] `HasLifecycle()` returns true for a struct with `func (S) OnStart(ctx)`
- [ ] `App.Start()` calls `OnStart` on implicitly detected services
- [ ] Explicit `.OnStart` hook prevents `Starter.OnStart` from running on the same service
