# Phase 10: Documentation & Examples - Research

**Researched:** 2026-01-27
**Domain:** Go documentation (godoc, markdown, testable examples)
**Confidence:** HIGH

## Summary

This phase focuses on creating comprehensive documentation for the `gaz` DI framework. Research covers Go's standard documentation approach: godoc comments for API reference (published to pkg.go.dev), markdown files for guides and tutorials, and testable examples that serve as both documentation and tests.

Go 1.19+ introduced improved doc comment syntax with explicit headings (`# Heading`), documentation links (`[Type]` or `[pkg.Type]`), and cleaner list formatting. The library currently has good inline doc comments but lacks: a README, dedicated documentation files, testable godoc examples, and standalone example applications.

**Primary recommendation:** Use godoc for API reference (with testable examples), markdown in `/docs` for guides, and `/examples` for complete runnable applications. All examples should compile and be tested via `go test`.

## Standard Stack

The established tools for Go library documentation:

### Core
| Tool | Version | Purpose | Why Standard |
|------|---------|---------|--------------|
| godoc | Go 1.25+ | API reference generation | Built into Go, publishes to pkg.go.dev automatically |
| `go doc` | Go 1.25+ | CLI doc viewer | Standard tooling, immediate feedback |
| pkg.go.dev | N/A | Public documentation hosting | Official Go module documentation |
| Markdown | N/A | Guide/tutorial documentation | GitHub renders automatically, familiar format |

### Supporting
| Tool | Purpose | When to Use |
|------|---------|-------------|
| `go test` | Run testable examples | All examples should pass `go test` |
| `gofmt` | Format doc comments | Run before commit to ensure canonical formatting |
| Go playground | Interactive examples | Reference from docs for simple snippets |

### Not Needed
| Tool | Why Not |
|------|---------|
| External doc generators (gomarkdoc, etc.) | Godoc is sufficient, adds complexity |
| Documentation websites (Hugo, Docusaurus) | Overkill for library; markdown + pkg.go.dev sufficient |
| Swagger/OpenAPI | Not an HTTP API library |

**No installation needed** - standard Go tooling suffices.

## Architecture Patterns

### Recommended Project Structure
```
gaz/
├── README.md               # Entry point: install, quickstart, core concepts
├── docs/
│   ├── getting-started.md  # Step-by-step first app
│   ├── concepts.md         # DI fundamentals, scopes, lifecycle
│   ├── configuration.md    # Config loading, env vars, profiles
│   ├── validation.md       # Struct validation, tags
│   └── advanced.md         # Modules, testing, Cobra integration
├── examples/
│   ├── basic/              # Minimal working app
│   │   ├── main.go
│   │   └── README.md
│   ├── http-server/        # HTTP server with graceful shutdown
│   │   ├── main.go
│   │   └── README.md
│   ├── config-loading/     # Config files, env vars, validation
│   │   ├── main.go
│   │   ├── config.yaml
│   │   └── README.md
│   ├── lifecycle/          # Services with OnStart/OnStop
│   │   ├── main.go
│   │   └── README.md
│   ├── modules/            # Organizing providers into modules
│   │   ├── main.go
│   │   └── README.md
│   └── cobra-cli/          # CLI app with Cobra integration
│       ├── main.go
│       └── README.md
├── example_test.go         # Testable godoc examples (package level)
├── app_example_test.go     # App-specific examples
└── ...                     # Existing source files
```

### Pattern 1: Testable Godoc Examples
**What:** Example functions in `*_test.go` files that demonstrate API usage and are executed by `go test`
**When to use:** For every major exported function, type, and pattern
**Example:**
```go
// Source: https://go.dev/blog/examples
// File: example_test.go

package gaz_test

import (
    "fmt"
    "github.com/petabytecl/gaz"
)

func ExampleNew() {
    app := gaz.New()
    app.ProvideSingleton(func(c *gaz.Container) (*MyService, error) {
        return &MyService{Name: "example"}, nil
    })
    if err := app.Build(); err != nil {
        fmt.Println("error:", err)
        return
    }
    svc, _ := gaz.Resolve[*MyService](app.Container())
    fmt.Println(svc.Name)
    // Output: example
}

type MyService struct {
    Name string
}
```

### Pattern 2: Whole-File Examples
**What:** Complete `_test.go` files demonstrating complex scenarios requiring type definitions
**When to use:** When example needs custom types, interfaces, or multiple functions
**Example:**
```go
// Source: https://go.dev/blog/examples
// File: example_lifecycle_test.go

package gaz_test

import (
    "context"
    "fmt"
    "github.com/petabytecl/gaz"
)

// Server demonstrates a service with lifecycle hooks.
type Server struct {
    running bool
}

func (s *Server) OnStart(ctx context.Context) error {
    s.running = true
    return nil
}

func (s *Server) OnStop(ctx context.Context) error {
    s.running = false
    return nil
}

func Example_lifecycle() {
    app := gaz.New()
    app.ProvideSingleton(func(c *gaz.Container) (*Server, error) {
        return &Server{}, nil
    })
    _ = app.Build()
    fmt.Println("built")
    // Output: built
}
```

### Pattern 3: Standalone Example Applications
**What:** Complete, runnable Go applications in `/examples`
**When to use:** For realistic, production-quality demonstrations
**Structure:**
```
examples/http-server/
├── main.go          # Runnable application
├── handlers.go      # HTTP handlers (if needed)
├── config.yaml      # Example configuration
└── README.md        # What it demonstrates, how to run
```

### Anti-Patterns to Avoid
- **Pseudo-code in docs:** All code in documentation must compile and ideally run
- **Stale examples:** Examples not verified by tests will drift from API
- **Links to external files from godoc:** Inline code snippets in doc comments, not links
- **Complex examples first:** Start simple, build complexity gradually

## Don't Hand-Roll

Problems with existing solutions:

| Problem | Don't Build | Use Instead | Why |
|---------|-------------|-------------|-----|
| API reference | Custom doc generator | godoc + pkg.go.dev | Standard, auto-published, searchable |
| Example verification | Manual testing | Testable examples (`go test`) | Automatic, CI-integrated |
| Code formatting in docs | Manual formatting | `gofmt` on doc comments | Canonical, automatic |
| Doc links | `[text](url)` | `[Type]` or `[pkg.Type]` | Go 1.19+ native, auto-linked |

**Key insight:** Go's built-in documentation tooling is comprehensive. External tools add friction without proportional value for a library project.

## Common Pitfalls

### Pitfall 1: Undocumented Exported Symbols
**What goes wrong:** pkg.go.dev shows warnings, API is unclear
**Why it happens:** Fast iteration, forgetting to add comments
**How to avoid:** Linter check (`golangci-lint` with `revive` or `godot`), pre-commit hook
**Warning signs:** Yellow warnings on pkg.go.dev, users asking basic questions

### Pitfall 2: Examples Without Output Comments
**What goes wrong:** Examples compile but don't run in `go test`
**Why it happens:** Forgetting the `// Output:` comment
**How to avoid:** Always include output comment; if output is non-deterministic, use `// Unordered output:` or omit (but then example won't be tested)
**Warning signs:** Example passes without any assertions

### Pitfall 3: Import Path Mismatches in Examples
**What goes wrong:** Examples in `_test.go` use wrong package name
**Why it happens:** Confusing `package gaz` vs `package gaz_test`
**How to avoid:** 
- Use `package gaz_test` for black-box examples (tests public API)
- Use `package gaz` for white-box examples (tests internals)
**Warning signs:** Import cycles, unexported symbol errors

### Pitfall 4: Hardcoded Values That Change
**What goes wrong:** Examples fail when underlying behavior changes
**Why it happens:** Testing exact output that includes timestamps, goroutine IDs, etc.
**How to avoid:** Control example outputs, use stable test data
**Warning signs:** Flaky example tests

### Pitfall 5: Documentation Drift
**What goes wrong:** Docs describe old API, users confused
**Why it happens:** Code updated, docs forgotten
**How to avoid:** 
1. Testable examples catch API changes (compilation fails)
2. Review docs as part of PR review
3. Include doc updates in definition of done
**Warning signs:** GitHub issues about docs

### Pitfall 6: Doc Comment Formatting Errors
**What goes wrong:** Godoc renders incorrectly (unintended code blocks, broken lists)
**Why it happens:** Indentation creates code blocks, missing blank lines
**How to avoid:** Run `gofmt` (fixes doc comment formatting since Go 1.19), preview with `go doc`
**Warning signs:** Preformatted text where prose expected

## Code Examples

### Testable Example Function (Basic)
```go
// Source: https://go.dev/blog/examples

func ExampleResolve() {
    c := gaz.NewContainer()
    _ = gaz.For[*Config](c).Instance(&Config{Debug: true})
    _ = c.Build()
    
    cfg, err := gaz.Resolve[*Config](c)
    if err != nil {
        fmt.Println("error:", err)
        return
    }
    fmt.Println(cfg.Debug)
    // Output: true
}
```

### Named Example (Multiple Examples for Same Function)
```go
// Source: https://go.dev/blog/examples

func ExampleFor_singleton() {
    // ... singleton example
    // Output: ...
}

func ExampleFor_transient() {
    // ... transient example  
    // Output: ...
}
```

### Doc Comment with Go 1.19+ Headings and Links
```go
// Source: https://go.dev/doc/comment

// Package gaz provides a simple, type-safe dependency injection container
// with lifecycle management for Go applications.
//
// # Quick Start
//
// Create an application, register providers, build, and run:
//
//     app := gaz.New()
//     app.ProvideSingleton(NewDatabase)
//     if err := app.Build(); err != nil {
//         log.Fatal(err)
//     }
//     app.Run(context.Background())
//
// # Service Scopes
//
// Services can be registered with different scopes:
//
//   - Singleton: One instance for container lifetime (default)
//   - Transient: New instance on every resolution
//   - Eager: Singleton instantiated at [App.Build] time
//
// See [For], [App.ProvideSingleton], and [App.ProvideTransient] for details.
package gaz
```

### Doc Links (Go 1.19+)
```go
// Source: https://go.dev/doc/comment

// Resolve retrieves a service of type T from the container.
// Returns [ErrNotFound] if the service is not registered,
// or [ErrCycle] if a circular dependency is detected.
// 
// For named resolution, use the [Named] option.
func Resolve[T any](c *Container, opts ...ResolveOption) (T, error)
```

### README Badge for pkg.go.dev
```markdown
[![Go Reference](https://pkg.go.dev/badge/github.com/petabytecl/gaz.svg)](https://pkg.go.dev/github.com/petabytecl/gaz)
```

## State of the Art

| Old Approach | Current Approach | When Changed | Impact |
|--------------|------------------|--------------|--------|
| Implicit headings in godoc | Explicit `# Heading` syntax | Go 1.19 (Aug 2022) | Clearer doc structure |
| Manual cross-references | `[Type]` and `[pkg.Type]` links | Go 1.19 (Aug 2022) | Auto-linked in pkg.go.dev |
| Unformatted doc comments | `gofmt` reformats doc comments | Go 1.19 (Aug 2022) | Canonical formatting |
| godoc.org | pkg.go.dev | 2020 | Official replacement |

**Current best practices:**
- Use explicit `# Heading` for sections
- Use `[Type]` for documentation links
- All exported symbols need doc comments starting with the symbol name
- Lists use `-` for bullets, `1.` for numbered
- Code blocks indented 1 tab
- Run `gofmt` to auto-format doc comments

## Open Questions

1. **Example naming for App vs Container API**
   - What we know: Library has two APIs (App builder API, Container low-level API)
   - What's unclear: How to organize examples that show both
   - Recommendation: Prioritize App API in examples (it's the primary API), Container examples secondary

2. **Health module documentation**
   - What we know: `health` package exists with its own types
   - What's unclear: Should it have separate examples or be integrated into main examples?
   - Recommendation: Include health module in one comprehensive example (http-server), not standalone

## Sources

### Primary (HIGH confidence)
- https://go.dev/doc/comment - Official Go doc comment syntax reference
- https://go.dev/blog/examples - Testable examples blog post (still accurate)
- pkg.go.dev - Current Go documentation platform

### Secondary (MEDIUM confidence)
- Codebase analysis - Existing doc comments in gaz source files

### Context Used
- CONTEXT.md decisions:
  - README + `/docs` folder structure (locked)
  - Inline code snippets in docs (locked)
  - 5-8 comprehensive examples (locked)
  - Godoc + testable examples for API reference (locked)
  - Go experts who are DI newcomers (audience)
  - Terse, technical writing style (locked)

## Metadata

**Confidence breakdown:**
- Standard stack: HIGH - Official Go tooling, well-documented
- Architecture: HIGH - Standard Go project patterns
- Pitfalls: HIGH - Based on official Go docs and common patterns

**Research date:** 2026-01-27
**Valid until:** 90 days (stable domain, Go doc syntax mature)
