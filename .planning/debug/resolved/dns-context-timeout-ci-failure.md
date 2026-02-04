---
status: resolved
trigger: "DNS health check test TestNew_ContextTimeout fails on CI but passes locally"
created: 2026-02-04T00:00:00Z
updated: 2026-02-04T00:00:00Z
---

## Current Focus

hypothesis: "localhost" DNS resolution is synchronous (from /etc/hosts) and completes before context cancellation is checked
test: Examine how LookupHost handles cancelled context with instant resolutions
expecting: If localhost resolves from hosts file, it may not check context at all
next_action: Verify hypothesis by understanding Go's LookupHost behavior with pre-cancelled contexts

## Symptoms

expected: DNS lookup with a cancelled context should return an error
actual: Test passes locally but fails on CI - the cancelled context doesn't produce an error on GitHub Actions
errors: dns_test.go:59: expected error for cancelled context
reproduction: Run `make cover` on CI (GitHub Actions ubuntu runner). Test `TestNew_ContextTimeout` in `health/checks/dns` package fails.
started: Multiple recent commits failing - appears to be environment-dependent behavior

## Eliminated

## Evidence

- timestamp: 2026-02-04T00:01:00Z
  checked: Test code in dns_test.go lines 47-61
  found: Test creates cancelled context, then calls check with "localhost" as host
  implication: Test relies on LookupHost respecting cancelled context

- timestamp: 2026-02-04T00:02:00Z
  checked: Implementation in dns.go lines 38-54
  found: Uses net.Resolver.LookupHost with the passed context
  implication: If LookupHost returns instantly (hosts file), context may not be checked

- timestamp: 2026-02-04T00:03:00Z
  checked: Behavior with CGO_ENABLED=0 vs CGO_ENABLED=1
  found: |
    CGO_ENABLED=0 (pure Go resolver): LookupHost succeeds with cancelled context!
    CGO_ENABLED=1 (cgo resolver): LookupHost returns "operation was canceled"
  implication: CI uses pure Go resolver (no cgo), which reads /etc/hosts synchronously without checking context

- timestamp: 2026-02-04T00:04:00Z
  checked: Why CI uses pure Go resolver
  found: GitHub Actions may have CGO_ENABLED=0 or no C toolchain, forcing pure Go resolver
  implication: The test is flaky because it depends on resolver implementation detail

## Resolution

root_cause: Pure Go DNS resolver reads /etc/hosts synchronously without checking context cancellation. CI (GitHub Actions) uses pure Go resolver, which ignores the cancelled context when "localhost" resolves from /etc/hosts. The test passes locally when cgo resolver is available but fails on CI.
fix: Added explicit context.Err() check before calling LookupHost in dns.go. This ensures consistent behavior regardless of resolver implementation.
verification: Tests pass with both CGO_ENABLED=0 and CGO_ENABLED=1. Full test suite passes with 90.4% coverage. Lint passes.
files_changed:
  - health/checks/dns/dns.go
