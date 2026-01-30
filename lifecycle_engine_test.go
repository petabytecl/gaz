package gaz

import (
	"context"
	"testing"

	"github.com/stretchr/testify/suite"

	"github.com/petabytecl/gaz/di"
)

type LifecycleEngineSuite struct {
	suite.Suite
}

func TestLifecycleEngineSuite(t *testing.T) {
	suite.Run(t, new(LifecycleEngineSuite))
}

func (s *LifecycleEngineSuite) TestComputeStartupOrder_SimpleLinear() {
	// Graph: A -> B -> C
	graph := map[string][]string{
		"A": {"B"},
		"B": {"C"},
		"C": {},
	}

	services := map[string]di.ServiceWrapper{
		"A": &mockServiceWrapper{nameVal: "A", hasLifecycleVal: true},
		"B": &mockServiceWrapper{nameVal: "B", hasLifecycleVal: true},
		"C": &mockServiceWrapper{nameVal: "C", hasLifecycleVal: true},
	}

	order, err := ComputeStartupOrder(graph, services)
	s.Require().NoError(err)

	// Expected: [[C], [B], [A]]
	s.Len(order, 3)
	s.Equal([]string{"C"}, order[0])
	s.Equal([]string{"B"}, order[1])
	s.Equal([]string{"A"}, order[2])
}

func (s *LifecycleEngineSuite) TestComputeStartupOrder_Parallel() {
	// Graph: A -> C, B -> C
	// A and B are independent, can be in same layer.
	graph := map[string][]string{
		"A": {"C"},
		"B": {"C"},
		"C": {},
	}

	services := map[string]di.ServiceWrapper{
		"A": &mockServiceWrapper{nameVal: "A", hasLifecycleVal: true},
		"B": &mockServiceWrapper{nameVal: "B", hasLifecycleVal: true},
		"C": &mockServiceWrapper{nameVal: "C", hasLifecycleVal: true},
	}

	order, err := ComputeStartupOrder(graph, services)
	s.Require().NoError(err)

	// Expected: [[C], [A, B]] or [[C], [B, A]]
	s.Len(order, 2)
	s.Equal([]string{"C"}, order[0])
	s.ElementsMatch([]string{"A", "B"}, order[1])
}

func (s *LifecycleEngineSuite) TestComputeStartupOrder_Cycle() {
	// Graph: A -> B -> A
	graph := map[string][]string{
		"A": {"B"},
		"B": {"A"},
	}

	services := map[string]di.ServiceWrapper{
		"A": &mockServiceWrapper{nameVal: "A", hasLifecycleVal: true},
		"B": &mockServiceWrapper{nameVal: "B", hasLifecycleVal: true},
	}

	_, err := ComputeStartupOrder(graph, services)
	s.Require().Error(err)
	s.Contains(err.Error(), "circular dependency detected")
}

func (s *LifecycleEngineSuite) TestComputeStartupOrder_FilterNoLifecycle() {
	// Graph: A -> B -> C
	// B has no lifecycle hooks.
	// Expected: [[C], [A]] (B is filtered out)
	graph := map[string][]string{
		"A": {"B"},
		"B": {"C"},
		"C": {},
	}

	services := map[string]di.ServiceWrapper{
		"A": &mockServiceWrapper{nameVal: "A", hasLifecycleVal: true},
		"B": &mockServiceWrapper{nameVal: "B", hasLifecycleVal: false},
		"C": &mockServiceWrapper{nameVal: "C", hasLifecycleVal: true},
	}

	order, err := ComputeStartupOrder(graph, services)
	s.Require().NoError(err)

	// Expected: [[C], [A]]
	s.Len(order, 2)
	s.Equal([]string{"C"}, order[0])
	s.Equal([]string{"A"}, order[1])
}

func (s *LifecycleEngineSuite) TestComputeShutdownOrder() {
	startupOrder := [][]string{
		{"C"},
		{"B", "D"},
		{"A"},
	}

	shutdownOrder := ComputeShutdownOrder(startupOrder)

	// Expected reverse: [[A], [B, D] (or D, B), [C]]
	// Actually, strictly reversing the layers is enough.
	// [[A], [B, D], [C]]
	s.Len(shutdownOrder, 3)
	s.Equal([]string{"A"}, shutdownOrder[0])
	s.ElementsMatch([]string{"B", "D"}, shutdownOrder[1])
	s.Equal([]string{"C"}, shutdownOrder[2])
}

// mockServiceWrapper implements di.ServiceWrapper for testing.
type mockServiceWrapper struct {
	nameVal         string
	typeNameVal     string
	hasLifecycleVal bool
}

func (m *mockServiceWrapper) Name() string      { return m.nameVal }
func (m *mockServiceWrapper) TypeName() string  { return m.typeNameVal }
func (m *mockServiceWrapper) IsEager() bool     { return false }
func (m *mockServiceWrapper) IsTransient() bool { return false }

func (m *mockServiceWrapper) GetInstance(
	_ *di.Container,
	_ []string,
) (any, error) {
	return nil, nil
}
func (m *mockServiceWrapper) Start(context.Context) error { return nil }
func (m *mockServiceWrapper) Stop(context.Context) error  { return nil }
func (m *mockServiceWrapper) HasLifecycle() bool          { return m.hasLifecycleVal }
