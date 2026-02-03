---
quick_task: 006
type: execute
files_modified:
  - server/module.go
  - server/module_test.go
autonomous: true

must_haves:
  truths:
    - "server/module.go no longer imports gaz package"
    - "NewModuleWithFlags takes *pflag.FlagSet as first parameter"
    - "NewModuleWithFlags returns di.Module (not gaz.Module)"
    - "Flag values are read via deferred pointer evaluation"
    - "All tests pass with updated signatures"
  artifacts:
    - path: "server/module.go"
      provides: "Refactored module without gaz import"
      contains: "func NewModuleWithFlags(fs *pflag.FlagSet"
    - path: "server/module_test.go"
      provides: "Updated tests for new signature"
      contains: "NewModuleWithFlags(fs"
  key_links:
    - from: "server/module.go"
      to: "di.NewModuleFunc"
      via: "return statement"
      pattern: "return di\\.NewModuleFunc"
---

<objective>
Refactor server/module.go to remove gaz package import and follow the gateway pattern.

Purpose: Eliminate potential import cycle by removing gaz import from server package.
Output: server/module.go using di.NewModuleFunc pattern, updated tests passing.
</objective>

<context>
@.planning/STATE.md
@server/module.go
@server/module_test.go
@server/gateway/module.go (reference pattern)
</context>

<tasks>

<task type="auto">
  <name>Task 1: Refactor server/module.go to gateway pattern</name>
  <files>server/module.go</files>
  <action>
Refactor NewModuleWithFlags to follow the gateway/module.go pattern:

1. Remove `gaz` import from server/module.go - only keep di, pflag, and server sub-packages
2. Change signature from:
   ```go
   func NewModuleWithFlags(opts ...ModuleOption) gaz.Module
   ```
   to:
   ```go
   func NewModuleWithFlags(fs *pflag.FlagSet, opts ...ModuleOption) di.Module
   ```

3. Replace gaz.NewModule().Flags().Provide().Build() with:
   - Define flag pointers using fs.IntVar/fs.BoolVar to bind to cfg struct (same as current)
   - Return di.NewModuleFunc with closure that reads cfg values (deferred evaluation)

4. Implementation pattern (from gateway):
   ```go
   func NewModuleWithFlags(fs *pflag.FlagSet, opts ...ModuleOption) di.Module {
       cfg := defaultModuleConfig()
       for _, opt := range opts {
           opt(cfg)
       }

       // Bind flags to cfg struct (values written when flags parsed)
       fs.IntVar(&cfg.grpcPort, "grpc-port", cfg.grpcPort, "gRPC server port")
       fs.IntVar(&cfg.httpPort, "http-port", cfg.httpPort, "HTTP server port")
       fs.BoolVar(&cfg.grpcReflection, "grpc-reflection", cfg.grpcReflection, "Enable gRPC reflection")
       fs.BoolVar(&cfg.grpcDevMode, "grpc-dev-mode", cfg.grpcDevMode, "Enable gRPC development mode")

       return di.NewModuleFunc("server", func(c *di.Container) error {
           // cfg values are read HERE after flag parsing
           return registerServerComponents(cfg, c)
       })
   }
   ```

5. Update doc comment to reflect new signature (takes fs *pflag.FlagSet as first param).

Note: The key difference is:
- OLD: Returns gaz.Module with FlagsFn() method that App.Use() calls
- NEW: Takes fs directly, binds flags immediately, returns di.Module
  </action>
  <verify>
go build ./server/...
  </verify>
  <done>
server/module.go compiles without gaz import, NewModuleWithFlags takes *pflag.FlagSet and returns di.Module.
  </done>
</task>

<task type="auto">
  <name>Task 2: Update server/module_test.go for new signature</name>
  <files>server/module_test.go</files>
  <action>
Update TestNewModuleWithFlags tests to work with new signature:

1. Remove `gaz` import from test file (if possible - may still need for app integration tests)

2. Update test cases:

   a) "flags registration" test:
   - Create pflag.FlagSet first
   - Pass fs to NewModuleWithFlags(fs)
   - Verify flags are registered on fs directly (no FlagsFn interface check)

   b) "options affect defaults" test:
   - Create fs, call NewModuleWithFlags(fs, opts...)
   - Verify flag defaults on fs

   c) "flag values used at resolution" test:
   - Create fs, call NewModuleWithFlags(fs)
   - Parse flags on fs
   - Verify values were parsed

   d) "cobra integration" test:
   - Create cobra.Command
   - Pass cmd.PersistentFlags() to NewModuleWithFlags
   - Parse and verify

   e) "module name" test:
   - Create fs, call NewModuleWithFlags(fs)
   - Check m.Name() == "server"

   f) "full module apply" test:
   - This test uses gaz.New().Use(m) - the new di.Module won't work with Use()
   - Change to use di.Container directly: create container, register logger, call m.Register(c)

   g) "resolved config uses flag values" test:
   - Similar refactor: use di.Container directly instead of gaz.App

3. Remove any assertions that check for FlagsFn interface (di.Module doesn't have it).

4. Keep gaz import ONLY if needed for integration tests that test the actual server behavior.
  </action>
  <verify>
go test -race -v ./server/...
  </verify>
  <done>
All server module tests pass. Tests verify flag registration, flag parsing, and component resolution with new signature.
  </done>
</task>

</tasks>

<verification>
```bash
# Build all server packages
go build ./server/...

# Run server tests with race detection
go test -race -v ./server/...

# Verify no gaz import in server/module.go
! grep -q '"github.com/petabytecl/gaz"' server/module.go && echo "PASS: No gaz import" || echo "FAIL: gaz import found"

# Run full test suite
make test

# Check coverage still meets threshold
make cover
```
</verification>

<success_criteria>
- server/module.go has no import of "github.com/petabytecl/gaz"
- NewModuleWithFlags signature is func(fs *pflag.FlagSet, opts ...ModuleOption) di.Module
- All tests in server/module_test.go pass
- Coverage threshold (90%) still met
- make lint passes
</success_criteria>

<output>
After completion, create `.planning/quick/006-refactor-server-module-remove-gaz-import/006-SUMMARY.md`
</output>
