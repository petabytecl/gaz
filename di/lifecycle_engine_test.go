package di

import (
	"context"
	"testing"

	"github.com/stretchr/testify/suite"
)

// =============================================================================
// LifecycleEngineSuite - Tests for ComputeStartupOrder and ComputeShutdownOrder
// =============================================================================

type LifecycleEngineSuite struct {
	suite.Suite
}

func TestLifecycleEngineSuite(t *testing.T) {
	suite.Run(t, new(LifecycleEngineSuite))
}

// =============================================================================
// ComputeStartupOrder Tests
// =============================================================================

func (s *LifecycleEngineSuite) TestComputeStartupOrder_DI_LinearDependencyChain() {
	// A -> B -> C (A depends on B, B depends on C)
	graph := map[string][]string{
		"A": {"B"},
		"B": {"C"},
		"C": {},
	}
	services := map[string]ServiceWrapper{
		"A": newMockLifecycleService("A", true),
		"B": newMockLifecycleService("B", true),
		"C": newMockLifecycleService("C", true),
	}

	order, err := ComputeStartupOrder(graph, services)
	s.Require().NoError(err)

	// C should be first (no deps), then B, then A
	s.Require().Len(order, 3, "should have 3 layers for linear chain")
	s.Equal([]string{"C"}, order[0], "C should start first")
	s.Equal([]string{"B"}, order[1], "B should start second")
	s.Equal([]string{"A"}, order[2], "A should start last")
}

func (s *LifecycleEngineSuite) TestComputeStartupOrder_DI_NoDependencies() {
	// All services have no dependencies - should all be in first layer
	graph := map[string][]string{
		"A": {},
		"B": {},
		"C": {},
	}
	services := map[string]ServiceWrapper{
		"A": newMockLifecycleService("A", true),
		"B": newMockLifecycleService("B", true),
		"C": newMockLifecycleService("C", true),
	}

	order, err := ComputeStartupOrder(graph, services)
	s.Require().NoError(err)

	s.Require().Len(order, 1, "should have 1 layer when no dependencies")
	s.Len(order[0], 3, "all services should be in first layer")
	s.Contains(order[0], "A")
	s.Contains(order[0], "B")
	s.Contains(order[0], "C")
}

func (s *LifecycleEngineSuite) TestComputeStartupOrder_DI_MultipleWaves() {
	// Mixed dependencies creating multiple waves
	// D -> A, D -> B, C -> B (D depends on A and B, C depends on B)
	// Layer 1: A, B (no deps)
	// Layer 2: C, D (D waits for A,B; C waits for B)
	graph := map[string][]string{
		"A": {},
		"B": {},
		"C": {"B"},
		"D": {"A", "B"},
	}
	services := map[string]ServiceWrapper{
		"A": newMockLifecycleService("A", true),
		"B": newMockLifecycleService("B", true),
		"C": newMockLifecycleService("C", true),
		"D": newMockLifecycleService("D", true),
	}

	order, err := ComputeStartupOrder(graph, services)
	s.Require().NoError(err)

	s.Require().Len(order, 2, "should have 2 layers")
	// First layer: A and B
	s.Len(order[0], 2)
	s.Contains(order[0], "A")
	s.Contains(order[0], "B")
	// Second layer: C and D
	s.Len(order[1], 2)
	s.Contains(order[1], "C")
	s.Contains(order[1], "D")
}

func (s *LifecycleEngineSuite) TestComputeStartupOrder_DI_CircularDependency() {
	// A -> B -> C -> A (circular)
	graph := map[string][]string{
		"A": {"B"},
		"B": {"C"},
		"C": {"A"},
	}
	services := map[string]ServiceWrapper{
		"A": newMockLifecycleService("A", true),
		"B": newMockLifecycleService("B", true),
		"C": newMockLifecycleService("C", true),
	}

	order, err := ComputeStartupOrder(graph, services)
	s.Error(err, "should detect circular dependency")
	s.Contains(err.Error(), "circular", "error should mention circular dependency")
	s.Nil(order, "order should be nil on error")
}

func (s *LifecycleEngineSuite) TestComputeStartupOrder_DI_NodeInGraphButNotInServices() {
	// Graph has a node that doesn't exist in services map
	graph := map[string][]string{
		"A":       {"B"},
		"B":       {},
		"ORPHAN":  {}, // In graph but not in services
		"ORPHAN2": {"B"},
	}
	services := map[string]ServiceWrapper{
		"A": newMockLifecycleService("A", true),
		"B": newMockLifecycleService("B", true),
		// ORPHAN and ORPHAN2 not in services
	}

	order, err := ComputeStartupOrder(graph, services)
	s.Require().NoError(err)

	// Should only include services that exist AND have lifecycle
	s.Require().Len(order, 2, "should have 2 layers")
	s.Equal([]string{"B"}, order[0])
	s.Equal([]string{"A"}, order[1])
}

func (s *LifecycleEngineSuite) TestComputeStartupOrder_DI_ServicesWithoutLifecycle() {
	// Some services have lifecycle, some don't
	graph := map[string][]string{
		"A": {},
		"B": {},
		"C": {"A"},
	}
	services := map[string]ServiceWrapper{
		"A": newMockLifecycleService("A", true),  // Has lifecycle
		"B": newMockLifecycleService("B", false), // No lifecycle
		"C": newMockLifecycleService("C", true),  // Has lifecycle
	}

	order, err := ComputeStartupOrder(graph, services)
	s.Require().NoError(err)

	// B should be filtered out since it has no lifecycle
	totalServices := 0
	for _, layer := range order {
		for _, svc := range layer {
			s.NotEqual("B", svc, "B should be filtered out - no lifecycle")
			totalServices++
		}
	}
	s.Equal(2, totalServices, "should only have A and C")
}

func (s *LifecycleEngineSuite) TestComputeStartupOrder_DI_EmptyGraph() {
	graph := map[string][]string{}
	services := map[string]ServiceWrapper{}

	order, err := ComputeStartupOrder(graph, services)
	s.Require().NoError(err)
	s.Empty(order, "empty graph should produce empty order")
}

func (s *LifecycleEngineSuite) TestComputeStartupOrder_DI_GraphWithServicesNoGraph() {
	// Services exist but no graph entries
	graph := map[string][]string{}
	services := map[string]ServiceWrapper{
		"A": newMockLifecycleService("A", true),
		"B": newMockLifecycleService("B", true),
	}

	order, err := ComputeStartupOrder(graph, services)
	s.Require().NoError(err)

	// All services with lifecycle but no deps should be in first layer
	s.Require().Len(order, 1)
	s.Len(order[0], 2)
	s.Contains(order[0], "A")
	s.Contains(order[0], "B")
}

// =============================================================================
// ComputeShutdownOrder Tests
// =============================================================================

func (s *LifecycleEngineSuite) TestComputeShutdownOrder_DI_ReversesStartupOrder() {
	startupOrder := [][]string{
		{"C"},
		{"B"},
		{"A"},
	}

	shutdownOrder := ComputeShutdownOrder(startupOrder)

	s.Require().Len(shutdownOrder, 3)
	s.Equal([]string{"A"}, shutdownOrder[0], "A should stop first")
	s.Equal([]string{"B"}, shutdownOrder[1], "B should stop second")
	s.Equal([]string{"C"}, shutdownOrder[2], "C should stop last")
}

func (s *LifecycleEngineSuite) TestComputeShutdownOrder_DI_EmptyOrder() {
	startupOrder := [][]string{}
	shutdownOrder := ComputeShutdownOrder(startupOrder)
	s.Empty(shutdownOrder, "empty startup order should produce empty shutdown order")
}

func (s *LifecycleEngineSuite) TestComputeShutdownOrder_DI_SingleLayer() {
	startupOrder := [][]string{
		{"A", "B", "C"},
	}

	shutdownOrder := ComputeShutdownOrder(startupOrder)

	s.Require().Len(shutdownOrder, 1)
	s.Equal(startupOrder[0], shutdownOrder[0], "single layer should be same")
}

func (s *LifecycleEngineSuite) TestComputeShutdownOrder_DI_MultipleLayers() {
	startupOrder := [][]string{
		{"D", "E"},
		{"B", "C"},
		{"A"},
	}

	shutdownOrder := ComputeShutdownOrder(startupOrder)

	s.Require().Len(shutdownOrder, 3)
	s.Equal([]string{"A"}, shutdownOrder[0])
	s.Equal([]string{"B", "C"}, shutdownOrder[1])
	s.Equal([]string{"D", "E"}, shutdownOrder[2])
}

// =============================================================================
// Mock Service Implementation
// =============================================================================

type mockLifecycleService struct {
	name         string
	hasLifecycle bool
}

func newMockLifecycleService(name string, hasLifecycle bool) *mockLifecycleService {
	return &mockLifecycleService{
		name:         name,
		hasLifecycle: hasLifecycle,
	}
}

func (m *mockLifecycleService) Name() string                                      { return m.name }
func (m *mockLifecycleService) TypeName() string                                  { return m.name }
func (m *mockLifecycleService) IsEager() bool                                     { return false }
func (m *mockLifecycleService) IsTransient() bool                                 { return false }
func (m *mockLifecycleService) GetInstance(_ *Container, _ []string) (any, error) { return nil, nil }
func (m *mockLifecycleService) Start(_ context.Context) error                     { return nil }
func (m *mockLifecycleService) Stop(_ context.Context) error                      { return nil }
func (m *mockLifecycleService) HasLifecycle() bool                                { return m.hasLifecycle }
