package health

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/health"
	healthpb "google.golang.org/grpc/health/grpc_health_v1"

	"github.com/petabytecl/gaz/health/internal"
)

// DefaultGRPCCheckInterval is the default interval for checking Manager status.
const DefaultGRPCCheckInterval = 5 * time.Second

// GRPCServer wraps grpc-go's health.Server and syncs status from the Manager.
// It polls the Manager's readiness checks at a configurable interval and updates
// the gRPC health status accordingly.
//
// GRPCServer implements di.Starter and di.Stopper for lifecycle management.
type GRPCServer struct {
	health   *health.Server
	manager  *Manager
	logger   *slog.Logger
	interval time.Duration

	mu         sync.Mutex
	lastStatus healthpb.HealthCheckResponse_ServingStatus
	stopCh     chan struct{}
	stopped    chan struct{}
}

// NewGRPCServer creates a new gRPC health server.
// The server wraps grpc-go's built-in health.Server and syncs status from the Manager.
//
// The server starts with UNKNOWN status and polls the Manager's readiness checks
// at the configured interval, updating the gRPC health status accordingly:
//   - All checks pass -> SERVING
//   - Any check fails -> NOT_SERVING
func NewGRPCServer(manager *Manager, logger *slog.Logger, opts ...GRPCServerOption) *GRPCServer {
	s := &GRPCServer{
		health:     health.NewServer(),
		manager:    manager,
		logger:     logger,
		interval:   DefaultGRPCCheckInterval,
		lastStatus: healthpb.HealthCheckResponse_UNKNOWN,
		stopCh:     make(chan struct{}),
		stopped:    make(chan struct{}),
	}

	for _, opt := range opts {
		opt(s)
	}

	// Start with UNKNOWN status (per gRPC health protocol).
	s.health.SetServingStatus("", healthpb.HealthCheckResponse_UNKNOWN)

	return s
}

// GRPCServerOption configures the GRPCServer.
type GRPCServerOption func(*GRPCServer)

// WithCheckInterval sets the interval for checking Manager status.
// The default is 5 seconds.
func WithCheckInterval(d time.Duration) GRPCServerOption {
	return func(s *GRPCServer) {
		if d > 0 {
			s.interval = d
		}
	}
}

// Register registers the health service with the gRPC server.
// This should be called before the gRPC server starts serving.
func (s *GRPCServer) Register(registrar grpc.ServiceRegistrar) {
	healthpb.RegisterHealthServer(registrar, s.health)
}

// OnStart starts the background polling loop that syncs Manager status to gRPC health.
// Implements di.Starter.
func (s *GRPCServer) OnStart(ctx context.Context) error {
	// Run initial check immediately.
	s.checkAndUpdate(ctx)

	// Start background polling loop.
	go s.pollLoop(ctx)

	s.logger.InfoContext(ctx, "gRPC health server started",
		slog.Duration("interval", s.interval),
	)

	return nil
}

// OnStop stops the background polling loop.
// Implements di.Stopper.
func (s *GRPCServer) OnStop(ctx context.Context) error {
	s.logger.InfoContext(ctx, "gRPC health server stopping")

	// Signal stop.
	close(s.stopCh)

	// Wait for poll loop to exit or context to timeout.
	select {
	case <-s.stopped:
		s.logger.InfoContext(ctx, "gRPC health server stopped")
	case <-ctx.Done():
		s.logger.WarnContext(ctx, "gRPC health server stop timed out")
		return fmt.Errorf("grpc health server stop: %w", ctx.Err())
	}

	// Mark all services as NOT_SERVING on shutdown.
	s.health.Shutdown()

	return nil
}

// pollLoop runs the background polling loop.
func (s *GRPCServer) pollLoop(ctx context.Context) {
	defer close(s.stopped)

	ticker := time.NewTicker(s.interval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			s.checkAndUpdate(ctx)
		case <-s.stopCh:
			return
		case <-ctx.Done():
			return
		}
	}
}

// checkAndUpdate checks the Manager's readiness status and updates gRPC health.
func (s *GRPCServer) checkAndUpdate(ctx context.Context) {
	checker := s.manager.ReadinessChecker()
	result := checker.Check(ctx)

	// Map internal status to gRPC health status.
	var newStatus healthpb.HealthCheckResponse_ServingStatus
	if result.Status == internal.StatusUp {
		newStatus = healthpb.HealthCheckResponse_SERVING
	} else {
		newStatus = healthpb.HealthCheckResponse_NOT_SERVING
	}

	// Update status and log transitions.
	s.mu.Lock()
	oldStatus := s.lastStatus
	if newStatus != oldStatus {
		s.lastStatus = newStatus
		s.mu.Unlock()

		s.health.SetServingStatus("", newStatus)
		s.logStatusTransition(oldStatus, newStatus)
	} else {
		s.mu.Unlock()
	}
}

// logStatusTransition logs when the health status changes.
func (s *GRPCServer) logStatusTransition(oldStatus, newStatus healthpb.HealthCheckResponse_ServingStatus) {
	oldStr := statusToString(oldStatus)
	newStr := statusToString(newStatus)

	if newStatus == healthpb.HealthCheckResponse_SERVING {
		s.logger.Info("gRPC health status changed",
			slog.String("from", oldStr),
			slog.String("to", newStr),
		)
	} else {
		s.logger.Warn("gRPC health status changed",
			slog.String("from", oldStr),
			slog.String("to", newStr),
		)
	}
}

// statusToString converts a gRPC health status to a human-readable string.
func statusToString(status healthpb.HealthCheckResponse_ServingStatus) string {
	switch status {
	case healthpb.HealthCheckResponse_UNKNOWN:
		return "UNKNOWN"
	case healthpb.HealthCheckResponse_SERVING:
		return "SERVING"
	case healthpb.HealthCheckResponse_NOT_SERVING:
		return "NOT_SERVING"
	case healthpb.HealthCheckResponse_SERVICE_UNKNOWN:
		return "SERVICE_UNKNOWN"
	default:
		return "UNKNOWN"
	}
}

// HealthServer returns the underlying grpc health.Server for direct access.
// This is useful for advanced scenarios like setting custom service statuses.
func (s *GRPCServer) HealthServer() *health.Server {
	return s.health
}
