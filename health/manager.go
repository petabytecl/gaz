package health

import (
	"fmt"
	"sync"

	"github.com/petabytecl/gaz/health/internal"
)

// Manager implements Registrar and manages health checkers.
type Manager struct {
	mu sync.Mutex

	livenessChecks  []internal.Check
	readinessChecks []internal.Check
	startupChecks   []internal.Check

	maxHealthChecks int // Maximum number of health checks (0 = unlimited)
}

// NewManager creates a new Health Manager with default options.
func NewManager() *Manager {
	return NewManagerWithOptions(100) // Default: 100 health checks
}

// NewManagerWithOptions creates a new Health Manager with custom resource limits.
//
// maxHealthChecks is the maximum number of health checks allowed (0 = unlimited).
func NewManagerWithOptions(maxHealthChecks int) *Manager {
	return &Manager{
		maxHealthChecks: maxHealthChecks,
	}
}

// AddLivenessCheck registers a check for liveness probes.
func (m *Manager) AddLivenessCheck(name string, check CheckFunc) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Check resource limit
	if m.maxHealthChecks > 0 {
		totalChecks := len(m.livenessChecks) + len(m.readinessChecks) + len(m.startupChecks)
		if totalChecks >= m.maxHealthChecks {
			return fmt.Errorf("health: resource limit exceeded: max health checks (%d) reached", m.maxHealthChecks)
		}
	}

	m.livenessChecks = append(m.livenessChecks, internal.Check{
		Name:     name,
		Check:    check,
		Critical: true, // Default to critical per existing behavior
	})
	return nil
}

// AddReadinessCheck registers a check for readiness probes.
func (m *Manager) AddReadinessCheck(name string, check CheckFunc) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Check resource limit
	if m.maxHealthChecks > 0 {
		totalChecks := len(m.livenessChecks) + len(m.readinessChecks) + len(m.startupChecks)
		if totalChecks >= m.maxHealthChecks {
			return fmt.Errorf("health: resource limit exceeded: max health checks (%d) reached", m.maxHealthChecks)
		}
	}

	m.readinessChecks = append(m.readinessChecks, internal.Check{
		Name:     name,
		Check:    check,
		Critical: true, // Default to critical per existing behavior
	})
	return nil
}

// AddStartupCheck registers a check for startup probes.
func (m *Manager) AddStartupCheck(name string, check CheckFunc) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Check resource limit
	if m.maxHealthChecks > 0 {
		totalChecks := len(m.livenessChecks) + len(m.readinessChecks) + len(m.startupChecks)
		if totalChecks >= m.maxHealthChecks {
			return fmt.Errorf("health: resource limit exceeded: max health checks (%d) reached", m.maxHealthChecks)
		}
	}

	m.startupChecks = append(m.startupChecks, internal.Check{
		Name:     name,
		Check:    check,
		Critical: true, // Default to critical per existing behavior
	})
	return nil
}

// LivenessChecker builds the Checker for liveness checks.
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
