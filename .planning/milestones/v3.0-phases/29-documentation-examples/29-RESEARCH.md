# Phase 29: Documentation & Examples - Research

**Researched:** 2026-02-01
**Domain:** Go documentation (godoc, README, tutorials, examples)
**Confidence:** HIGH

## Summary

This phase focuses on completing user-facing documentation for gaz v3. The research covers Go documentation best practices: testable examples (godoc `Example` functions with `// Output:` comments), README structure for libraries, `/docs` page organization, and `/examples` directory conventions.

The codebase already has partial documentation infrastructure: `doc.go` files for 7 packages (gaz, di, config, worker, cron, eventbus, gaztest), some example functions in `example_test.go` files, existing `/docs` markdown files, and 7 example applications in `/examples`. The task is to expand coverage to comprehensive v3-pattern documentation across all packages.

Key findings: Go testable examples are the gold standard for living documentation - they appear on pkg.go.dev and are verified by `go test`. The README should follow code-first approach (get running in 2 minutes), and tutorials should be runnable code in `/examples` with minimal prose.

**Primary recommendation:** Create `example_test.go` files with comprehensive `Example` functions for all packages, update README to code-first v3 patterns, and add CI verification for `/examples` apps.

## Standard Stack

The established libraries/tools for this domain:

### Core
| Tool | Purpose | Why Standard |
|------|---------|--------------|
| `go doc` | View package documentation | Built-in Go toolchain |
| `go test` | Verify testable examples | Examples are tests |
| `pkgsite` | Preview pkg.go.dev locally | Official Go tool |
| Markdown | `/docs` and README files | GitHub standard |

### Supporting
| Tool | Purpose | When to Use |
|------|---------|-------------|
| `golangci-lint` (revive) | Check doc comment formatting | Already in project |
| `go vet` | Catch example naming issues | Part of go test |

### Documentation Preview
```bash
# Install pkgsite to preview documentation locally
go install golang.org/x/pkgsite/cmd/pkgsite@latest
pkgsite
# View at http://localhost:8080
```

## Architecture Patterns

### Recommended Documentation Structure
```
gaz/
├── README.md                    # Quick start (2-minute path)
├── CHANGELOG.md                 # Version history (exists)
├── STYLE.md                     # Contributor guide (exists)
├── doc.go                       # Package-level godoc (exists)
├── example_test.go              # Core examples (expand)
├── example_lifecycle_test.go    # Lifecycle examples (exists)
├── example_config_test.go       # Config examples (exists)
├── docs/
│   ├── getting-started.md       # First app walkthrough (update)
│   ├── concepts.md              # DI fundamentals (update)
│   ├── configuration.md         # Config system (update)
│   ├── validation.md            # Struct validation (exists)
│   ├── advanced.md              # Modules, testing (update)
│   ├── troubleshooting.md       # Common mistakes (new)
│   └── api/                     # (optional) per-package docs
│       ├── di.md
│       ├── config.md
│       └── ...
├── examples/
│   ├── basic/                   # Minimal example (exists)
│   ├── http-server/             # Web service (exists)
│   ├── config-loading/          # Config patterns (exists)
│   ├── lifecycle/               # Lifecycle hooks (exists)
│   ├── modules/                 # Module organization (exists)
│   ├── cobra-cli/               # CLI app (exists)
│   ├── background-workers/      # Worker tutorial (new)
│   └── microservice/            # Full service (new)
├── di/
│   ├── doc.go                   # Package doc (exists)
│   └── example_test.go          # DI examples (new)
├── config/
│   ├── doc.go                   # Package doc (exists)
│   └── example_test.go          # Config examples (new)
├── health/
│   ├── doc.go                   # Package doc (new)
│   └── example_test.go          # Health examples (new)
├── worker/
│   ├── doc.go                   # Package doc (exists)
│   └── example_test.go          # Worker examples (new)
├── cron/
│   ├── doc.go                   # Package doc (exists)
│   └── example_test.go          # Cron examples (new)
├── eventbus/
│   ├── doc.go                   # Package doc (exists)
│   └── example_test.go          # Eventbus examples (new)
└── gaztest/
    ├── doc.go                   # Package doc (exists)
    └── example_test.go          # Testing examples (expand)
```

### Pattern 1: Testable Example Function
**What:** Functions named `Example*` with `// Output:` comments
**When to use:** Every exported function/type that can produce deterministic output
**Example:**
```go
// Source: https://go.dev/blog/examples
package gaz_test

import (
    "fmt"
    "github.com/petabytecl/gaz"
)

func ExampleFor_singleton() {
    c := gaz.NewContainer()
    
    counter := &Counter{}
    gaz.For[*Counter](c).Instance(counter)
    
    // Resolve twice - same instance returned
    c1, _ := gaz.Resolve[*Counter](c)
    c2, _ := gaz.Resolve[*Counter](c)
    
    fmt.Println("same instance:", c1 == c2)
    // Output: same instance: true
}
```

### Pattern 2: Example Without Output (Compile-Only)
**What:** Example functions without `// Output:` comment
**When to use:** Lifecycle-heavy APIs, async operations, network calls
**Example:**
```go
// Demonstrates HTTP server with graceful shutdown.
// No Output comment because lifecycle is async.
func ExampleApp_Run() {
    app := gaz.New()
    gaz.For[*Server](app.Container()).Eager().Provider(NewServer)
    app.Build()
    // app.Run() blocks until SIGTERM - can't capture output
}
```

### Pattern 3: Whole-File Example
**What:** Single example with supporting types in dedicated file
**When to use:** Complex examples requiring type definitions
**Example:**
```go
// example_sort_test.go - whole file
package gaz_test

import (
    "context"
    "fmt"
    "github.com/petabytecl/gaz"
)

type MyService struct {
    ready bool
}

func (s *MyService) OnStart(ctx context.Context) error {
    s.ready = true
    fmt.Println("started")
    return nil
}

func Example_serviceLifecycle() {
    app := gaz.New()
    gaz.For[*MyService](app.Container()).Provider(...)
    // ...
    // Output:
    // started
}
```

### Pattern 4: Example Naming Convention
**What:** Consistent naming for example functions
**When to use:** All examples
```go
// Package-level example
func Example()                      // Documents package

// Function examples
func ExampleNew()                   // Documents gaz.New()
func ExampleResolve()               // Documents gaz.Resolve()
func ExampleFor()                   // Documents gaz.For()

// Method examples
func ExampleContainer_Build()       // Documents Container.Build()
func ExampleApp_Run()               // Documents App.Run()

// Multiple examples (suffix with lowercase after underscore)
func ExampleFor_singleton()         // Documents For() for singleton pattern
func ExampleFor_transient()         // Documents For() for transient pattern
func ExampleFor_eager()             // Documents For() for eager pattern
```

### Anti-Patterns to Avoid
- **Panic in examples:** Use `if err != nil { fmt.Println("error:", err); return }` not `log.Fatal` or `panic`
- **External dependencies:** Don't require network, database, or filesystem in examples
- **Non-deterministic output:** Avoid maps (iteration order), goroutines (timing), or timestamps
- **Long setup:** If setup exceeds 10 lines, the API may be too complex
- **Missing imports:** Always show the full import block in complex examples

## Don't Hand-Roll

Problems that look simple but have existing solutions:

| Problem | Don't Build | Use Instead | Why |
|---------|-------------|-------------|-----|
| API documentation | Custom markdown generators | Go's built-in godoc | Integrated with pkg.go.dev, verified by tests |
| Example verification | Manual review | Testable examples with `// Output:` | Automated via `go test` |
| Documentation preview | Push and check | `pkgsite` local server | Instant feedback loop |
| README badges | Manual SVG creation | shields.io URLs | Auto-updated, recognized format |
| Code block highlighting | Custom CSS | GitHub/pkg.go.dev defaults | Automatic language detection |

**Key insight:** Go's documentation toolchain (godoc, pkgsite, testable examples) provides everything needed. Custom solutions fragment the experience and don't appear on pkg.go.dev.

## Common Pitfalls

### Pitfall 1: Examples Without Output Comments
**What goes wrong:** Examples compile but don't run during `go test`, can become stale
**Why it happens:** Developer forgets or thinks output is "obvious"
**How to avoid:** Add `// Output:` comment to ALL examples where output is deterministic
**Warning signs:** Running `go test -v` shows examples as "skipped"

### Pitfall 2: Non-Deterministic Example Output
**What goes wrong:** Examples fail intermittently
**Why it happens:** Maps, goroutines, timestamps produce variable output
**How to avoid:** 
- Use `// Unordered output:` for map iteration
- Skip `// Output:` for async code
- Mock time sources
**Warning signs:** CI failures that pass locally

### Pitfall 3: Example Function Naming Errors
**What goes wrong:** Examples don't appear in documentation
**Why it happens:** Wrong naming pattern (e.g., `ExampleFoo_Bar()` for non-method)
**How to avoid:** Follow exact pattern: `ExampleTypeName_MethodName()` for methods
**Warning signs:** Examples visible in code but missing from `go doc` output

### Pitfall 4: Stale README Examples
**What goes wrong:** README code doesn't compile with current API
**Why it happens:** API changes without README updates
**How to avoid:** 
- Keep README examples minimal (link to testable examples)
- Reference example_test.go patterns
- Add CI step to compile /examples directory
**Warning signs:** User issues about README code not working

### Pitfall 5: Too Much Prose in Tutorials
**What goes wrong:** Tutorials become walls of text that users skip
**Why it happens:** Writer explains every line
**How to avoid:** Code-primary approach - show runnable code, minimal prose for "why"
**Warning signs:** Tutorial files longer than example code files

### Pitfall 6: Missing Package-Level Documentation
**What goes wrong:** Package appears undocumented on pkg.go.dev
**Why it happens:** No `doc.go` file or empty package comment
**How to avoid:** Every package needs `doc.go` with comprehensive package comment
**Warning signs:** Empty "Overview" section on pkg.go.dev

### Pitfall 7: Examples Import Wrong Package
**What goes wrong:** Examples show internal package usage
**Why it happens:** Copy-paste from internal tests
**How to avoid:** Examples must use `_test` package suffix, import public API
**Warning signs:** Examples import `github.com/foo/bar/internal/...`

## Code Examples

Verified patterns from official sources:

### Basic Testable Example
```go
// Source: https://go.dev/blog/examples
package reverse_test

import (
    "fmt"
    "golang.org/x/example/hello/reverse"
)

func ExampleString() {
    fmt.Println(reverse.String("hello"))
    // Output: olleh
}
```

### Example with Setup and Error Handling
```go
// Source: Standard library pattern
func ExampleNew() {
    app := gaz.New()
    err := gaz.For[*MyService](app.Container()).Provider(func(c *gaz.Container) (*MyService, error) {
        return &MyService{Name: "example"}, nil
    })
    if err != nil {
        fmt.Println("error:", err)
        return
    }
    if err := app.Build(); err != nil {
        fmt.Println("error:", err)
        return
    }
    svc, err := gaz.Resolve[*MyService](app.Container())
    if err != nil {
        fmt.Println("error:", err)
        return
    }
    fmt.Println(svc.Name)
    // Output: example
}
```

### Multiple Examples for Same Function
```go
// Source: Go testing package conventions
func ExampleFor_singleton() {
    c := gaz.NewContainer()
    gaz.For[*Counter](c).Instance(&Counter{})
    c1, _ := gaz.Resolve[*Counter](c)
    c2, _ := gaz.Resolve[*Counter](c)
    fmt.Println("same:", c1 == c2)
    // Output: same: true
}

func ExampleFor_transient() {
    c := gaz.NewContainer()
    gaz.For[*Counter](c).Transient().Provider(func(_ *gaz.Container) (*Counter, error) {
        return &Counter{}, nil
    })
    c1, _ := gaz.Resolve[*Counter](c)
    c2, _ := gaz.Resolve[*Counter](c)
    fmt.Println("different:", c1 != c2)
    // Output: different: true
}
```

### README Code-First Structure
```markdown
# gaz

[![Go Reference](https://pkg.go.dev/badge/...)](https://pkg.go.dev/...)
![Go Version](https://img.shields.io/badge/Go-1.25+-00ADD8)

Simple, type-safe dependency injection with lifecycle management for Go.

## Install

```bash
go get github.com/petabytecl/gaz
```

## Hello World

```go
package main

import (
    "github.com/petabytecl/gaz"
)

func main() {
    app := gaz.New()
    gaz.For[*MyService](app.Container()).Provider(NewMyService)
    app.Build()
    app.Run(context.Background())
}
```

## Next Steps

- [Concepts](docs/concepts.md) - DI fundamentals
- [Configuration](docs/configuration.md) - Config loading
- [Examples](examples/) - Runnable code
```

## State of the Art

| Old Approach | Current Approach | When Changed | Impact |
|--------------|------------------|--------------|--------|
| Separate doc comments | Testable examples | Go 1.0 (2012) | Examples verified by tests |
| README-only docs | pkg.go.dev integration | Go modules (2019) | Automatic API docs |
| Manual example testing | `// Output:` comments | Always | CI catches stale docs |
| godoc server | pkgsite | 2020 | Local preview matches pkg.go.dev |

**Current best practices (2025+):**
- Testable examples in `example_test.go` files
- README focuses on 2-minute quickstart
- Detailed docs link from README to /docs
- Examples directory has CI-tested apps
- Package docs use doc links `[Type]` syntax (Go 1.19+)

## Open Questions

1. **Exact /docs structure**
   - What we know: Need getting-started, concepts, configuration, advanced, troubleshooting
   - What's unclear: Whether to add per-subsystem pages (docs/health.md, docs/worker.md)
   - Recommendation: Start simple, add subsystem pages only if README sections become too long

2. **CI testing for /examples**
   - What we know: Examples should compile and run
   - What's unclear: How to test lifecycle-heavy apps (HTTP servers, workers)
   - Recommendation: `go build ./examples/...` in CI, manual smoke test for complex apps

## Sources

### Primary (HIGH confidence)
- Go Blog: Testable Examples in Go (https://go.dev/blog/examples) - Example function patterns
- Go Doc Comments (https://go.dev/doc/comment) - Comment formatting
- Context7 /websites/go_dev_doc - godoc patterns, function docs

### Secondary (MEDIUM confidence)
- WebSearch "go godoc testable examples best practices 2025" - Confirmed patterns
- WebSearch "Go README best practices structure 2025" - README conventions

### Tertiary (LOW confidence)
- None - all findings verified with official sources

## Existing Documentation State

**Packages with doc.go:**
- gaz (root) ✓
- di ✓
- config ✓
- worker ✓
- cron ✓
- eventbus ✓
- gaztest ✓

**Missing doc.go:**
- health (needs creation)
- config/viper (has doc.go)
- logger (needs review)

**Existing example files:**
- gaz: example_test.go, example_lifecycle_test.go, example_config_test.go (19 examples)
- gaztest: example_test.go, examples_test.go (8 examples)
- di: none (needs creation)
- config: none (needs creation)
- health: none (needs creation)
- worker: none (needs creation)
- cron: none (needs creation)
- eventbus: none (needs creation)

**Existing /examples apps:** 7 (basic, http-server, config-loading, lifecycle, modules, cobra-cli, system-info-cli)

**Existing /docs pages:** 5 (getting-started, concepts, configuration, validation, advanced)

## Metadata

**Confidence breakdown:**
- Standard stack: HIGH - Go's built-in toolchain, official docs
- Architecture: HIGH - Patterns from official Go blog and stdlib
- Pitfalls: HIGH - Verified against official documentation
- Code examples: HIGH - From official Go examples blog post

**Research date:** 2026-02-01
**Valid until:** 2026-03-01 (documentation patterns are stable)
