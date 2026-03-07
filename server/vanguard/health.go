package vanguard

import (
	"net/http"

	"github.com/petabytecl/gaz/health"
)

// buildHealthMux creates an http.ServeMux with health endpoints.
// Returns nil if manager is nil.
func buildHealthMux(manager *health.Manager) *http.ServeMux {
	if manager == nil {
		return nil
	}
	mux := http.NewServeMux()
	mux.Handle("/healthz", manager.NewReadinessHandler())
	mux.Handle("/readyz", manager.NewReadinessHandler())
	mux.Handle("/livez", manager.NewLivenessHandler())
	return mux
}
