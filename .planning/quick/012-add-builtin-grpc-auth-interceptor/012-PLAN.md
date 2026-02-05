---
task: 012-add-builtin-grpc-auth-interceptor
type: execute
files_modified:
  - server/grpc/interceptors.go
  - server/grpc/module.go
autonomous: true

must_haves:
  truths:
    - "Auth interceptor is only registered when AuthFunc exists in DI container"
    - "Auth interceptor chains correctly with existing interceptors (logging -> auth -> validation -> recovery)"
    - "Users can register custom AuthFunc to enable authentication"
  artifacts:
    - path: "server/grpc/interceptors.go"
      provides: "AuthBundle implementation and PriorityAuth constant"
      contains: "type AuthBundle struct"
    - path: "server/grpc/module.go"
      provides: "Conditional auth bundle provider"
      contains: "provideAuthBundle"
  key_links:
    - from: "provideAuthBundle"
      to: "di.Resolve[AuthFunc]"
      via: "conditional registration"
      pattern: "gaz\\.Resolve\\[AuthFunc\\]"
---

<objective>
Add builtin gRPC auth interceptor that integrates with go-grpc-middleware/v2/interceptors/auth.

Purpose: Enable optional authentication for gRPC services via DI-based AuthFunc registration.
Output: AuthBundle implementation with conditional registration based on AuthFunc presence.
</objective>

<execution_context>
@~/.config/opencode/get-shit-done/workflows/execute-plan.md
@~/.config/opencode/get-shit-done/templates/summary.md
</execution_context>

<context>
@server/grpc/interceptors.go
@server/grpc/module.go
</context>

<tasks>

<task type="auto">
  <name>Task 1: Add AuthBundle to interceptors.go</name>
  <files>server/grpc/interceptors.go</files>
  <action>
Add the auth interceptor components to interceptors.go:

1. Add import for auth interceptor:
   ```go
   "github.com/grpc-ecosystem/go-grpc-middleware/v2/interceptors/auth"
   ```

2. Add PriorityAuth constant (after PriorityLogging, before PriorityValidation):
   ```go
   // PriorityAuth is the priority for the auth interceptor (after logging, before validation).
   PriorityAuth = 50
   ```

3. Export AuthFunc type alias for convenience:
   ```go
   // AuthFunc is the authentication function type.
   // It extracts and validates credentials from the context, returning an enriched context
   // or an error if authentication fails.
   //
   // Use auth.AuthFromMD to extract tokens from metadata:
   //
   //   func myAuthFunc(ctx context.Context) (context.Context, error) {
   //       token, err := auth.AuthFromMD(ctx, "bearer")
   //       if err != nil {
   //           return nil, status.Errorf(codes.Unauthenticated, "invalid auth token: %v", err)
   //       }
   //       // Validate token and enrich context...
   //       return ctx, nil
   //   }
   //
   // Register in DI to enable auth interceptor:
   //
   //   gaz.For[grpc.AuthFunc](c).Instance(myAuthFunc)
   type AuthFunc = auth.AuthFunc
   ```

4. Create AuthBundle struct:
   ```go
   // AuthBundle is the built-in authentication interceptor bundle.
   // It validates requests using the registered AuthFunc.
   type AuthBundle struct {
       authFunc AuthFunc
   }
   ```

5. Create NewAuthBundle constructor:
   ```go
   // NewAuthBundle creates a new auth interceptor bundle.
   func NewAuthBundle(authFunc AuthFunc) *AuthBundle {
       return &AuthBundle{authFunc: authFunc}
   }
   ```

6. Implement InterceptorBundle interface:
   ```go
   // Name returns the bundle identifier.
   func (b *AuthBundle) Name() string {
       return "auth"
   }

   // Priority returns the auth priority (after logging, before validation).
   func (b *AuthBundle) Priority() int {
       return PriorityAuth
   }

   // Interceptors returns the auth interceptors.
   func (b *AuthBundle) Interceptors() (grpc.UnaryServerInterceptor, grpc.StreamServerInterceptor) {
       return auth.UnaryServerInterceptor(b.authFunc),
           auth.StreamServerInterceptor(b.authFunc)
   }
   ```

Place AuthBundle after LoggingBundle and before RecoveryBundle to maintain priority ordering in the file.
  </action>
  <verify>
Run: `go build ./server/grpc/...` - should compile without errors.
Run: `go vet ./server/grpc/...` - no issues.
  </verify>
  <done>
AuthBundle struct exists with Name(), Priority(), Interceptors() methods.
PriorityAuth constant = 50.
AuthFunc type alias exported.
  </done>
</task>

<task type="auto">
  <name>Task 2: Add conditional provider to module.go</name>
  <files>server/grpc/module.go</files>
  <action>
Add the conditional auth bundle provider to module.go:

1. Create provideAuthBundle function that:
   - Attempts to resolve AuthFunc from container
   - Only registers AuthBundle if AuthFunc exists
   - Silently skips if no AuthFunc (auth is optional)

   ```go
   // provideAuthBundle creates an AuthBundle provider function.
   // The bundle is only registered if an AuthFunc is registered in the container.
   // This makes authentication opt-in - services without AuthFunc skip auth.
   func provideAuthBundle(c *gaz.Container) error {
       // Check if AuthFunc is registered - auth is optional.
       authFunc, err := gaz.Resolve[AuthFunc](c)
       if err != nil {
           // No AuthFunc registered - skip auth interceptor silently.
           return nil
       }

       if err := gaz.For[*AuthBundle](c).Provider(func(_ *gaz.Container) (*AuthBundle, error) {
           return NewAuthBundle(authFunc), nil
       }); err != nil {
           return fmt.Errorf("register auth bundle: %w", err)
       }
       return nil
   }
   ```

2. Add provideAuthBundle to NewModule() chain:
   - Place after provideLoggingBundle
   - Place before provideValidationBundle
   
   Update the module builder chain to include:
   ```go
   Provide(provideLoggingBundle).
   Provide(provideAuthBundle).      // NEW - after logging, before validation
   Provide(provideValidationBundle).
   ```

3. Update the NewModule() docstring to include AuthBundle:
   Add to "Components registered:" list:
   ```
   //   - *grpc.AuthBundle (auth interceptor, only if AuthFunc registered)
   ```
  </action>
  <verify>
Run: `go build ./server/grpc/...` - compiles.
Run: `go vet ./server/grpc/...` - no issues.
  </verify>
  <done>
provideAuthBundle function exists with conditional registration logic.
NewModule() includes provideAuthBundle in the provider chain.
AuthBundle documented in NewModule() docstring.
  </done>
</task>

<task type="auto">
  <name>Task 3: Add tests and verify integration</name>
  <files>server/grpc/interceptors_test.go</files>
  <action>
Add tests for the AuthBundle:

1. Test AuthBundle interface compliance:
   ```go
   func (s *InterceptorsSuite) TestAuthBundle_ImplementsInterface() {
       authFunc := func(ctx context.Context) (context.Context, error) {
           return ctx, nil
       }
       bundle := NewAuthBundle(authFunc)

       // Verify interface compliance.
       var _ InterceptorBundle = bundle

       s.Equal("auth", bundle.Name())
       s.Equal(PriorityAuth, bundle.Priority())

       unary, stream := bundle.Interceptors()
       s.NotNil(unary)
       s.NotNil(stream)
   }
   ```

2. Test PriorityAuth ordering:
   ```go
   func (s *InterceptorsSuite) TestPriorityAuth_Ordering() {
       // Auth should be after logging (0), before validation (100).
       s.Greater(PriorityAuth, PriorityLogging)
       s.Less(PriorityAuth, PriorityValidation)
   }
   ```

3. Test conditional registration (AuthFunc present):
   ```go
   func (s *InterceptorsSuite) TestProvideAuthBundle_WithAuthFunc() {
       c := di.New()

       // Register AuthFunc.
       authFunc := AuthFunc(func(ctx context.Context) (context.Context, error) {
           return ctx, nil
       })
       err := gaz.For[AuthFunc](c).Instance(authFunc)
       s.Require().NoError(err)

       // Run provider.
       err = provideAuthBundle(c)
       s.Require().NoError(err)

       // AuthBundle should be registered.
       bundle, err := gaz.Resolve[*AuthBundle](c)
       s.Require().NoError(err)
       s.NotNil(bundle)
       s.Equal("auth", bundle.Name())
   }
   ```

4. Test conditional registration (AuthFunc absent):
   ```go
   func (s *InterceptorsSuite) TestProvideAuthBundle_WithoutAuthFunc() {
       c := di.New()

       // No AuthFunc registered.
       err := provideAuthBundle(c)
       s.Require().NoError(err) // Should succeed (skip silently).

       // AuthBundle should NOT be registered.
       _, err = gaz.Resolve[*AuthBundle](c)
       s.Error(err) // Not found.
   }
   ```

Add required imports if not present:
- "context"
- "github.com/petabytecl/gaz"
- "github.com/petabytecl/gaz/di"
  </action>
  <verify>
Run: `go test -race -v ./server/grpc/... -run "TestAuthBundle|TestPriorityAuth|TestProvideAuthBundle"` - all tests pass.
Run: `make lint` - no linting errors.
Run: `make cover` - coverage maintained at 90%+.
  </verify>
  <done>
All AuthBundle tests pass.
Linting passes.
Coverage threshold maintained.
  </done>
</task>

</tasks>

<verification>
1. `go build ./server/grpc/...` - package compiles
2. `go test -race ./server/grpc/...` - all tests pass
3. `make lint` - no linting issues
4. `make cover` - 90%+ coverage maintained
</verification>

<success_criteria>
1. AuthFunc type alias exported for user convenience
2. AuthBundle implements InterceptorBundle with Priority=50
3. Auth interceptor only registered when AuthFunc exists in DI
4. Auth interceptor chains correctly: logging(0) -> auth(50) -> validation(100) -> recovery(1000)
5. All tests pass, linting clean, coverage maintained
</success_criteria>

<output>
After completion, create `.planning/quick/012-add-builtin-grpc-auth-interceptor/012-SUMMARY.md`
</output>
