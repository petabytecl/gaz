package grpc

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/health"
	healthpb "google.golang.org/grpc/health/grpc_health_v1"

	gazhealth "github.com/petabytecl/gaz/health"
)

// healthAdapter wraps grpc-go's health.Server and syncs status from the Manager.
// It polls the Manager's readiness checks at a configurable interval and updates
// the gRPC health status accordingly.
type healthAdapter struct {
	health   *health.Server
	manager  *gazhealth.Manager
	logger   *slog.Logger
	interval time.Duration

	mu         sync.Mutex
	lastStatus healthpb.HealthCheckResponse_ServingStatus
	stopCh     chan struct{}
	stopped    chan struct{}
}

// newHealthAdapter creates a new gRPC health adapter.
func newHealthAdapter(manager *gazhealth.Manager, interval time.Duration, logger *slog.Logger) *healthAdapter {
	if logger == nil {
		logger = slog.Default()
	}

	s := &healthAdapter{
		health:     health.NewServer(),
		manager:    manager,
		logger:     logger,
		interval:   interval,
		lastStatus: healthpb.HealthCheckResponse_UNKNOWN,
		stopCh:     make(chan struct{}),
		stopped:    make(chan struct{}),
	}

	// Start with UNKNOWN status (per gRPC health protocol).
	s.health.SetServingStatus("", healthpb.HealthCheckResponse_UNKNOWN)

	return s
}

// Register registers the health service with the gRPC server.
func (s *healthAdapter) Register(registrar grpc.ServiceRegistrar) {
	healthpb.RegisterHealthServer(registrar, s.health)
}

// Start starts the background polling loop.
func (s *healthAdapter) Start(ctx context.Context) {
	// Run initial check immediately.
	s.checkAndUpdate(ctx)

	// Start background polling loop.
	go s.pollLoop(ctx)

	s.logger.InfoContext(ctx, "gRPC health adapter started",
		slog.Duration("interval", s.interval),
	)
}

// Stop stops the background polling loop.
func (s *healthAdapter) Stop(ctx context.Context) error {
	s.logger.InfoContext(ctx, "gRPC health adapter stopping")

	// Signal stop.
	close(s.stopCh)

	// Wait for poll loop to exit or context to timeout.
	select {
	case <-s.stopped:
		s.logger.InfoContext(ctx, "gRPC health adapter stopped")
	case <-ctx.Done():
		s.logger.WarnContext(ctx, "gRPC health adapter stop timed out")
		return fmt.Errorf("grpc health adapter stop: %w", ctx.Err())
	}

	// Mark all services as NOT_SERVING on shutdown.
	s.health.Shutdown()

	return nil
}

// pollLoop runs the background polling loop.
func (s *healthAdapter) pollLoop(ctx context.Context) {
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
func (s *healthAdapter) checkAndUpdate(ctx context.Context) {
	checker := s.manager.ReadinessChecker()
	result := checker.Check(ctx)

	// Map internal status to gRPC health status.
	var newStatus healthpb.HealthCheckResponse_ServingStatus
	if result.Status == gazhealth.StatusUp {
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
func (s *healthAdapter) logStatusTransition(oldStatus, newStatus healthpb.HealthCheckResponse_ServingStatus) {
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
