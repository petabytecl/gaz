package vanguard

import (
	"net/http"

	"github.com/petabytecl/gaz/health"
)

// buildHealthMux creates an http.ServeMux with health endpoints using paths
// from the provided health.Config. If cfg is nil, health.DefaultConfig() paths
// are used. Returns nil if manager is nil.
func buildHealthMux(manager *health.Manager, cfg *health.Config) *http.ServeMux {
	if manager == nil {
		return nil
	}

	hcfg := health.DefaultConfig()
	if cfg != nil {
		hcfg = *cfg
	}

	mux := http.NewServeMux()
	mux.Handle(hcfg.ReadinessPath, manager.NewReadinessHandler())
	mux.Handle(hcfg.LivenessPath, manager.NewLivenessHandler())
	if hcfg.StartupPath != "" {
		mux.Handle(hcfg.StartupPath, manager.NewStartupHandler())
	}
	return mux
}

// mountHealthEndpoints registers health handlers on the given mux using paths
// from the provided health.Config. If cfg is nil, health.DefaultConfig() paths
// are used. Does nothing if manager is nil.
func mountHealthEndpoints(mux *http.ServeMux, manager *health.Manager, cfg *health.Config) {
	if manager == nil {
		return
	}

	hcfg := health.DefaultConfig()
	if cfg != nil {
		hcfg = *cfg
	}

	mux.Handle(hcfg.ReadinessPath, manager.NewReadinessHandler())
	mux.Handle(hcfg.LivenessPath, manager.NewLivenessHandler())
	if hcfg.StartupPath != "" {
		mux.Handle(hcfg.StartupPath, manager.NewStartupHandler())
	}
}
