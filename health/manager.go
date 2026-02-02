package health

import (
	"sync"

	"github.com/petabytecl/gaz/health/internal"
)

// Manager implements Registrar and manages health checkers.
type Manager struct {
	mu sync.Mutex

	livenessChecks  []internal.Check
	readinessChecks []internal.Check
	startupChecks   []internal.Check
}

// NewManager creates a new Health Manager.
func NewManager() *Manager {
	return &Manager{}
}

// AddLivenessCheck registers a check for liveness probes.
func (m *Manager) AddLivenessCheck(name string, check CheckFunc) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.livenessChecks = append(m.livenessChecks, internal.Check{
		Name:     name,
		Check:    check,
		Critical: true, // Default to critical per existing behavior
	})
}

// AddReadinessCheck registers a check for readiness probes.
func (m *Manager) AddReadinessCheck(name string, check CheckFunc) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.readinessChecks = append(m.readinessChecks, internal.Check{
		Name:     name,
		Check:    check,
		Critical: true, // Default to critical per existing behavior
	})
}

// AddStartupCheck registers a check for startup probes.
func (m *Manager) AddStartupCheck(name string, check CheckFunc) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.startupChecks = append(m.startupChecks, internal.Check{
		Name:     name,
		Check:    check,
		Critical: true, // Default to critical per existing behavior
	})
}

// LivenessChecker builds the Checker for liveness checks.
//
//nolint:ireturn // Checker interface is the intended return type for flexibility
func (m *Manager) LivenessChecker(opts ...CheckerOption) Checker {
	m.mu.Lock()

	defer m.mu.Unlock()

	finalOpts := make([]CheckerOption, 0, len(m.livenessChecks)+len(opts))
	for _, c := range m.livenessChecks {
		finalOpts = append(finalOpts, internal.WithCheck(c))
	}
	finalOpts = append(finalOpts, opts...)

	return internal.NewChecker(finalOpts...)
}

// ReadinessChecker builds the Checker for readiness checks.
//
//nolint:ireturn // Checker interface is the intended return type for flexibility
func (m *Manager) ReadinessChecker(opts ...CheckerOption) Checker {
	m.mu.Lock()

	defer m.mu.Unlock()

	finalOpts := make([]CheckerOption, 0, len(m.readinessChecks)+len(opts))
	for _, c := range m.readinessChecks {
		finalOpts = append(finalOpts, internal.WithCheck(c))
	}
	finalOpts = append(finalOpts, opts...)

	return internal.NewChecker(finalOpts...)
}

// StartupChecker builds the Checker for startup checks.
//
//nolint:ireturn // Checker interface is the intended return type for flexibility
func (m *Manager) StartupChecker(opts ...CheckerOption) Checker {
	m.mu.Lock()

	defer m.mu.Unlock()

	finalOpts := make([]CheckerOption, 0, len(m.startupChecks)+len(opts))
	for _, c := range m.startupChecks {
		finalOpts = append(finalOpts, internal.WithCheck(c))
	}
	finalOpts = append(finalOpts, opts...)

	return internal.NewChecker(finalOpts...)
}
