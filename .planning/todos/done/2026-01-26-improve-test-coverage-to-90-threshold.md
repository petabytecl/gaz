---
created: 2026-01-26T19:13
title: Improve test coverage to 90% threshold
area: testing
files:
  - service_wrappers.go
---

## Problem

Coverage is at 85.2% (below the 90% threshold enforced in Makefile). The gap is due to *Any wrapper lifecycle methods (OnStart, OnStop, etc.) that are never called because they're filtered out by hasLifecycle() during startup/shutdown.

This is technical debt from Plan 03-01 which introduced non-generic *Any service wrappers for reflection-based registration. The *Any wrappers implement the full serviceWrapper interface but their lifecycle methods are unreachable in practice.

Documented as known issue in STATE.md and 03-04-SUMMARY.md.

## Solution

Options to consider:
1. Add direct unit tests for *Any wrapper lifecycle methods (even if not called in production flow)
2. Refactor to extract lifecycle logic, reducing duplication
3. Use coverage exclusion comments for intentionally dead code paths
4. Review if hasLifecycle() filter can be adjusted to include *Any wrappers when appropriate

TBD - needs investigation of actual wrapper structure.
