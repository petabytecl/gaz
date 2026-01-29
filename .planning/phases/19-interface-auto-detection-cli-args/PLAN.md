# Phase 19: Interface Auto-Detection + CLI Args

**Goal:** Enhance developer experience by automatically detecting lifecycle interfaces (`Starter`/`Stopper`) on services and injecting CLI arguments.

## Dependencies
- Phase 18 (v2.0) completed

## Plans
| ID | Name | Description | Wave |
|----|------|-------------|------|
| 01 | [Interface Detection](./01-interface-detection.md) | Implement `HasLifecycle` logic for auto-detection | 1 |
| 02 | [Lifecycle Execution](./02-lifecycle-execution.md) | Handle pointer receivers and explicit precedence | 2 |
| 03 | [CLI Integration](./03-cli-args.md) | Inject `CommandArgs` into DI container | 3 |

## Must Haves
- `HasLifecycle()` returns true for types implementing `Starter`/`Stopper`
- `OnStart`/`OnStop` called automatically for implementing services
- Explicit `.OnStart()` prevents `Starter.OnStart()` from running
- CLI args accessible via `gaz.GetArgs()`
