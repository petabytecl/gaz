# Phase 37: Core Discovery - Research

**Researched:** 2026-02-02
**Domain:** Dependency Injection / Service Discovery
**Confidence:** HIGH

## Summary

This phase introduces "Core Discovery" capabilities to the `gaz` container, enabling the resolution of multiple service providers for a single type (`ResolveAll`) and group-based resolution (`ResolveGroup`). This supports auto-discovery patterns where modules register plugins or handlers (e.g., event listeners, API routes) without the consumer knowing about them at compile time.

The standard approach for this in Go DI frameworks (like uber/fx or Google Wire) involves "value groups" or explicit collection types. For `gaz`, we have decided on an implicit collection model: registering multiple providers for the same type simply accumulates them.

**Primary recommendation:** Refactor the internal `services` map to store a list of providers `[]ServiceWrapper` instead of a single instance, and implement `ResolveAll` by scanning for type assignability.

## Standard Stack

### Core
| Library | Version | Purpose | Why Standard |
|---------|---------|---------|--------------|
| `reflect` | stdlib | Type introspection | Required for runtime dependency injection and assignability checks. |

### Supporting
| Library | Version | Purpose | When to Use |
|---------|---------|---------|-------------|
| `sync` | stdlib | Concurrency control | Protecting the modified service map. |

## Architecture Patterns

### Internal Storage Refactoring
The current `map[string]any` (1:1 mapping) is insufficient.

**Recommended Structure:**
Change `Container.services` to effectively store `[]ServiceWrapper`.
```go
// Current
services map[string]any // value is ServiceWrapper

// Proposed
services map[string][]ServiceWrapper
```

### Registration Logic (`di/registration.go`)
- **Current:** Returns `ErrDuplicate` if key exists and `!Replace`.
- **New:**
  - If key exists: Append to the list.
  - `Replace` semantic: Overwrites the *entire list* for that key (or maybe just the specific named instance? simpler to overwrite all).
  - *Decision:* `Replace()` should probably replace the list to maintain test isolation, but "Implicit Collection" suggests we usually want to append. For this phase, if `Replace` is not used, we append.

### Resolution Logic (`di/resolution.go`)

#### 1. Single Resolution (`Resolve[T]`)
Must enforce the "Ambiguity Policy".
- Look up `services[name]`.
- If list is empty: `ErrNotFound`.
- If list has > 1 element: `ErrAmbiguous` (new error).
- If list has 1 element: Resolve and return.

#### 2. List Resolution (`ResolveAll[T]`)
Must find *all* providers that satisfy `T`, regardless of their registration key (name).
- **Algorithm:**
  1. Iterate over all entries in `c.services`.
  2. For each `ServiceWrapper` in each list:
  3. Check if `wrapper.ServiceType()` is assignable to `reflect.TypeOf(T)`.
     - `wrapper.ServiceType().AssignableTo(targetType)` (for structs)
     - `wrapper.ServiceType().Implements(targetType)` (for interfaces)
  4. If match, resolve instance and append to result.
  5. Return `[]T` (empty slice if none found).

#### 3. Group Resolution (`ResolveGroup[T]`)
- Add `groups []string` to `ServiceWrapper`.
- **Algorithm:** Same as `ResolveAll`, but add check: `contains(wrapper.Groups(), groupName)`.

### Tagging Support
- Add `InGroup(name string)` to `RegistrationBuilder`.
- Pass this metadata to `ServiceWrapper` constructors.

## Don't Hand-Roll

| Problem | Don't Build | Use Instead | Why |
|---------|-------------|-------------|-----|
| Type Checking | Custom string matching | `reflect.Type.AssignableTo/Implements` | Go's type system is complex (interfaces, pointers, aliases); `reflect` handles it correctly. |

## Common Pitfalls

### Pitfall 1: Ambiguity in Single Resolve
**What goes wrong:** `Resolve[T]` returns the first registered service when multiple exist.
**Why it happens:** "First-one-wins" logic is common but dangerous for finding bugs.
**How to avoid:** Explicitly check `len(services) > 1` and return an error. Force user to use `ResolveAll` if cardinality > 1.

### Pitfall 2: Hidden Services
**What goes wrong:** `ResolveAll[T]` only looks up `TypeName[T]` and misses named services.
**Why it happens:** Assuming `services[TypeName[T]]` contains all `T`s. Named services (e.g., `Named("primary")`) are stored under "primary", not the type name.
**How to avoid:** `ResolveAll` MUST scan the entire container (or a secondary type index) to find *all* instances of `T`, regardless of registration name.

### Pitfall 3: Lifecycle duplication
**What goes wrong:** If a service is in multiple groups or resolved multiple times via `ResolveAll`, it might be initialized twice.
**How to avoid:** `ServiceWrapper` (Singleton) already handles idempotent `GetInstance`. Ensure we don't wrap the wrapper again.

## Code Examples

### Registration with Groups
```go
// Registration
gaz.For[*Handler](c).InGroup("api").Provider(NewAuthHandler)
gaz.For[*Handler](c).InGroup("api").Provider(NewUserHandler)

// Resolution
handlers := gaz.ResolveGroup[*Handler](c, "api")
for _, h := range handlers {
    h.Register(router)
}
```

### Discovery Pattern (ResolveAll)
```go
// Plugins register themselves
gaz.For[Plugin](c).Provider(NewPluginA)
gaz.For[Plugin](c).Provider(NewPluginB)

// Host discovers them
plugins := gaz.ResolveAll[Plugin](c)
```

## Open Questions

1. **`Replace()` Semantics with Collections**
   - **Recommendation:** `Replace()` clears the existing list for that name and starts a new one. This keeps testing simple (mock replaces real implementation).

2. **Performance of O(N) Scan**
   - **Assessment:** For N < 1000 services, full scan is negligible (~microseconds). If scalability becomes an issue later, we can add a `map[reflect.Type][]string` index. For now, Keep It Simple.

## Metadata

**Confidence breakdown:**
- Standard stack: HIGH - `reflect` is standard.
- Architecture: HIGH - 1:N mapping is standard for this feature.
- Pitfalls: HIGH - Common DI issues.

**Research date:** 2026-02-02
