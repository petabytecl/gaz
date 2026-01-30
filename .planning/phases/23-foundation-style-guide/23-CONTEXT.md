# Phase 23: Foundation & Style Guide - Context

**Gathered:** 2026-01-29
**Status:** Ready for planning

<domain>
## Phase Boundary

Establish naming conventions and API patterns before making v3.0 API changes. Produces STYLE.md with documented conventions for constructors, errors, and module factories. This phase creates the foundation that all subsequent v3.0 phases will follow.

</domain>

<decisions>
## Implementation Decisions

### Document structure & navigation
- Single STYLE.md file at repository root
- Organized by categories (naming, constructors, errors, modules)
- Headers only for navigation (no TOC or summary tables)
- Primary audience: internal contributors modifying gaz

### Convention depth & examples
- Standard depth: rule, one good example, one bad example
- Hybrid examples: real gaz code when clear, simplified when complex
- Brief rationale for each convention (one-liner explaining why)
- Self-contained document (no external references required)

### Prescription level
- Strict MUST language — all rules are mandatory
- Exception process documented for deviations
- Future automation noted — linter rules added later where possible
- Mark automatable rules in the document for later tooling

### Scope of conventions
- Applies to all code (public and internal)
- Phase 23 focuses on API conventions only
- Code style conventions deferred to a later pass
- Cleanup pass after conventions defined (not update-on-touch)

### Claude's Discretion
- Exact section ordering within categories
- How to format good/bad examples (side-by-side vs sequential)
- Exception process wording
- Which conventions to mark as automatable

</decisions>

<specifics>
## Specific Ideas

No specific requirements — open to standard approaches.

</specifics>

<deferred>
## Deferred Ideas

- Code style conventions (formatting, comments) — follow-up after API conventions
- Linter implementation — Phase 28 testing infrastructure or separate phase

</deferred>

---

*Phase: 23-foundation-style-guide*
*Context gathered: 2026-01-29*
