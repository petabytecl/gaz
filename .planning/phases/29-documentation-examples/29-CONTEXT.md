# Phase 29: Documentation & Examples - Context

**Gathered:** 2026-02-01
**Status:** Ready for planning

<domain>
## Phase Boundary

Complete user-facing documentation for gaz v3. README with getting started guide, godoc examples for all major public APIs, and example code using v3 patterns exclusively. Tutorials cover common use cases: DI setup, lifecycle, modules, config.

</domain>

<decisions>
## Implementation Decisions

### README structure
- Code-first approach: Install -> Hello World -> Concepts (get running in 2 minutes)
- Show both paths after quick start: DI core (Container/Provide/Resolve) AND App-centric (App.New/Modules/Run)
- Hybrid with links: README has sections, each links to detailed /docs pages
- README sections: Getting Started, Core Concepts, Subsystems
- Testing and Comparison sections go in /docs only

### Godoc examples
- Comprehensive coverage: every exported function/type gets an example
- Full scenario examples: realistic usage with context (not just minimal API calls)
- Testable examples with `// Output:` comments when feasible (skip for async/lifecycle-heavy APIs)
- All packages need examples: gaz, di, config, health, worker, cron, eventbus, gaztest

### Tutorial depth
- Use-case based tutorials: Web service, Background workers, CLI application, Microservices
- Full working apps: 50-100+ lines of tutorial code each
- Code-primary: runnable examples in /examples directory, minimal prose
- CI-tested: example apps verified to compile and run in CI

### Audience level
- DI-experienced developers: assume knowledge of DI patterns, just show gaz approach
- No fx/wire comparison: let gaz stand alone, don't reference other DI frameworks
- Conceptual depth: explain why patterns exist and when to use them
- Troubleshooting page: brief section covering common mistakes (not inline)

### Claude's Discretion
- Exact directory structure under /docs
- Code example formatting and style
- Order of concepts within each section
- Troubleshooting content selection

</decisions>

<specifics>
## Specific Ideas

No specific requirements -- open to standard approaches

</specifics>

<deferred>
## Deferred Ideas

None -- discussion stayed within phase scope

</deferred>

---

*Phase: 29-documentation-examples*
*Context gathered: 2026-02-01*
