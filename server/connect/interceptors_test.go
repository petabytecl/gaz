package connect

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"testing"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/suite"

	"github.com/petabytecl/gaz/di"
)

// InterceptorBundleTestSuite tests the InterceptorBundle interface and discovery.
type InterceptorBundleTestSuite struct {
	suite.Suite
}

func TestInterceptorBundleTestSuite(t *testing.T) {
	suite.Run(t, new(InterceptorBundleTestSuite))
}

func (s *InterceptorBundleTestSuite) TestPriorityConstants() {
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

func (s *InterceptorBundleTestSuite) TestLoggingBundle_ImplementsInterface() {
	logger := slog.Default()
	bundle := NewLoggingBundle(logger)

	// Compile-time interface compliance.
	var _ InterceptorBundle = bundle

	s.Equal("logging", bundle.Name())
	s.Equal(PriorityLogging, bundle.Priority())

	interceptors := bundle.Interceptors()
	s.NotEmpty(interceptors)
}

func (s *InterceptorBundleTestSuite) TestLoggingBundle_NilLogger() {
	bundle := NewLoggingBundle(nil)
	s.NotNil(bundle)
	s.Equal("logging", bundle.Name())

	interceptors := bundle.Interceptors()
	s.NotEmpty(interceptors)
}

func (s *InterceptorBundleTestSuite) TestRecoveryBundle_ImplementsInterface() {
	logger := slog.Default()
	bundle := NewRecoveryBundle(logger, false)

	// Compile-time interface compliance.
	var _ InterceptorBundle = bundle

	s.Equal("recovery", bundle.Name())
	s.Equal(PriorityRecovery, bundle.Priority())

	interceptors := bundle.Interceptors()
	s.NotEmpty(interceptors)
}

func (s *InterceptorBundleTestSuite) TestRecoveryBundle_PanicInUnary_ProductionMode() {
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

func (s *InterceptorBundleTestSuite) TestRecoveryBundle_PanicInUnary_DevMode() {
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

func (s *InterceptorBundleTestSuite) TestRecoveryBundle_PanicInStreamingHandler() {
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

func (s *InterceptorBundleTestSuite) TestRecoveryBundle_NoPanic() {
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

func (s *InterceptorBundleTestSuite) TestRecoveryBundle_NilLogger() {
	bundle := NewRecoveryBundle(nil, false)
	s.NotNil(bundle)
	s.Equal("recovery", bundle.Name())
}

func (s *InterceptorBundleTestSuite) TestAuthBundle_ImplementsInterface() {
	authFunc := AuthFunc(func(_ context.Context, _ http.Header, _ connect.Spec) (context.Context, error) {
		return context.Background(), nil
	})
	bundle := NewAuthBundle(authFunc)

	// Compile-time interface compliance.
	var _ InterceptorBundle = bundle

	s.Equal("auth", bundle.Name())
	s.Equal(PriorityAuth, bundle.Priority())

	interceptors := bundle.Interceptors()
	s.NotEmpty(interceptors)
}

func (s *InterceptorBundleTestSuite) TestRateLimitBundle_ImplementsInterface() {
	bundle := NewRateLimitBundle(nil) // Uses AlwaysPassLimiter.

	// Compile-time interface compliance.
	var _ InterceptorBundle = bundle

	s.Equal("ratelimit", bundle.Name())
	s.Equal(PriorityRateLimit, bundle.Priority())

	interceptors := bundle.Interceptors()
	s.NotEmpty(interceptors)
}

func (s *InterceptorBundleTestSuite) TestRateLimitBundle_WithNilLimiter_UsesAlwaysPass() {
	bundle := NewRateLimitBundle(nil)

	interceptors := bundle.Interceptors()
	s.Require().NotEmpty(interceptors)

	interceptor := interceptors[0]

	// Should pass through without error.
	req := connect.NewRequest[any](nil)
	wrappedFunc := interceptor.WrapUnary(func(_ context.Context, _ connect.AnyRequest) (connect.AnyResponse, error) {
		return nil, nil
	})

	_, err := wrappedFunc(context.Background(), req)
	s.NoError(err)
}

func (s *InterceptorBundleTestSuite) TestRateLimitBundle_WithCustomLimiter() {
	limiter := &mockLimiter{shouldReject: true}
	bundle := NewRateLimitBundle(limiter)

	interceptors := bundle.Interceptors()
	s.Require().NotEmpty(interceptors)

	interceptor := interceptors[0]

	// Should reject with rate limit error.
	req := connect.NewRequest[any](nil)
	wrappedFunc := interceptor.WrapUnary(func(_ context.Context, _ connect.AnyRequest) (connect.AnyResponse, error) {
		return nil, nil
	})

	_, err := wrappedFunc(context.Background(), req)
	s.Require().Error(err)

	var connectErr *connect.Error
	s.Require().True(errors.As(err, &connectErr))
	s.Equal(connect.CodeResourceExhausted, connectErr.Code())
}

func (s *InterceptorBundleTestSuite) TestAlwaysPassLimiter_AllowsAllRequests() {
	limiter := AlwaysPassLimiter{}
	err := limiter.Limit(context.Background(), nil, connect.Spec{})
	s.NoError(err)
}

// --- Logging Wrap* method tests ---

func (s *InterceptorBundleTestSuite) TestLoggingInterceptor_WrapUnary_Success() {
	bundle := NewLoggingBundle(slog.Default())
	interceptor := bundle.Interceptors()[0]

	called := false
	wrappedFunc := interceptor.WrapUnary(func(_ context.Context, _ connect.AnyRequest) (connect.AnyResponse, error) {
		called = true
		return nil, nil
	})

	// Call with nil request (procedure check handles nil).
	resp, err := wrappedFunc(context.Background(), nil)
	s.True(called, "next handler should be called")
	s.Nil(resp)
	s.NoError(err)
}

func (s *InterceptorBundleTestSuite) TestLoggingInterceptor_WrapUnary_WithRequest() {
	bundle := NewLoggingBundle(slog.Default())
	interceptor := bundle.Interceptors()[0]

	req := connect.NewRequest[any](nil)
	wrappedFunc := interceptor.WrapUnary(func(_ context.Context, _ connect.AnyRequest) (connect.AnyResponse, error) {
		return nil, nil
	})

	resp, err := wrappedFunc(context.Background(), req)
	s.Nil(resp)
	s.NoError(err)
}

func (s *InterceptorBundleTestSuite) TestLoggingInterceptor_WrapUnary_Error() {
	bundle := NewLoggingBundle(slog.Default())
	interceptor := bundle.Interceptors()[0]

	expectedErr := connect.NewError(connect.CodeInternal, errors.New("handler error"))
	wrappedFunc := interceptor.WrapUnary(func(_ context.Context, _ connect.AnyRequest) (connect.AnyResponse, error) {
		return nil, expectedErr
	})

	resp, err := wrappedFunc(context.Background(), nil)
	s.Nil(resp)
	s.Require().Error(err)
	s.ErrorIs(err, expectedErr)
}

func (s *InterceptorBundleTestSuite) TestLoggingInterceptor_WrapStreamingClient_IsPassthrough() {
	bundle := NewLoggingBundle(slog.Default())
	interceptor := bundle.Interceptors()[0]

	next := connect.StreamingClientFunc(func(_ context.Context, _ connect.Spec) connect.StreamingClientConn {
		return nil
	})

	// WrapStreamingClient should return next unchanged.
	result := interceptor.WrapStreamingClient(next)
	s.NotNil(result)
}

func (s *InterceptorBundleTestSuite) TestLoggingInterceptor_WrapStreamingHandler_Success() {
	bundle := NewLoggingBundle(slog.Default())
	interceptor := bundle.Interceptors()[0]

	called := false
	wrappedFunc := interceptor.WrapStreamingHandler(func(_ context.Context, _ connect.StreamingHandlerConn) error {
		called = true
		return nil
	})

	// Call with a mock conn for procedure logging.
	err := wrappedFunc(context.Background(), &mockStreamingHandlerConn{})
	s.True(called, "next handler should be called")
	s.NoError(err)
}

func (s *InterceptorBundleTestSuite) TestLoggingInterceptor_WrapStreamingHandler_Error() {
	bundle := NewLoggingBundle(slog.Default())
	interceptor := bundle.Interceptors()[0]

	expectedErr := connect.NewError(connect.CodeInternal, errors.New("stream error"))
	wrappedFunc := interceptor.WrapStreamingHandler(func(_ context.Context, _ connect.StreamingHandlerConn) error {
		return expectedErr
	})

	err := wrappedFunc(context.Background(), nil)
	s.Require().Error(err)
	s.ErrorIs(err, expectedErr)
}

// --- Recovery Wrap* streaming tests ---

func (s *InterceptorBundleTestSuite) TestRecoveryInterceptor_WrapStreamingClient_IsPassthrough() {
	bundle := NewRecoveryBundle(slog.Default(), false)
	interceptor := bundle.Interceptors()[0]

	next := connect.StreamingClientFunc(func(_ context.Context, _ connect.Spec) connect.StreamingClientConn {
		return nil
	})

	result := interceptor.WrapStreamingClient(next)
	s.NotNil(result)
}

func (s *InterceptorBundleTestSuite) TestRecoveryInterceptor_WrapStreamingHandler_DevMode_Panic() {
	bundle := NewRecoveryBundle(slog.Default(), true)
	interceptor := bundle.Interceptors()[0]

	wrappedFunc := interceptor.WrapStreamingHandler(func(_ context.Context, _ connect.StreamingHandlerConn) error {
		panic("dev stream panic")
	})

	err := wrappedFunc(context.Background(), nil)
	s.Require().Error(err)

	var connectErr *connect.Error
	s.Require().True(errors.As(err, &connectErr))
	s.Equal(connect.CodeInternal, connectErr.Code())
	s.Contains(connectErr.Message(), "dev stream panic")
}

func (s *InterceptorBundleTestSuite) TestRecoveryInterceptor_WrapStreamingHandler_NoPanic() {
	bundle := NewRecoveryBundle(slog.Default(), false)
	interceptor := bundle.Interceptors()[0]

	wrappedFunc := interceptor.WrapStreamingHandler(func(_ context.Context, _ connect.StreamingHandlerConn) error {
		return nil
	})

	err := wrappedFunc(context.Background(), nil)
	s.NoError(err)
}

// --- Auth Wrap* method tests ---

func (s *InterceptorBundleTestSuite) TestAuthInterceptor_WrapUnary_Success() {
	type ctxKey struct{}
	authFunc := AuthFunc(func(ctx context.Context, _ http.Header, _ connect.Spec) (context.Context, error) {
		return context.WithValue(ctx, ctxKey{}, "authenticated"), nil
	})
	bundle := NewAuthBundle(authFunc)
	interceptor := bundle.Interceptors()[0]

	var receivedCtx context.Context
	req := connect.NewRequest[any](nil)
	wrappedFunc := interceptor.WrapUnary(func(ctx context.Context, _ connect.AnyRequest) (connect.AnyResponse, error) {
		receivedCtx = ctx
		return nil, nil
	})

	_, err := wrappedFunc(context.Background(), req)
	s.NoError(err)
	s.Equal("authenticated", receivedCtx.Value(ctxKey{}))
}

func (s *InterceptorBundleTestSuite) TestAuthInterceptor_WrapUnary_Failure() {
	authErr := connect.NewError(connect.CodeUnauthenticated, errors.New("invalid token"))
	authFunc := AuthFunc(func(_ context.Context, _ http.Header, _ connect.Spec) (context.Context, error) {
		return nil, authErr
	})
	bundle := NewAuthBundle(authFunc)
	interceptor := bundle.Interceptors()[0]

	nextCalled := false
	req := connect.NewRequest[any](nil)
	wrappedFunc := interceptor.WrapUnary(func(_ context.Context, _ connect.AnyRequest) (connect.AnyResponse, error) {
		nextCalled = true
		return nil, nil
	})

	_, err := wrappedFunc(context.Background(), req)
	s.Require().Error(err)
	s.False(nextCalled, "next should not be called on auth failure")

	var connectErr *connect.Error
	s.Require().True(errors.As(err, &connectErr))
	s.Equal(connect.CodeUnauthenticated, connectErr.Code())
}

func (s *InterceptorBundleTestSuite) TestAuthInterceptor_WrapStreamingClient_IsPassthrough() {
	authFunc := AuthFunc(func(ctx context.Context, _ http.Header, _ connect.Spec) (context.Context, error) {
		return ctx, nil
	})
	bundle := NewAuthBundle(authFunc)
	interceptor := bundle.Interceptors()[0]

	next := connect.StreamingClientFunc(func(_ context.Context, _ connect.Spec) connect.StreamingClientConn {
		return nil
	})

	result := interceptor.WrapStreamingClient(next)
	s.NotNil(result)
}

func (s *InterceptorBundleTestSuite) TestAuthInterceptor_WrapStreamingHandler_Success() {
	type ctxKey struct{}
	authFunc := AuthFunc(func(ctx context.Context, _ http.Header, _ connect.Spec) (context.Context, error) {
		return context.WithValue(ctx, ctxKey{}, "stream-auth"), nil
	})
	bundle := NewAuthBundle(authFunc)
	interceptor := bundle.Interceptors()[0]

	var receivedCtx context.Context
	wrappedFunc := interceptor.WrapStreamingHandler(func(ctx context.Context, _ connect.StreamingHandlerConn) error {
		receivedCtx = ctx
		return nil
	})

	err := wrappedFunc(context.Background(), &mockStreamingHandlerConn{})
	s.NoError(err)
	s.Equal("stream-auth", receivedCtx.Value(ctxKey{}))
}

func (s *InterceptorBundleTestSuite) TestAuthInterceptor_WrapStreamingHandler_Failure() {
	authErr := connect.NewError(connect.CodeUnauthenticated, errors.New("stream auth failed"))
	authFunc := AuthFunc(func(_ context.Context, _ http.Header, _ connect.Spec) (context.Context, error) {
		return nil, authErr
	})
	bundle := NewAuthBundle(authFunc)
	interceptor := bundle.Interceptors()[0]

	nextCalled := false
	wrappedFunc := interceptor.WrapStreamingHandler(func(_ context.Context, _ connect.StreamingHandlerConn) error {
		nextCalled = true
		return nil
	})

	err := wrappedFunc(context.Background(), &mockStreamingHandlerConn{})
	s.Require().Error(err)
	s.False(nextCalled, "next should not be called on auth failure")
}

// --- RateLimit Wrap* streaming tests ---

func (s *InterceptorBundleTestSuite) TestRateLimitInterceptor_WrapStreamingClient_IsPassthrough() {
	bundle := NewRateLimitBundle(nil)
	interceptor := bundle.Interceptors()[0]

	next := connect.StreamingClientFunc(func(_ context.Context, _ connect.Spec) connect.StreamingClientConn {
		return nil
	})

	result := interceptor.WrapStreamingClient(next)
	s.NotNil(result)
}

func (s *InterceptorBundleTestSuite) TestRateLimitInterceptor_WrapStreamingHandler_Allows() {
	bundle := NewRateLimitBundle(AlwaysPassLimiter{})
	interceptor := bundle.Interceptors()[0]

	called := false
	wrappedFunc := interceptor.WrapStreamingHandler(func(_ context.Context, _ connect.StreamingHandlerConn) error {
		called = true
		return nil
	})

	err := wrappedFunc(context.Background(), &mockStreamingHandlerConn{})
	s.NoError(err)
	s.True(called, "next handler should be called when rate limit allows")
}

func (s *InterceptorBundleTestSuite) TestRateLimitInterceptor_WrapStreamingHandler_Rejects() {
	limiter := &mockLimiter{shouldReject: true}
	bundle := NewRateLimitBundle(limiter)
	interceptor := bundle.Interceptors()[0]

	nextCalled := false
	wrappedFunc := interceptor.WrapStreamingHandler(func(_ context.Context, _ connect.StreamingHandlerConn) error {
		nextCalled = true
		return nil
	})

	err := wrappedFunc(context.Background(), &mockStreamingHandlerConn{})
	s.Require().Error(err)
	s.False(nextCalled, "next should not be called when rate limited")

	var connectErr *connect.Error
	s.Require().True(errors.As(err, &connectErr))
	s.Equal(connect.CodeResourceExhausted, connectErr.Code())
}

func (s *InterceptorBundleTestSuite) TestValidationBundle_ImplementsInterface() {
	bundle := NewValidationBundle()

	// Compile-time interface compliance.
	var _ InterceptorBundle = bundle

	s.Equal("validation", bundle.Name())
	s.Equal(PriorityValidation, bundle.Priority())

	interceptors := bundle.Interceptors()
	s.NotEmpty(interceptors)
}

func (s *InterceptorBundleTestSuite) TestCollectInterceptors_Ordering() {
	logger := slog.Default()
	container := di.New()

	// Register bundles in reverse priority order to test sorting.
	_ = di.For[*RecoveryBundle](container).Instance(NewRecoveryBundle(logger, false))
	_ = di.For[*LoggingBundle](container).Instance(NewLoggingBundle(logger))
	_ = di.For[*ValidationBundle](container).Instance(NewValidationBundle())
	_ = di.For[*mockInterceptorBundle](container).Instance(&mockInterceptorBundle{
		name:     "custom",
		priority: 500,
	})

	interceptors := CollectInterceptors(container, logger)

	// Should have interceptors from all 4 bundles.
	// Logging (1) + Recovery (1) + Validation (1) + custom (1) = 4.
	s.Len(interceptors, 4)
}

func (s *InterceptorBundleTestSuite) TestCollectInterceptors_EmptyContainer() {
	logger := slog.Default()
	container := di.New()

	interceptors := CollectInterceptors(container, logger)

	s.Nil(interceptors)
}

func (s *InterceptorBundleTestSuite) TestCollectInterceptors_FlattensMultiInterceptorBundles() {
	logger := slog.Default()
	container := di.New()

	// Register a bundle that returns multiple interceptors.
	_ = di.For[*mockInterceptorBundle](container).Instance(&mockInterceptorBundle{
		name:             "multi",
		priority:         50,
		interceptorCount: 3,
	})

	interceptors := CollectInterceptors(container, logger)

	// Should flatten 3 interceptors from the single bundle.
	s.Len(interceptors, 3)
}

func (s *InterceptorBundleTestSuite) TestCollectInterceptors_SortsByPriority() {
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

	interceptors := CollectInterceptors(container, logger)
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

// mockLimiter is a test double for ConnectLimiter.
type mockLimiter struct {
	shouldReject bool
}

func (m *mockLimiter) Limit(_ context.Context, _ http.Header, _ connect.Spec) error {
	if m.shouldReject {
		return connect.NewError(connect.CodeResourceExhausted, errors.New("rate limit exceeded"))
	}
	return nil
}

// mockInterceptorBundle is a test double for InterceptorBundle.
type mockInterceptorBundle struct {
	name             string
	priority         int
	interceptorCount int
}

func (m *mockInterceptorBundle) Name() string {
	return m.name
}

func (m *mockInterceptorBundle) Priority() int {
	return m.priority
}

func (m *mockInterceptorBundle) Interceptors() []connect.Interceptor {
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

// mockStreamingHandlerConn is a minimal implementation of connect.StreamingHandlerConn for testing.
type mockStreamingHandlerConn struct{}

func (m *mockStreamingHandlerConn) Spec() connect.Spec {
	return connect.Spec{Procedure: "/test.Service/Method"}
}

func (m *mockStreamingHandlerConn) Peer() connect.Peer {
	return connect.Peer{}
}

func (m *mockStreamingHandlerConn) Receive(_ any) error {
	return nil
}

func (m *mockStreamingHandlerConn) RequestHeader() http.Header {
	return http.Header{}
}

func (m *mockStreamingHandlerConn) Send(_ any) error {
	return nil
}

func (m *mockStreamingHandlerConn) ResponseHeader() http.Header {
	return http.Header{}
}

func (m *mockStreamingHandlerConn) ResponseTrailer() http.Header {
	return http.Header{}
}
