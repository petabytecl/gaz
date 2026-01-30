# Phase 23: Foundation & Style Guide - Research

**Researched:** 2026-01-29
**Domain:** Go API naming conventions and style guide documentation
**Confidence:** HIGH

## Summary

This research investigates Go API naming conventions to document in STYLE.md. The codebase already follows consistent patterns that align with standard Go practices from Effective Go, Google's Go Style Guide, and Uber's Go Style Guide.

Key findings: The gaz codebase has established patterns for constructors (`New*`), errors (`Err*` prefix with `pkg: message` format), and module factories (`Module(c *gaz.Container) error`). These existing patterns should be codified as-is since they match Go community standards.

**Primary recommendation:** Document existing gaz patterns with strict MUST language, using real gaz code examples where clear and providing rationale for each convention.

## Standard Stack

This phase requires no external dependencies. The deliverable is a documentation file (STYLE.md).

### Core
| Tool | Purpose | Why Standard |
|------|---------|--------------|
| Markdown | Documentation format | Universal, renders in GitHub, IDE-supported |

### Supporting
No supporting tools needed for documentation.

### Alternatives Considered
| Instead of | Could Use | Tradeoff |
|------------|-----------|----------|
| Single STYLE.md | Multiple docs | Single file easier to navigate; decision locked in CONTEXT.md |

## Architecture Patterns

### Document Structure (from CONTEXT.md)
```
STYLE.md (at repository root)
├── [Header-based navigation only]
├── Naming Conventions
│   ├── Package Names
│   ├── Type Names
│   └── Variable Names
├── Constructor Patterns
│   ├── New*() Pattern
│   ├── Builder Pattern
│   └── Fluent API Pattern
├── Error Conventions
│   ├── Error Variable Naming
│   ├── Error Message Format
│   └── Error Wrapping
└── Module Patterns
    └── Module Factory Functions
```

### Pattern: RFC 2119 Language
**What:** Use MUST/SHOULD/MAY keywords for prescription levels
**When to use:** All rules should use MUST (per CONTEXT.md decision for strict language)
**Example:**
```markdown
### Error Variable Naming

Error variables MUST use the `Err` prefix followed by a descriptive name.
```

### Anti-Patterns to Avoid
- **Vague prescription:** "Consider using X" — be definitive with MUST
- **Missing rationale:** Rules without "why" lead to pushback
- **Examples without context:** Show both good AND bad patterns

## Don't Hand-Roll

This phase is documentation-only. No code to hand-roll.

## Common Pitfalls

### Pitfall 1: Inconsistent Examples
**What goes wrong:** Style guide shows patterns not actually used in codebase
**Why it happens:** Copying external examples without checking existing code
**How to avoid:** Extract all examples from actual gaz code
**Warning signs:** Example code that doesn't compile or differs from codebase
**Automatable:** No

### Pitfall 2: Over-Specification
**What goes wrong:** Documenting rules the codebase doesn't actually need
**Why it happens:** Copying full style guides instead of focused conventions
**How to avoid:** Focus on the four areas in success criteria (naming, constructors, errors, modules)
**Warning signs:** Document grows beyond ~200-300 lines
**Automatable:** No

### Pitfall 3: Missing Exception Process
**What goes wrong:** Contributors blocked when legitimate exceptions arise
**Why it happens:** Strict MUST language without escape valve
**How to avoid:** Include documented exception process (per CONTEXT.md decision)
**Warning signs:** Developers asking "what if I need to..." questions
**Automatable:** No

### Pitfall 4: Stale Examples
**What goes wrong:** Examples reference code that no longer exists
**Why it happens:** Style guide not updated when code changes
**How to avoid:** Use simplified examples where codebase might change; mark real examples for manual verification
**Warning signs:** Links to code that don't exist
**Automatable:** Partially (broken link checker)

## Code Examples

### Existing Constructor Pattern in gaz
```go
// Source: gaz/health/manager.go
// NewManager creates a new Health Manager.
func NewManager() *Manager {
    return &Manager{}
}

// Source: gaz/health/server.go
// NewManagementServer creates a new ManagementServer.
func NewManagementServer(
    config Config,
    manager *Manager,
    shutdownCheck *ShutdownCheck,
) *ManagementServer {
    // ...
    return &ManagementServer{...}
}
```

### Existing Error Pattern in gaz
```go
// Source: gaz/di/errors.go
var (
    // ErrNotFound is returned when a requested service is not registered.
    ErrNotFound = errors.New("di: service not found")

    // ErrCycle is returned when a circular dependency is detected.
    ErrCycle = errors.New("di: circular dependency detected")

    // ErrDuplicate is returned when attempting to register an existing service.
    ErrDuplicate = errors.New("di: service already registered")
)

// Source: gaz/worker/errors.go
var (
    // ErrCircuitBreakerTripped indicates max restarts exceeded.
    ErrCircuitBreakerTripped = errors.New("worker: circuit breaker tripped, max restarts exceeded")
)
```

### Existing Module Pattern in gaz
```go
// Source: gaz/health/module.go
// Module registers the health module components.
func Module(c *gaz.Container) error {
    if err := gaz.For[*ShutdownCheck](c).
        ProviderFunc(func(_ *gaz.Container) *ShutdownCheck {
            return NewShutdownCheck()
        }); err != nil {
        return fmt.Errorf("register shutdown check: %w", err)
    }
    // ...
    return nil
}
```

### Existing Builder Pattern in gaz
```go
// Source: gaz/service/builder.go
// New returns a new service Builder.
func New() *Builder {
    return &Builder{}
}

func (b *Builder) WithCmd(cmd *cobra.Command) *Builder {
    b.cmd = cmd
    return b
}

func (b *Builder) Build() (*gaz.App, error) {
    // ...
}
```

### Existing ModuleBuilder Pattern in gaz
```go
// Source: gaz/module_builder.go
// NewModule creates a new ModuleBuilder with the given name.
func NewModule(name string) *ModuleBuilder {
    return &ModuleBuilder{name: name}
}

func (b *ModuleBuilder) Provide(fns ...func(*Container) error) *ModuleBuilder {
    b.providers = append(b.providers, fns...)
    return b
}

func (b *ModuleBuilder) Build() Module {
    return &builtModule{...}
}
```

## Documented Conventions to Include

Based on codebase analysis and Go style guides, the following conventions should be documented:

### 1. Constructor Naming (HIGH confidence)
| Pattern | When to Use | gaz Example |
|---------|-------------|-------------|
| `NewX()` | Returns single type `*X` | `NewManager()`, `NewShutdownCheck()` |
| `New()` | Package exports only one type | `config.New()`, `service.New()` |
| `NewXWithY()` | Variant constructor | `config.NewWithBackend()` |
| Builder pattern | Many optional config | `ModuleBuilder`, `service.Builder` |

**Source:** Effective Go (constructors section), Google Go Style Guide, existing gaz code

### 2. Error Naming (HIGH confidence)
| Pattern | Format | gaz Example |
|---------|--------|-------------|
| Sentinel errors | `ErrXxx` | `ErrNotFound`, `ErrCycle`, `ErrDuplicate` |
| Error message | `pkg: description` | `"di: service not found"` |
| Error types | `XxxError` | (not currently used in gaz) |

**Source:** Uber Go Style Guide (Error Naming section), existing gaz/di/errors.go, gaz/worker/errors.go

### 3. Module Factory Pattern (HIGH confidence)
| Pattern | Signature | gaz Example |
|---------|-----------|-------------|
| Module function | `Module(c *gaz.Container) error` | `health.Module` |
| Error wrapping | `fmt.Errorf("context: %w", err)` | Throughout module |

**Source:** gaz/health/module.go, established gaz pattern

### 4. Package Documentation (HIGH confidence)
| Element | Convention | gaz Example |
|---------|------------|-------------|
| Location | `doc.go` file | All packages have doc.go |
| First line | `// Package xxx provides...` | Consistent across codebase |
| Sections | Quick Start, Examples, API groups | gaz/doc.go |

**Source:** Effective Go (commentary), Google Go Style Guide, existing gaz doc.go files

## Style Guide Format Conventions

Based on CONTEXT.md decisions:

### Rule Format
```markdown
### [Convention Name]

[Rule] MUST [action].

**Rationale:** [One-liner explaining why]

**Good:**
```go
// Example of correct usage
```

**Bad:**
```go
// Example of incorrect usage
```

[AUTOMATABLE] (if applicable)
```

### Exception Process Format
```markdown
## Exceptions

When a convention cannot be followed:

1. Document the reason in code comment
2. Get approval in code review
3. Link to this style guide with rationale
```

## State of the Art

| Aspect | Go Community Standard | gaz Already Does | Action |
|--------|----------------------|------------------|--------|
| Constructor naming | `New*()` pattern | Yes | Document as-is |
| Error variables | `Err*` prefix | Yes | Document as-is |
| Error messages | lowercase, no punctuation | Yes (`"di: service not found"`) | Document as-is |
| Module factories | No Go standard | `Module(c *Container) error` | Document gaz pattern |
| Builder pattern | Common for complex config | `ModuleBuilder`, `service.Builder` | Document as-is |

**Deprecated/outdated:** None identified — gaz patterns are current.

## Conventions Marked for Automation

Per CONTEXT.md, mark rules that can be enforced with linters:

| Convention | Automatable | Linter |
|------------|-------------|--------|
| Error variable naming (`Err*`) | Yes | Custom golangci-lint rule |
| Error message format (`pkg: msg`) | Partially | Custom rule |
| Constructor naming (`New*`) | No | Semantic, context-dependent |
| Package doc exists | Yes | `golint` / `revive` |
| Doc comment starts with name | Yes | `golint` / `revive` |

## Open Questions

All major questions resolved by CONTEXT.md decisions. No blocking open questions.

## Sources

### Primary (HIGH confidence)
- **Effective Go** (https://go.dev/doc/effective_go) - Naming, constructors, commentary
- **Google Go Style Guide - Decisions** (https://google.github.io/styleguide/go/decisions) - Error naming, naming conventions
- **Uber Go Style Guide** (https://github.com/uber-go/guide) - Error naming, error wrapping, constructors
- **gaz codebase** - Actual patterns extracted from:
  - `gaz/di/errors.go` - Error variable patterns
  - `gaz/worker/errors.go` - Error message format
  - `gaz/health/module.go` - Module factory pattern
  - `gaz/health/manager.go`, `server.go`, `shutdown.go` - Constructor patterns
  - `gaz/module_builder.go` - Builder pattern
  - `gaz/service/builder.go` - Builder pattern
  - `gaz/doc.go`, `di/doc.go`, `config/doc.go` - Package documentation

### Secondary (MEDIUM confidence)
- None needed — primary sources sufficient

### Tertiary (LOW confidence)
- None

## Metadata

**Confidence breakdown:**
- Constructor conventions: HIGH - Verified in Effective Go, Google Style, and gaz code
- Error conventions: HIGH - Verified in Uber Style Guide and gaz code
- Module conventions: HIGH - Extracted from gaz codebase (project-specific)
- Documentation format: HIGH - CONTEXT.md decisions lock format

**Research date:** 2026-01-29
**Valid until:** Indefinite (style conventions are stable)
