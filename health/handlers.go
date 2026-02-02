package health

import (
	"net/http"

	"github.com/petabytecl/gaz/health/internal"
)

// HandlerHandler produces http.Handler for health checks.

// NewLivenessHandler creates an http.Handler for liveness probes.
// It returns 200 OK even on failure, relying on the body to indicate status,
// unless the server is completely unresponsive.
func (m *Manager) NewLivenessHandler() http.Handler {
	checker := m.LivenessChecker()
	return internal.NewHandler(checker,
		internal.WithResultWriter(internal.NewIETFResultWriter()),
		internal.WithStatusCodeUp(http.StatusOK),
		internal.WithStatusCodeDown(http.StatusOK), // 200 on failure per requirement
	)
}

// NewReadinessHandler creates an http.Handler for readiness probes.
// It returns 503 Service Unavailable on failure to stop traffic routing.
func (m *Manager) NewReadinessHandler() http.Handler {
	checker := m.ReadinessChecker()
	return internal.NewHandler(checker,
		internal.WithResultWriter(internal.NewIETFResultWriter()),
		internal.WithStatusCodeUp(http.StatusOK),
		internal.WithStatusCodeDown(http.StatusServiceUnavailable),
	)
}

// NewStartupHandler creates an http.Handler for startup probes.
// It returns 503 Service Unavailable on failure to hold off other probes.
func (m *Manager) NewStartupHandler() http.Handler {
	checker := m.StartupChecker()
	return internal.NewHandler(checker,
		internal.WithResultWriter(internal.NewIETFResultWriter()),
		internal.WithStatusCodeUp(http.StatusOK),
		internal.WithStatusCodeDown(http.StatusServiceUnavailable),
	)
}
