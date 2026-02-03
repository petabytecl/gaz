---
phase: 003-improve-test-coverage-to-90
plan: 003
type: execute
wave: 1
depends_on: []
files_modified:
  - examples/cobra-cli/main_test.go
  - examples/lifecycle/main_test.go
  - examples/system-info-cli/go.mod
  - examples/system-info-cli/go.sum
autonomous: true
must_haves:
  truths:
    - "Test coverage for examples/cobra-cli increases significantly"
    - "Test coverage for examples/lifecycle increases significantly"
    - "Total project coverage exceeds 90%"
    - "All examples compile successfully"
  artifacts:
    - path: "examples/cobra-cli/main_test.go"
      provides: "Tests for Server lifecycle and CLI flags"
    - path: "examples/lifecycle/main_test.go"
      provides: "Tests for Server lifecycle methods"
  key_links:
    - from: "examples/cobra-cli/main_test.go"
      to: "examples/cobra-cli/main.go"
      via: "Unit tests"
---

<objective>
Improve test coverage to >90% by adding tests to low-coverage examples.

Purpose: Ensure examples are reliable and contribute positively to overall project coverage metrics.
Output: Updated test files in examples/cobra-cli and examples/lifecycle.
</objective>

<execution_context>
@~/.config/opencode/get-shit-done/workflows/execute-plan.md
@~/.config/opencode/get-shit-done/templates/summary.md
</execution_context>

<context>
@.planning/STATE.md
@examples/cobra-cli/main.go
@examples/lifecycle/main.go
</context>

<tasks>

<task type="auto">
  <name>Task 1: Fix dependencies in examples/system-info-cli</name>
  <files>examples/system-info-cli/go.mod, examples/system-info-cli/go.sum</files>
  <action>
    Run `go mod tidy` in `examples/system-info-cli` to fix missing dependencies identified by the build check.
    Ensure `make test` runs cleanly on all examples.
  </action>
  <verify>cd examples/system-info-cli && go build .</verify>
  <done>examples/system-info-cli builds successfully</done>
</task>

<task type="auto">
  <name>Task 2: Add tests to examples/cobra-cli</name>
  <files>examples/cobra-cli/main_test.go</files>
  <action>
    Add tests for:
    - `NewServer` and `Server` struct methods (`Start`, `OnStart`, `OnStop`).
    - `execute` function with different flag combinations (debug, port, host).
    - `runServe` (if possible) using a cancelled context or mocked args to ensure flag parsing logic is covered.
    
    Use `bytes.Buffer` to capture output and assert expectations.
  </action>
  <verify>go test -v -cover ./examples/cobra-cli/...</verify>
  <done>Coverage for examples/cobra-cli > 60%</done>
</task>

<task type="auto">
  <name>Task 3: Add tests to examples/lifecycle</name>
  <files>examples/lifecycle/main_test.go</files>
  <action>
    Add tests for:
    - `Server.OnStart` and `Server.OnStop` directly to ensure unit coverage.
    - Enhance `TestRun` to verify successful shutdown and error propagation.
    - Cover the `run` function's error paths if possible.
  </action>
  <verify>go test -v -cover ./examples/lifecycle/...</verify>
  <done>Coverage for examples/lifecycle > 70%</done>
</task>

<task type="auto">
  <name>Task 4: Verify total coverage</name>
  <files>coverage.out</files>
  <action>
    Run the full project coverage check to ensure we met the 90% threshold.
  </action>
  <verify>make cover</verify>
  <done>Total coverage >= 90%</done>
</task>

</tasks>

<verification>
Run `make cover` and confirm the final percentage is >= 90%.
</verification>

<success_criteria>
- [ ] examples/system-info-cli builds successfully
- [ ] examples/cobra-cli coverage improved
- [ ] examples/lifecycle coverage improved
- [ ] Total coverage >= 90%
</success_criteria>

<output>
After completion, create .planning/phases/003-improve-test-coverage-to-90/003-003-SUMMARY.md
</output>
