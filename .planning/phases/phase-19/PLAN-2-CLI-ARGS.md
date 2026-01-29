# Plan: CLI Arguments Injection

**Phase:** 19
**Wave:** 1
**Depends On:** None
**Files Modified:**
- `cobra.go`
- `args.go` (new)
- `cobra_test.go` (new or update)

## Context
Applications often need access to command-line arguments passed to the binary. While `cobra` handles parsing, passing these down to services usually requires global state or manual plumbing. We want to inject these arguments directly into the DI container so services can request them.

## Goals
1.  Capture the `*cobra.Command` and positional `args []string` during app bootstrap.
2.  Register a `CommandArgs` struct in the container.
3.  Provide a helper `gaz.GetArgs(c)` for easy access.

## Tasks

<task>
<id>1</id>
<description>Define CommandArgs Type</description>
<steps>
    <step>Create `args.go` in `gaz` package</step>
    <step>Define public struct `CommandArgs` with fields `Command *cobra.Command` and `Args []string`</step>
    <step>Add `GetArgs(c *di.Container) []string` helper function</step>
</steps>
</task>

<task>
<id>2</id>
<description>Inject Args in Bootstrap</description>
<steps>
    <step>Modify `gaz/cobra.go` implementation of `bootstrap`</step>
    <step>Construct `CommandArgs` from the passed `cmd` and `args`</step>
    <step>Register `CommandArgs` as a singleton instance in the container: `gaz.For[*CommandArgs](a.container).Instance(cmdArgs)`</step>
    <step>Ensure this happens BEFORE `a.container.Build()`</step>
</steps>
</task>

<task>
<id>3</id>
<description>Verify Argument Injection</description>
<steps>
    <step>Create `gaz/cobra_test.go`</step>
    <step>Setup a test `cobra.Command` with a `gaz.New` app</step>
    <step>Execute the command with arguments (e.g. `cmd.SetArgs(...)`)</step>
    <step>Inside the app (e.g. via a test service or explicit check), verify `gaz.GetArgs` returns the expected values</step>
</steps>
</task>

## Verification Criteria
- [ ] `CommandArgs` struct is exported
- [ ] `gaz.GetArgs(container)` returns the slice of positional arguments provided to the CLI command
- [ ] Services can depend on `*gaz.CommandArgs`
