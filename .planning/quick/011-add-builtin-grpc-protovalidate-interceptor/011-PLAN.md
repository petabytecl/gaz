---
phase: 011
plan: 01
type: execute
wave: 1
depends_on: []
files_modified:
  - server/grpc/interceptors.go
  - server/grpc/module.go
  - server/grpc/interceptors_test.go
  - go.mod
autonomous: true
---

<objective>
Add a built-in ValidationBundle that integrates protovalidate from buf.build for gRPC request validation.

Purpose: Enable automatic validation of protobuf messages using protovalidate rules before they reach handlers.
Output: ValidationBundle with priority 100, auto-discovered like LoggingBundle and RecoveryBundle.
</objective>

<execution_context>
@~/.config/opencode/get-shit-done/workflows/execute-plan.md
</execution_context>

<context>
@server/grpc/interceptors.go
@server/grpc/module.go
@server/grpc/interceptors_test.go
@AGENTS.md
</context>

<tasks>

<task type="auto">
  <name>Task 1: Add protovalidate dependency and ValidationBundle</name>
  <files>
    go.mod
    server/grpc/interceptors.go
  </files>
  <action>
1. Add dependency:
   ```bash
   go get github.com/bufbuild/protovalidate-go
   ```
   Note: go-grpc-middleware/v2/interceptors/protovalidate is already available via existing go-grpc-middleware dependency.

2. In `server/grpc/interceptors.go`:
   - Add import for `github.com/bufbuild/protovalidate-go` and `github.com/grpc-ecosystem/go-grpc-middleware/v2/interceptors/protovalidate`
   - Add `PriorityValidation = 100` constant (after logging=0, before custom interceptors, before recovery=1000)
   - Create `ValidationBundle` struct with `validator *protovalidate.Validator` field
   - Create `NewValidationBundle()` constructor that calls `protovalidate.New()` (returns error if validator creation fails)
   - Implement `Name() string` returning "validation"
   - Implement `Priority() int` returning `PriorityValidation`
   - Implement `Interceptors()` returning `protovalidate.UnaryServerInterceptor(b.validator)` and `protovalidate.StreamServerInterceptor(b.validator)`

Follow exact pattern from LoggingBundle and RecoveryBundle.
  </action>
  <verify>
    `go build ./server/grpc/...` succeeds
  </verify>
  <done>
    ValidationBundle implements InterceptorBundle with priority 100, uses protovalidate for validation.
  </done>
</task>

<task type="auto">
  <name>Task 2: Register ValidationBundle in module</name>
  <files>
    server/grpc/module.go
  </files>
  <action>
1. Add `provideValidationBundle` function following the pattern of `provideLoggingBundle`:
   ```go
   func provideValidationBundle(c *gaz.Container) error {
       if err := gaz.For[*ValidationBundle](c).Provider(func(c *gaz.Container) (*ValidationBundle, error) {
           return NewValidationBundle()
       }); err != nil {
           return fmt.Errorf("register validation bundle: %w", err)
       }
       return nil
   }
   ```

2. Add `.Provide(provideValidationBundle)` to the module builder chain in `NewModule()`, after `provideLoggingBundle` and before `provideRecoveryBundle`.

3. Update the docstring for `NewModule()` to include `*grpc.ValidationBundle (protovalidate interceptor)` in the components list.
  </action>
  <verify>
    `go build ./server/grpc/...` succeeds
  </verify>
  <done>
    ValidationBundle is registered in DI container and will be auto-discovered by the gRPC server.
  </done>
</task>

<task type="auto">
  <name>Task 3: Add tests for ValidationBundle</name>
  <files>
    server/grpc/interceptors_test.go
  </files>
  <action>
1. Add test `TestValidationBundleImplementsInterface` to the suite:
   - Create bundle with `NewValidationBundle()`
   - Assert no error on creation
   - Verify interface compliance: `var _ InterceptorBundle = bundle`
   - Assert `bundle.Name() == "validation"`
   - Assert `bundle.Priority() == PriorityValidation`
   - Assert both unary and stream interceptors are not nil

2. Update `TestCollectInterceptorsOrdering`:
   - Register `*ValidationBundle` instance in the container
   - Update assertions: expect 4 interceptors (logging, validation, custom, recovery)

3. Add test verifying priority order: logging (0) < validation (100) < custom (50 -> change to 500) < recovery (1000)
   - Or adjust custom priority to 500 to test the full ordering: logging=0, validation=100, custom=500, recovery=1000
  </action>
  <verify>
    `go test -race -v ./server/grpc/... -run "TestValidation|TestCollectInterceptors"` passes
  </verify>
  <done>
    ValidationBundle has test coverage for interface compliance and priority ordering.
  </done>
</task>

</tasks>

<verification>
```bash
# Build passes
go build ./server/grpc/...

# All tests pass with race detection
go test -race -v ./server/grpc/...

# Lint passes
make lint
```
</verification>

<success_criteria>
- ValidationBundle implements InterceptorBundle
- Priority constant PriorityValidation = 100 exists
- ValidationBundle registered in gRPC module
- Auto-discovered and chained with LoggingBundle and RecoveryBundle
- Tests pass, coverage maintained >90%
</success_criteria>

<output>
After completion, update `.planning/STATE.md` Quick Tasks Completed table with:
| 011 | Add builtin grpc protovalidate interceptor | {date} | {commit} | [011-...](./quick/011-add-builtin-grpc-protovalidate-interceptor/) |
</output>
