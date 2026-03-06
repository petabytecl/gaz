package connect

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"testing"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/suite"

	"github.com/petabytecl/gaz/di"
)

// ConnectInterceptorBundleTestSuite tests the ConnectInterceptorBundle interface and discovery.
type ConnectInterceptorBundleTestSuite struct {
	suite.Suite
}

func TestConnectInterceptorBundleTestSuite(t *testing.T) {
	suite.Run(t, new(ConnectInterceptorBundleTestSuite))
}

func (s *ConnectInterceptorBundleTestSuite) TestPriorityConstants() {
	// Priority ordering: Logging(0) < RateLimit(25) < Auth(50) < Validation(100) < Recovery(1000).
	s.Equal(0, PriorityLogging)
	s.Equal(25, PriorityRateLimit)
	s.Equal(50, PriorityAuth)
	s.Equal(100, PriorityValidation)
	s.Equal(1000, PriorityRecovery)

	s.Less(PriorityLogging, PriorityRateLimit)
	s.Less(PriorityRateLimit, PriorityAuth)
	s.Less(PriorityAuth, PriorityValidation)
	s.Less(PriorityValidation, PriorityRecovery)
}

func (s *ConnectInterceptorBundleTestSuite) TestLoggingBundle_ImplementsInterface() {
	logger := slog.Default()
	bundle := NewLoggingBundle(logger)

	// Compile-time interface compliance.
	var _ ConnectInterceptorBundle = bundle

	s.Equal("logging", bundle.Name())
	s.Equal(PriorityLogging, bundle.Priority())

	interceptors := bundle.Interceptors()
	s.NotEmpty(interceptors)
}

func (s *ConnectInterceptorBundleTestSuite) TestLoggingBundle_NilLogger() {
	bundle := NewLoggingBundle(nil)
	s.NotNil(bundle)
	s.Equal("logging", bundle.Name())

	interceptors := bundle.Interceptors()
	s.NotEmpty(interceptors)
}

func (s *ConnectInterceptorBundleTestSuite) TestRecoveryBundle_ImplementsInterface() {
	logger := slog.Default()
	bundle := NewRecoveryBundle(logger, false)

	// Compile-time interface compliance.
	var _ ConnectInterceptorBundle = bundle

	s.Equal("recovery", bundle.Name())
	s.Equal(PriorityRecovery, bundle.Priority())

	interceptors := bundle.Interceptors()
	s.NotEmpty(interceptors)
}

func (s *ConnectInterceptorBundleTestSuite) TestRecoveryBundle_PanicInUnary_ProductionMode() {
	logger := slog.Default()
	bundle := NewRecoveryBundle(logger, false)

	interceptors := bundle.Interceptors()
	s.Require().NotEmpty(interceptors)

	interceptor := interceptors[0]

	// Wrap a handler that panics.
	wrappedFunc := interceptor.WrapUnary(func(_ context.Context, _ connect.AnyRequest) (connect.AnyResponse, error) {
		panic("test panic")
	})

	resp, err := wrappedFunc(context.Background(), nil)
	s.Nil(resp)
	s.Require().Error(err)

	// In production mode, error should be generic.
	var connectErr *connect.Error
	s.Require().True(errors.As(err, &connectErr))
	s.Equal(connect.CodeInternal, connectErr.Code())
	s.Equal("internal server error", connectErr.Message())
}

func (s *ConnectInterceptorBundleTestSuite) TestRecoveryBundle_PanicInUnary_DevMode() {
	logger := slog.Default()
	bundle := NewRecoveryBundle(logger, true)

	interceptors := bundle.Interceptors()
	s.Require().NotEmpty(interceptors)

	interceptor := interceptors[0]

	// Wrap a handler that panics.
	wrappedFunc := interceptor.WrapUnary(func(_ context.Context, _ connect.AnyRequest) (connect.AnyResponse, error) {
		panic("dev mode panic details")
	})

	resp, err := wrappedFunc(context.Background(), nil)
	s.Nil(resp)
	s.Require().Error(err)

	// In dev mode, error should contain panic details.
	var connectErr *connect.Error
	s.Require().True(errors.As(err, &connectErr))
	s.Equal(connect.CodeInternal, connectErr.Code())
	s.Contains(connectErr.Message(), "dev mode panic details")
}

func (s *ConnectInterceptorBundleTestSuite) TestRecoveryBundle_PanicInStreamingHandler() {
	logger := slog.Default()
	bundle := NewRecoveryBundle(logger, false)

	interceptors := bundle.Interceptors()
	s.Require().NotEmpty(interceptors)

	interceptor := interceptors[0]

	// Wrap a streaming handler that panics.
	wrappedFunc := interceptor.WrapStreamingHandler(func(_ context.Context, _ connect.StreamingHandlerConn) error {
		panic("stream panic")
	})

	err := wrappedFunc(context.Background(), nil)
	s.Require().Error(err)

	// Should return connect.CodeInternal.
	var connectErr *connect.Error
	s.Require().True(errors.As(err, &connectErr))
	s.Equal(connect.CodeInternal, connectErr.Code())
}

func (s *ConnectInterceptorBundleTestSuite) TestRecoveryBundle_NoPanic() {
	logger := slog.Default()
	bundle := NewRecoveryBundle(logger, false)

	interceptors := bundle.Interceptors()
	s.Require().NotEmpty(interceptors)

	interceptor := interceptors[0]

	// Wrap a handler that does NOT panic.
	wrappedFunc := interceptor.WrapUnary(func(_ context.Context, _ connect.AnyRequest) (connect.AnyResponse, error) {
		return nil, nil
	})

	resp, err := wrappedFunc(context.Background(), nil)
	s.Nil(resp)
	s.NoError(err)
}

func (s *ConnectInterceptorBundleTestSuite) TestRecoveryBundle_NilLogger() {
	bundle := NewRecoveryBundle(nil, false)
	s.NotNil(bundle)
	s.Equal("recovery", bundle.Name())
}

func (s *ConnectInterceptorBundleTestSuite) TestAuthBundle_ImplementsInterface() {
	authFunc := ConnectAuthFunc(func(_ context.Context, _ connect.AnyRequest) (context.Context, error) {
		return context.Background(), nil
	})
	bundle := NewAuthBundle(authFunc)

	// Compile-time interface compliance.
	var _ ConnectInterceptorBundle = bundle

	s.Equal("auth", bundle.Name())
	s.Equal(PriorityAuth, bundle.Priority())

	interceptors := bundle.Interceptors()
	s.NotEmpty(interceptors)
}

func (s *ConnectInterceptorBundleTestSuite) TestRateLimitBundle_ImplementsInterface() {
	bundle := NewRateLimitBundle(nil) // Uses AlwaysPassLimiter.

	// Compile-time interface compliance.
	var _ ConnectInterceptorBundle = bundle

	s.Equal("ratelimit", bundle.Name())
	s.Equal(PriorityRateLimit, bundle.Priority())

	interceptors := bundle.Interceptors()
	s.NotEmpty(interceptors)
}

func (s *ConnectInterceptorBundleTestSuite) TestRateLimitBundle_WithNilLimiter_UsesAlwaysPass() {
	bundle := NewRateLimitBundle(nil)

	interceptors := bundle.Interceptors()
	s.Require().NotEmpty(interceptors)

	interceptor := interceptors[0]

	// Should pass through without error.
	wrappedFunc := interceptor.WrapUnary(func(_ context.Context, _ connect.AnyRequest) (connect.AnyResponse, error) {
		return nil, nil
	})

	_, err := wrappedFunc(context.Background(), nil)
	s.NoError(err)
}

func (s *ConnectInterceptorBundleTestSuite) TestRateLimitBundle_WithCustomLimiter() {
	limiter := &mockConnectLimiter{shouldReject: true}
	bundle := NewRateLimitBundle(limiter)

	interceptors := bundle.Interceptors()
	s.Require().NotEmpty(interceptors)

	interceptor := interceptors[0]

	// Should reject with rate limit error.
	wrappedFunc := interceptor.WrapUnary(func(_ context.Context, _ connect.AnyRequest) (connect.AnyResponse, error) {
		return nil, nil
	})

	_, err := wrappedFunc(context.Background(), nil)
	s.Require().Error(err)

	var connectErr *connect.Error
	s.Require().True(errors.As(err, &connectErr))
	s.Equal(connect.CodeResourceExhausted, connectErr.Code())
}

func (s *ConnectInterceptorBundleTestSuite) TestAlwaysPassLimiter_AllowsAllRequests() {
	limiter := AlwaysPassLimiter{}
	err := limiter.Limit(context.Background(), nil)
	s.NoError(err)
}

func (s *ConnectInterceptorBundleTestSuite) TestValidationBundle_ImplementsInterface() {
	bundle := NewValidationBundle()

	// Compile-time interface compliance.
	var _ ConnectInterceptorBundle = bundle

	s.Equal("validation", bundle.Name())
	s.Equal(PriorityValidation, bundle.Priority())

	interceptors := bundle.Interceptors()
	s.NotEmpty(interceptors)
}

func (s *ConnectInterceptorBundleTestSuite) TestCollectConnectInterceptors_Ordering() {
	logger := slog.Default()
	container := di.New()

	// Register bundles in reverse priority order to test sorting.
	_ = di.For[*RecoveryBundle](container).Instance(NewRecoveryBundle(logger, false))
	_ = di.For[*LoggingBundle](container).Instance(NewLoggingBundle(logger))
	_ = di.For[*ValidationBundle](container).Instance(NewValidationBundle())
	_ = di.For[*mockConnectInterceptorBundle](container).Instance(&mockConnectInterceptorBundle{
		name:     "custom",
		priority: 500,
	})

	interceptors := collectConnectInterceptors(container, logger)

	// Should have interceptors from all 4 bundles.
	// Logging (1) + Recovery (1) + Validation (1) + custom (1) = 4.
	s.Len(interceptors, 4)
}

func (s *ConnectInterceptorBundleTestSuite) TestCollectConnectInterceptors_EmptyContainer() {
	logger := slog.Default()
	container := di.New()

	interceptors := collectConnectInterceptors(container, logger)

	s.Nil(interceptors)
}

func (s *ConnectInterceptorBundleTestSuite) TestCollectConnectInterceptors_FlattensMultiInterceptorBundles() {
	logger := slog.Default()
	container := di.New()

	// Register a bundle that returns multiple interceptors.
	_ = di.For[*mockConnectInterceptorBundle](container).Instance(&mockConnectInterceptorBundle{
		name:             "multi",
		priority:         50,
		interceptorCount: 3,
	})

	interceptors := collectConnectInterceptors(container, logger)

	// Should flatten 3 interceptors from the single bundle.
	s.Len(interceptors, 3)
}

func (s *ConnectInterceptorBundleTestSuite) TestCollectConnectInterceptors_SortsByPriority() {
	logger := slog.Default()
	container := di.New()

	var callOrder []string

	// Register bundles in reverse priority order.
	_ = di.For[*highPriorityBundle](container).Instance(&highPriorityBundle{
		name:     "high",
		priority: 1000,
		onCall: func() {
			callOrder = append(callOrder, "high")
		},
	})
	_ = di.For[*lowPriorityBundle](container).Instance(&lowPriorityBundle{
		name:     "low",
		priority: 0,
		onCall: func() {
			callOrder = append(callOrder, "low")
		},
	})
	_ = di.For[*midPriorityBundle](container).Instance(&midPriorityBundle{
		name:     "mid",
		priority: 50,
		onCall: func() {
			callOrder = append(callOrder, "mid")
		},
	})

	interceptors := collectConnectInterceptors(container, logger)
	s.Require().Len(interceptors, 3)

	// Execute all interceptors in order to verify they're sorted.
	for _, i := range interceptors {
		wrappedFunc := i.WrapUnary(func(_ context.Context, _ connect.AnyRequest) (connect.AnyResponse, error) {
			return nil, nil
		})
		_, _ = wrappedFunc(context.Background(), nil)
	}

	s.Equal([]string{"low", "mid", "high"}, callOrder)
}

// mockConnectLimiter is a test double for ConnectLimiter.
type mockConnectLimiter struct {
	shouldReject bool
}

func (m *mockConnectLimiter) Limit(_ context.Context, _ connect.AnyRequest) error {
	if m.shouldReject {
		return connect.NewError(connect.CodeResourceExhausted, fmt.Errorf("rate limit exceeded"))
	}
	return nil
}

// mockConnectInterceptorBundle is a test double for ConnectInterceptorBundle.
type mockConnectInterceptorBundle struct {
	name             string
	priority         int
	interceptorCount int
}

func (m *mockConnectInterceptorBundle) Name() string {
	return m.name
}

func (m *mockConnectInterceptorBundle) Priority() int {
	return m.priority
}

func (m *mockConnectInterceptorBundle) Interceptors() []connect.Interceptor {
	count := m.interceptorCount
	if count == 0 {
		count = 1
	}

	interceptors := make([]connect.Interceptor, count)
	for i := range count {
		interceptors[i] = &noopInterceptor{}
	}
	return interceptors
}

// noopInterceptor is a pass-through Connect interceptor for testing.
type noopInterceptor struct{}

func (n *noopInterceptor) WrapUnary(next connect.UnaryFunc) connect.UnaryFunc {
	return next
}

func (n *noopInterceptor) WrapStreamingClient(next connect.StreamingClientFunc) connect.StreamingClientFunc {
	return next
}

func (n *noopInterceptor) WrapStreamingHandler(next connect.StreamingHandlerFunc) connect.StreamingHandlerFunc {
	return next
}

// Priority-tracking bundles for sorting tests.
type highPriorityBundle struct {
	name     string
	priority int
	onCall   func()
}

func (b *highPriorityBundle) Name() string  { return b.name }
func (b *highPriorityBundle) Priority() int { return b.priority }
func (b *highPriorityBundle) Interceptors() []connect.Interceptor {
	return []connect.Interceptor{&trackingInterceptor{onCall: b.onCall}}
}

type midPriorityBundle struct {
	name     string
	priority int
	onCall   func()
}

func (b *midPriorityBundle) Name() string  { return b.name }
func (b *midPriorityBundle) Priority() int { return b.priority }
func (b *midPriorityBundle) Interceptors() []connect.Interceptor {
	return []connect.Interceptor{&trackingInterceptor{onCall: b.onCall}}
}

type lowPriorityBundle struct {
	name     string
	priority int
	onCall   func()
}

func (b *lowPriorityBundle) Name() string  { return b.name }
func (b *lowPriorityBundle) Priority() int { return b.priority }
func (b *lowPriorityBundle) Interceptors() []connect.Interceptor {
	return []connect.Interceptor{&trackingInterceptor{onCall: b.onCall}}
}

// trackingInterceptor records when WrapUnary is called for ordering tests.
type trackingInterceptor struct {
	onCall func()
}

func (t *trackingInterceptor) WrapUnary(next connect.UnaryFunc) connect.UnaryFunc {
	return func(ctx context.Context, req connect.AnyRequest) (connect.AnyResponse, error) {
		if t.onCall != nil {
			t.onCall()
		}
		return next(ctx, req)
	}
}

func (t *trackingInterceptor) WrapStreamingClient(next connect.StreamingClientFunc) connect.StreamingClientFunc {
	return next
}

func (t *trackingInterceptor) WrapStreamingHandler(next connect.StreamingHandlerFunc) connect.StreamingHandlerFunc {
	return next
}
