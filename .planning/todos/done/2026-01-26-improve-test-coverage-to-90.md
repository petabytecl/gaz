---
created: 2026-01-26T19:00
title: Improve test coverage to 90%
area: testing
files:
  - Makefile
---

## Problem

The project Makefile enforces a 90% code coverage threshold. We need to ensure the codebase meets this standard to pass CI and maintain quality. Current coverage may be insufficient or borderline.

## Solution

1. Run coverage analysis (e.g., `make test` or `go test -coverprofile=...`).
2. Identify packages and functions with low coverage.
3. Add targeted unit tests using the `testify` suite to cover edge cases and error paths.
4. Verify coverage meets or exceeds 90%.
