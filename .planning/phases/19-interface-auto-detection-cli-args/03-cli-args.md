---
id: 03-cli-args.md
wave: 3
depends_on: [02-lifecycle-execution.md]
files_modified:
  - gaz.go
  - types.go
  - cobra.go
  - integration_test.go
autonomous: true
---

# Plan 03: CLI Integration (CommandArgs)

**Goal:** Make CLI positional arguments and the active `cobra.Command` accessible to services via Dependency Injection.

## Context
Services often need access to the command line arguments passed to the application. Since `gaz` integrates with `cobra`, we can capture these during the bootstrap phase and inject them.

## Tasks

<task>
  <description>Define CommandArgs type</description>
  <instructions>
    In `types.go` (or create `args.go` if preferred):
    1. Define `type CommandArgs struct`.
    2. Fields: `Command *cobra.Command`, `Args []string`.
  </instructions>
  <files>
    <file>types.go</file>
  </files>
</task>

<task>
  <description>Add GetArgs helper</description>
  <instructions>
    In `gaz.go`:
    1. Add function `GetArgs(c *di.Container) []string`.
    2. Implementation:
       - Resolve `*CommandArgs`.
       - If error (not found), return nil (or empty slice).
       - If found, return `Args`.
  </instructions>
  <files>
    <file>gaz.go</file>
  </files>
</task>

<task>
  <description>Inject CommandArgs in bootstrap</description>
  <instructions>
    In `cobra.go`:
    1. Locate `bootstrap` method in `App`.
    2. Before `a.Build()`, create `&CommandArgs{Command: cmd, Args: args}`.
    3. Register it: `For[*CommandArgs](a.container).Instance(cmdArgs)`.
  </instructions>
  <files>
    <file>cobra.go</file>
  </files>
</task>

<task>
  <description>Integration Test</description>
  <instructions>
    In `integration_test.go` (or new `cli_test.go`):
    1. Create a test `App` with a service that depends on `*CommandArgs`.
    2. Simulate a run with arguments.
    3. Verify the service received the correct arguments.
  </instructions>
  <files>
    <file>integration_test.go</file>
  </files>
</task>

## Verification
- Run integration tests.
- Verify `CommandArgs` are available in the container.
