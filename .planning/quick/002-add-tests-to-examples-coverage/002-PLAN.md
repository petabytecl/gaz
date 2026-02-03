---
phase: 002-add-tests-to-examples-coverage
plan: 002
type: execute
wave: 1
depends_on: []
files_modified:
  - examples/basic/main.go
  - examples/basic/main_test.go
  - examples/config-loading/main.go
  - examples/config-loading/main_test.go
  - examples/modules/main.go
  - examples/modules/main_test.go
  - examples/http-server/main.go
  - examples/http-server/main_test.go
  - examples/lifecycle/main.go
  - examples/lifecycle/main_test.go
  - examples/background-workers/main.go
  - examples/background-workers/main_test.go
  - examples/microservice/main.go
  - examples/microservice/main_test.go
  - examples/cobra-cli/main.go
  - examples/cobra-cli/main_test.go
  - examples/system-info-cli/main.go
  - examples/system-info-cli/main_test.go
autonomous: true
must_haves:
  truths:
    - "All examples have a corresponding test file"
    - "Running go test ./examples/... executes the example code"
    - "Example main functions are refactored to be testable (run/execute methods)"
  artifacts:
    - path: "examples/basic/main_test.go"
      provides: "Test for basic example"
    - path: "examples/http-server/main_test.go"
      provides: "Test for http-server example"
  key_links:
    - from: "examples/*/main_test.go"
      to: "examples/*/main.go"
      via: "run/execute function call"
---

<objective>
Add tests to all examples to ensure they are covered by CI and don't drift from the codebase.
Refactor main packages to expose testable run/execute functions.

Purpose: Prevent regression in examples and increase code coverage metrics.
Output: *_test.go files in all example directories.
</objective>

<execution_context>
@~/.config/opencode/get-shit-done/workflows/execute-plan.md
@~/.config/opencode/get-shit-done/templates/summary.md
</execution_context>

<context>
@.planning/STATE.md
</context>

<tasks>

<task type="auto">
  <name>Task 1: Test simple non-blocking examples</name>
  <files>
    examples/basic/main.go
    examples/basic/main_test.go
    examples/config-loading/main.go
    examples/config-loading/main_test.go
    examples/modules/main.go
    examples/modules/main_test.go
  </files>
  <action>
    Refactor main() in basic, config-loading, and modules examples to extract a run() function that returns error.
    Call run() from main() and handle errors.
    Create main_test.go in each directory that calls run() and asserts no error.
    For config-loading, ensure it can run without config file (defaults) or provide one.
  </action>
  <verify>go test -v ./examples/basic ./examples/config-loading ./examples/modules</verify>
  <done>Tests pass and cover the run/main logic.</done>
</task>

<task type="auto">
  <name>Task 2: Test blocking/server examples</name>
  <files>
    examples/http-server/main.go
    examples/http-server/main_test.go
    examples/lifecycle/main.go
    examples/lifecycle/main_test.go
    examples/background-workers/main.go
    examples/background-workers/main_test.go
    examples/microservice/main.go
    examples/microservice/main_test.go
  </files>
  <action>
    Refactor main() in http-server, lifecycle, background-workers, and microservice to extract run(ctx context.Context, ...) or similar.
    Ensure configuration (like ports) can be overridden for testing (e.g. use port 0 or check for environment variables).
    Create main_test.go that runs the application in a goroutine or with a short-lived context/timeout.
    Verify clean startup and shutdown.
  </action>
  <verify>go test -v ./examples/http-server ./examples/lifecycle ./examples/background-workers ./examples/microservice</verify>
  <done>Tests pass, servers start and stop gracefully.</done>
</task>

<task type="auto">
  <name>Task 3: Test CLI examples</name>
  <files>
    examples/cobra-cli/main.go
    examples/cobra-cli/main_test.go
    examples/system-info-cli/main.go
    examples/system-info-cli/main_test.go
  </files>
  <action>
    Refactor cobra-cli main() to expose execute() or similar that accepts args/streams.
    Create main_test.go for cobra-cli that runs the "version" command or starts "serve" with timeout.
    
    For system-info-cli (separate module):
    Refactor main() to extract run logic.
    Add main_test.go.
    Note: verify command will need to be run inside the directory for system-info-cli.
  </action>
  <verify>
    go test -v ./examples/cobra-cli
    cd examples/system-info-cli && go test -v .
  </verify>
  <done>CLI examples are tested and verifiable.</done>
</task>

</tasks>

<verification>
Run all example tests:
go test -v ./examples/...
(cd examples/system-info-cli && go test -v .)
</verification>

<success_criteria>
- All examples have at least one test
- CI coverage includes examples code (when run with appropriate flags)
- No broken examples
</success_criteria>

<output>
After completion, create .planning/phases/002-add-tests-to-examples-coverage/002-002-SUMMARY.md
</output>
