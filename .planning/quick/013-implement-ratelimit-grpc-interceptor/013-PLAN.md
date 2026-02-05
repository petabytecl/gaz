---
id: "013"
type: quick
title: "Implement Rate Limit gRPC Interceptor"
files_modified:
  - server/grpc/interceptors.go
  - server/grpc/module.go
  - server/grpc/interceptors_test.go
autonomous: true
---

<objective>
Implement a rate limiting gRPC interceptor bundle using go-grpc-middleware/v2/interceptors/ratelimit.

Purpose: Add rate limiting capability to gRPC servers with a sensible default (AlwaysPassLimiter) that allows users to inject custom limiters via DI.

Output: RateLimitBundle with AlwaysPassLimiter default, integrated into gRPC module.
</objective>

<context>
@server/grpc/interceptors.go (bundle patterns, priority constants)
@server/grpc/module.go (provideAuthBundle pattern for optional DI)
@server/grpc/interceptors_test.go (test patterns)
</context>

<tasks>

<task type="auto">
  <name>Task 1: Add RateLimitBundle with AlwaysPassLimiter</name>
  <files>server/grpc/interceptors.go</files>
  <action>
1. Add priority constant (after line 28, between PriorityLogging and PriorityAuth):
   ```go
   // PriorityRateLimit is the priority for the rate limit interceptor (after logging, before auth).
   PriorityRateLimit = 25
   ```

2. Add import for ratelimit package:
   ```go
   "github.com/grpc-ecosystem/go-grpc-middleware/v2/interceptors/ratelimit"
   ```

3. Add Limiter type alias (after AuthFunc type alias, around line 203):
   ```go
   // Limiter defines the interface for rate limiting.
   // Implementations should return nil to allow the request, or an error to reject it.
   //
   // Register a custom limiter in DI to override the default AlwaysPassLimiter:
   //
   //   gaz.For[grpc.Limiter](c).Instance(myLimiter)
   type Limiter = ratelimit.Limiter
   ```

4. Add AlwaysPassLimiter (after Limiter type):
   ```go
   // AlwaysPassLimiter is a no-op limiter that allows all requests.
   // This is the default limiter when no custom Limiter is registered in DI.
   type AlwaysPassLimiter struct{}

   // Limit always returns nil, allowing all requests.
   func (l AlwaysPassLimiter) Limit(_ context.Context) error {
       return nil
   }
   ```

5. Add RateLimitBundle (after AlwaysPassLimiter):
   ```go
   // RateLimitBundle is the built-in rate limiting interceptor bundle.
   // It uses the registered Limiter to control request rates.
   type RateLimitBundle struct {
       limiter Limiter
   }

   // NewRateLimitBundle creates a new rate limit interceptor bundle.
   func NewRateLimitBundle(limiter Limiter) *RateLimitBundle {
       if limiter == nil {
           limiter = AlwaysPassLimiter{}
       }
       return &RateLimitBundle{limiter: limiter}
   }

   // Name returns the bundle identifier.
   func (b *RateLimitBundle) Name() string {
       return "ratelimit"
   }

   // Priority returns the rate limit priority (after logging, before auth).
   func (b *RateLimitBundle) Priority() int {
       return PriorityRateLimit
   }

   // Interceptors returns the rate limit interceptors.
   func (b *RateLimitBundle) Interceptors() (grpc.UnaryServerInterceptor, grpc.StreamServerInterceptor) {
       return ratelimit.UnaryServerInterceptor(b.limiter),
           ratelimit.StreamServerInterceptor(b.limiter)
   }
   ```
  </action>
  <verify>go build ./server/grpc/...</verify>
  <done>RateLimitBundle, AlwaysPassLimiter, Limiter type, and PriorityRateLimit constant exist in interceptors.go</done>
</task>

<task type="auto">
  <name>Task 2: Add DI registration and tests</name>
  <files>server/grpc/module.go, server/grpc/interceptors_test.go</files>
  <action>
**module.go changes:**

1. Add provideRateLimitBundle function (after provideAuthBundle, around line 96):
   ```go
   // provideRateLimitBundle creates a RateLimitBundle provider function.
   // If a Limiter is registered in DI, it uses that limiter.
   // Otherwise, it registers a bundle with AlwaysPassLimiter (allows all requests).
   func provideRateLimitBundle(c *gaz.Container) error {
       var limiter Limiter
       if gaz.Has[Limiter](c) {
           resolved, resolveErr := gaz.Resolve[Limiter](c)
           if resolveErr != nil {
               return fmt.Errorf("resolve limiter: %w", resolveErr)
           }
           limiter = resolved
       }
       // limiter is nil if not registered, NewRateLimitBundle handles this

       if regErr := gaz.For[*RateLimitBundle](c).Provider(func(_ *gaz.Container) (*RateLimitBundle, error) {
           return NewRateLimitBundle(limiter), nil
       }); regErr != nil {
           return fmt.Errorf("register ratelimit bundle: %w", regErr)
       }
       return nil
   }
   ```

2. Add to NewModule chain (after provideLoggingBundle, before provideAuthBundle):
   ```go
   Provide(provideRateLimitBundle).
   ```

3. Update NewModule docstring to include RateLimitBundle in the components list.

**interceptors_test.go changes:**

Add tests at end of file:
```go
func (s *InterceptorBundleTestSuite) TestAlwaysPassLimiter_AllowsAllRequests() {
    limiter := AlwaysPassLimiter{}
    err := limiter.Limit(context.Background())
    s.NoError(err)
}

func (s *InterceptorBundleTestSuite) TestRateLimitBundle_ImplementsInterface() {
    bundle := NewRateLimitBundle(nil) // Uses AlwaysPassLimiter

    // Verify interface compliance.
    var _ InterceptorBundle = bundle

    s.Equal("ratelimit", bundle.Name())
    s.Equal(PriorityRateLimit, bundle.Priority())

    unary, stream := bundle.Interceptors()
    s.NotNil(unary)
    s.NotNil(stream)
}

func (s *InterceptorBundleTestSuite) TestRateLimitBundle_WithCustomLimiter() {
    customLimiter := &mockLimiter{shouldReject: false}
    bundle := NewRateLimitBundle(customLimiter)

    s.Equal("ratelimit", bundle.Name())
    unary, stream := bundle.Interceptors()
    s.NotNil(unary)
    s.NotNil(stream)
}

func (s *InterceptorBundleTestSuite) TestPriorityRateLimit_Ordering() {
    // RateLimit should be after logging (0), before auth (50).
    s.Greater(PriorityRateLimit, PriorityLogging)
    s.Less(PriorityRateLimit, PriorityAuth)
}

func (s *InterceptorBundleTestSuite) TestProvideRateLimitBundle_WithoutLimiter() {
    c := di.New()

    // No Limiter registered - should use AlwaysPassLimiter.
    err := provideRateLimitBundle(c)
    s.Require().NoError(err)

    // RateLimitBundle should be registered.
    bundle, err := gaz.Resolve[*RateLimitBundle](c)
    s.Require().NoError(err)
    s.NotNil(bundle)
    s.Equal("ratelimit", bundle.Name())
}

func (s *InterceptorBundleTestSuite) TestProvideRateLimitBundle_WithLimiter() {
    c := di.New()

    // Register custom Limiter.
    customLimiter := Limiter(&mockLimiter{shouldReject: false})
    err := gaz.For[Limiter](c).Instance(customLimiter)
    s.Require().NoError(err)

    // Run provider.
    err = provideRateLimitBundle(c)
    s.Require().NoError(err)

    // RateLimitBundle should be registered.
    bundle, err := gaz.Resolve[*RateLimitBundle](c)
    s.Require().NoError(err)
    s.NotNil(bundle)
}

// mockLimiter is a test double for Limiter.
type mockLimiter struct {
    shouldReject bool
}

func (m *mockLimiter) Limit(_ context.Context) error {
    if m.shouldReject {
        return status.Error(codes.ResourceExhausted, "rate limit exceeded")
    }
    return nil
}
```
  </action>
  <verify>go test -race -v ./server/grpc/... && make lint</verify>
  <done>provideRateLimitBundle registered in NewModule, all tests pass, linter clean</done>
</task>

</tasks>

<verification>
1. `go build ./server/grpc/...` compiles without errors
2. `go test -race -v ./server/grpc/...` all tests pass
3. `make lint` no linter errors
4. Priority ordering: Logging (0) < RateLimit (25) < Auth (50) < Validation (100) < Recovery (1000)
</verification>

<success_criteria>
- RateLimitBundle registered as built-in interceptor
- AlwaysPassLimiter used when no Limiter in DI
- Custom Limiter from DI used when registered
- PriorityRateLimit = 25 (between logging and auth)
- All existing tests still pass
- New tests cover bundle implementation and DI registration
</success_criteria>

<output>
After completion, create `.planning/quick/013-implement-ratelimit-grpc-interceptor/013-SUMMARY.md`
</output>
