package health

import (
	"sync"

	"github.com/alexliesenfeld/health"
)

// Manager implements Registrar and manages health checkers.
type Manager struct {
	mu sync.Mutex

	livenessChecks  []health.Check
	readinessChecks []health.Check
	startupChecks   []health.Check
}

// NewManager creates a new Health Manager.
func NewManager() *Manager {
	return &Manager{}
}

// AddLivenessCheck registers a check for liveness probes.
func (m *Manager) AddLivenessCheck(name string, check CheckFunc) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.livenessChecks = append(m.livenessChecks, health.Check{
		Name:  name,
		Check: check,
	})
}

// AddReadinessCheck registers a check for readiness probes.
func (m *Manager) AddReadinessCheck(name string, check CheckFunc) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.readinessChecks = append(m.readinessChecks, health.Check{
		Name:  name,
		Check: check,
	})
}

// AddStartupCheck registers a check for startup probes.
func (m *Manager) AddStartupCheck(name string, check CheckFunc) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.startupChecks = append(m.startupChecks, health.Check{
		Name:  name,
		Check: check,
	})
}

// LivenessChecker builds the health.Checker for liveness checks.
//
//nolint:ireturn // health.Checker is an external interface we must return.
func (m *Manager) LivenessChecker(opts ...health.CheckerOption) health.Checker {
	m.mu.Lock()

	defer m.mu.Unlock()

	finalOpts := make([]health.CheckerOption, 0, len(m.livenessChecks)+len(opts))
	for _, c := range m.livenessChecks {
		finalOpts = append(finalOpts, health.WithCheck(c))
	}
	finalOpts = append(finalOpts, opts...)

	return health.NewChecker(finalOpts...)
}

// ReadinessChecker builds the health.Checker for readiness checks.
//
//nolint:ireturn // health.Checker is an external interface we must return.
func (m *Manager) ReadinessChecker(opts ...health.CheckerOption) health.Checker {
	m.mu.Lock()

	defer m.mu.Unlock()

	finalOpts := make([]health.CheckerOption, 0, len(m.readinessChecks)+len(opts))
	for _, c := range m.readinessChecks {
		finalOpts = append(finalOpts, health.WithCheck(c))
	}
	finalOpts = append(finalOpts, opts...)

	return health.NewChecker(finalOpts...)
}

// StartupChecker builds the health.Checker for startup checks.
//
//nolint:ireturn // health.Checker is an external interface we must return.
func (m *Manager) StartupChecker(opts ...health.CheckerOption) health.Checker {
	m.mu.Lock()

	defer m.mu.Unlock()

	finalOpts := make([]health.CheckerOption, 0, len(m.startupChecks)+len(opts))
	for _, c := range m.startupChecks {
		finalOpts = append(finalOpts, health.WithCheck(c))
	}
	finalOpts = append(finalOpts, opts...)

	return health.NewChecker(finalOpts...)
}
