---
id: 01-interface-detection.md
wave: 1
depends_on: []
files_modified:
  - di/service.go
  - di/lazy_singleton.go
  - di/eager_singleton.go
  - di/lifecycle_test.go
autonomous: true
---

# Plan 01: Interface Auto-Detection (Detection Logic)

**Goal:** Ensure the DI container recognizes services that implement `Starter` or `Stopper` interfaces as having lifecycle methods, even without explicit registration.

## Context
Currently, `HasLifecycle()` in `baseService` only returns true if explicit hooks (`OnStart`, `OnStop`) have been registered. This causes the App to ignore services that only implement the interfaces. We need to override `HasLifecycle()` in the generic service wrappers (`lazySingleton[T]`, `eagerSingleton[T]`) to check the type `T`.

## Tasks

<task>
  <description>Create lifecycle interface test suite</description>
  <instructions>
    Create `di/lifecycle_test.go`.
    Add test cases that define:
    1. A struct implementing `Starter` (value receiver).
    2. A struct implementing `Starter` (pointer receiver).
    3. A struct implementing `Stopper`.
    4. A struct implementing neither.
    
    Verify that `HasLifecycle()` returns true for 1-3 and false for 4.
  </instructions>
  <files>
    <file>di/lifecycle_test.go</file>
  </files>
</task>

<task>
  <description>Implement HasLifecycle for LazySingleton</description>
  <instructions>
    In `di/lazy_singleton.go`:
    1. Add `HasLifecycle() bool` method to `lazySingleton[T]`.
    2. Logic:
       - Return true if `s.baseService.HasLifecycle()` is true.
       - Use reflection or type assertion on `new(T)` and `*new(T)` (effectively `T` and `*T`) to check for `Starter` and `Stopper` implementation.
       - Note: For `T` (value type), checking `new(T)` gives `*T`. Checking `*new(T)` gives `T`.
       - Go pattern: `var z T`; `_, ok := any(z).(Starter)` (for value receiver check).
       - Go pattern: `_, ok := any(new(T)).(Starter)` (for pointer receiver check).
  </instructions>
  <files>
    <file>di/lazy_singleton.go</file>
  </files>
</task>

<task>
  <description>Implement HasLifecycle for EagerSingleton</description>
  <instructions>
    In `di/eager_singleton.go`:
    1. Add `HasLifecycle() bool` method to `eagerSingleton[T]`.
    2. Use the same logic as LazySingleton.
  </instructions>
  <files>
    <file>di/eager_singleton.go</file>
  </files>
</task>

## Verification
- Run `go test ./di/...`
- Ensure `HasLifecycle` correctly identifies interface implementors.
