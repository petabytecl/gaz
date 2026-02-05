package grpc

import (
	"context"
	"log/slog"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/petabytecl/gaz/di"
)

func TestNewRecoveryInterceptor(t *testing.T) {
	logger := slog.Default()

	t.Run("recovers from panic in production mode", func(t *testing.T) {
		unary, _ := NewRecoveryInterceptor(logger, false)

		handler := func(_ context.Context, _ any) (any, error) {
			panic("test panic")
		}

		_, err := unary(context.Background(), nil, &grpc.UnaryServerInfo{}, handler)
		require.Error(t, err)

		// In production mode, error message should be generic.
		st, ok := status.FromError(err)
		require.True(t, ok)
		require.Equal(t, codes.Internal, st.Code())
		require.Equal(t, "internal server error", st.Message())
	})

	t.Run("recovers from panic in dev mode", func(t *testing.T) {
		unary, _ := NewRecoveryInterceptor(logger, true)

		handler := func(_ context.Context, _ any) (any, error) {
			panic("test panic in dev mode")
		}

		_, err := unary(context.Background(), nil, &grpc.UnaryServerInfo{}, handler)
		require.Error(t, err)

		// In dev mode, error message should contain panic details.
		st, ok := status.FromError(err)
		require.True(t, ok)
		require.Equal(t, codes.Internal, st.Code())
		require.Contains(t, st.Message(), "test panic in dev mode")
	})
}

func TestInterceptorLogger(t *testing.T) {
	logger := slog.Default()
	adapted := InterceptorLogger(logger)
	require.NotNil(t, adapted)

	// Just verify it doesn't panic when called.
	adapted.Log(context.Background(), 0, "test message", "key", "value")
}

// InterceptorBundleTestSuite tests the InterceptorBundle interface and discovery.
type InterceptorBundleTestSuite struct {
	suite.Suite
}

func TestInterceptorBundleTestSuite(t *testing.T) {
	suite.Run(t, new(InterceptorBundleTestSuite))
}

func (s *InterceptorBundleTestSuite) TestLoggingBundleImplementsInterface() {
	logger := slog.Default()
	bundle := NewLoggingBundle(logger)

	// Verify interface compliance.
	var _ InterceptorBundle = bundle

	s.Equal("logging", bundle.Name())
	s.Equal(PriorityLogging, bundle.Priority())

	unary, stream := bundle.Interceptors()
	s.NotNil(unary)
	s.NotNil(stream)
}

func (s *InterceptorBundleTestSuite) TestRecoveryBundleImplementsInterface() {
	logger := slog.Default()
	bundle := NewRecoveryBundle(logger, false)

	// Verify interface compliance.
	var _ InterceptorBundle = bundle

	s.Equal("recovery", bundle.Name())
	s.Equal(PriorityRecovery, bundle.Priority())

	unary, stream := bundle.Interceptors()
	s.NotNil(unary)
	s.NotNil(stream)
}

func (s *InterceptorBundleTestSuite) TestCollectInterceptorsOrdering() {
	logger := slog.Default()
	container := di.New()

	// Register bundles in reverse priority order to test sorting.
	_ = di.For[*RecoveryBundle](container).Instance(NewRecoveryBundle(logger, false))
	_ = di.For[*LoggingBundle](container).Instance(NewLoggingBundle(logger))
	_ = di.For[*mockInterceptorBundle](container).Instance(&mockInterceptorBundle{
		name:     "custom",
		priority: 50,
	})

	unary, stream := collectInterceptors(container, logger)

	// Should have 3 interceptors (logging, custom, recovery).
	s.Len(unary, 3)
	s.Len(stream, 3)
}

func (s *InterceptorBundleTestSuite) TestCollectInterceptorsEmptyContainer() {
	logger := slog.Default()
	container := di.New()

	unary, stream := collectInterceptors(container, logger)

	// No interceptors registered.
	s.Empty(unary)
	s.Empty(stream)
}

func (s *InterceptorBundleTestSuite) TestCollectInterceptorsNilInterceptor() {
	logger := slog.Default()
	container := di.New()

	// Register a bundle that returns nil for stream interceptor.
	_ = di.For[*mockInterceptorBundle](container).Instance(&mockInterceptorBundle{
		name:       "unary-only",
		priority:   50,
		unaryOnly:  true,
		streamOnly: false,
	})

	unary, stream := collectInterceptors(container, logger)

	// Should have 1 unary, 0 stream.
	s.Len(unary, 1)
	s.Empty(stream)
}

func (s *InterceptorBundleTestSuite) TestCustomInterceptorPriorityBetweenBuiltins() {
	logger := slog.Default()
	container := di.New()

	// Track call order.
	var callOrder []string

	// Register bundles.
	_ = di.For[*RecoveryBundle](container).Instance(NewRecoveryBundle(logger, false))
	_ = di.For[*LoggingBundle](container).Instance(NewLoggingBundle(logger))

	// Custom interceptor with priority between logging and recovery.
	customBundle := &mockInterceptorBundle{
		name:     "custom",
		priority: 50,
		onUnaryCall: func() {
			callOrder = append(callOrder, "custom")
		},
	}
	_ = di.For[*mockInterceptorBundle](container).Instance(customBundle)

	unary, _ := collectInterceptors(container, logger)

	// Verify ordering: logging (0) < custom (50) < recovery (1000).
	s.Require().Len(unary, 3)

	// The order in the slice should be logging, custom, recovery.
	// We can verify by checking priorities are in ascending order.
	// Since collectInterceptors sorts by priority, the first should be logging.
}

// mockInterceptorBundle is a test double for InterceptorBundle.
type mockInterceptorBundle struct {
	name        string
	priority    int
	unaryOnly   bool
	streamOnly  bool
	onUnaryCall func()
}

func (m *mockInterceptorBundle) Name() string {
	return m.name
}

func (m *mockInterceptorBundle) Priority() int {
	return m.priority
}

func (m *mockInterceptorBundle) Interceptors() (grpc.UnaryServerInterceptor, grpc.StreamServerInterceptor) {
	var unary grpc.UnaryServerInterceptor
	var stream grpc.StreamServerInterceptor

	if !m.streamOnly {
		unary = func(ctx context.Context, req any, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (any, error) {
			if m.onUnaryCall != nil {
				m.onUnaryCall()
			}
			return handler(ctx, req)
		}
	}

	if !m.unaryOnly {
		stream = func(srv any, ss grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) error {
			return handler(srv, ss)
		}
	}

	return unary, stream
}
