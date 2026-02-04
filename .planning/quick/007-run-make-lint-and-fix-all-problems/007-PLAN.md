---
phase: 007-run-make-lint-and-fix-all-problems
plan: 007
type: execute
wave: 1
depends_on: []
files_modified:
  - server/gateway/handler.go
  - server/gateway/module.go
  - health/module.go
  - server/grpc/server.go
  - server/http/module.go
  - server/grpc/config.go
  - health/module_test.go
  - server/grpc/module.go
  - server/otel/module.go
  - server/gateway/gateway.go
autonomous: true
must_haves:
  truths:
    - "make lint passes with no errors"
    - "make test passes (no regressions)"
  artifacts:
    - path: "server/gateway/module.go"
      provides: "NewModule with reduced complexity"
  key_links: []
---

<objective>
Run `make lint` and fix all 17 identified issues to ensure code quality and standard compliance.

Purpose: Maintain 100% lint compliance as required by project standards.
Output: Clean `make lint` output.
</objective>

<execution_context>
@AGENTS.md
</execution_context>

<tasks>

<task type="auto">
  <name>Task 1: Fix Simple Linting Issues</name>
  <files>
    server/gateway/handler.go
    health/module.go
    server/grpc/server.go
    server/http/module.go
    server/grpc/config.go
    health/module_test.go
    server/grpc/module.go
    server/otel/module.go
    server/gateway/gateway.go
    server/gateway/module.go
  </files>
  <action>
    Fix the following lint errors:
    - errcheck: server/gateway/handler.go (check error)
    - gocritic: health/module.go (fix Deprecated comment format)
    - govet: server/grpc/server.go, server/http/module.go (fix shadow 'err')
    - mnd: server/grpc/config.go (replace '5' with constant or variable)
    - nolintlint: health/module_test.go (remove unused directive)
    - revive: 
        - health/module.go (unused parameters in deprecated funcs - name as _)
        - server/* modules (empty blocks - add comment or logging)
    - staticcheck: server/gateway/gateway.go (remove unnecessary type declaration)
  </action>
  <verify>
    make lint
    (Should only fail on server/gateway/module.go complexity)
  </verify>
  <done>All lint errors except gocognit complexity are resolved</done>
</task>

<task type="auto">
  <name>Task 2: Refactor Gateway Module Complexity</name>
  <files>server/gateway/module.go</files>
  <action>
    Refactor `NewModule` in `server/gateway/module.go` to reduce cognitive complexity below 20.
    - Extract configuration loading logic into a helper function (e.g., `loadConfig`).
    - Extract flag registration into a helper function (e.g., `registerFlags`).
    - Simplify the main builder flow.
  </action>
  <verify>make lint</verify>
  <done>make lint passes with 0 issues</done>
</task>

<task type="auto">
  <name>Task 3: Verify Integrity</name>
  <files>None</files>
  <action>
    Run full test suite to ensure fixes didn't introduce regressions.
  </action>
  <verify>make test</verify>
  <done>All tests pass</done>
</task>

</tasks>

<success_criteria>
- [ ] `make lint` returns exit code 0
- [ ] `make test` returns exit code 0
- [ ] Cognitive complexity of `NewModule` is < 20
</success_criteria>

<output>
After completion, create `.planning/quick/007-run-make-lint-and-fix-all-problems/007-SUMMARY.md`
</output>
