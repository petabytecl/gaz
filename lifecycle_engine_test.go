package gaz

import (
	"context"
	"testing"

	"github.com/stretchr/testify/suite"
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

	services := map[string]serviceWrapper{
		"A": &mockServiceWrapper{nameVal: "A", hasLifecycleVal: true},
		"B": &mockServiceWrapper{nameVal: "B", hasLifecycleVal: true},
		"C": &mockServiceWrapper{nameVal: "C", hasLifecycleVal: true},
	}

	order, err := ComputeStartupOrder(graph, services)
	s.Require().NoError(err)

	// Expected: [[C], [B], [A]]
	s.Equal(3, len(order))
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

	services := map[string]serviceWrapper{
		"A": &mockServiceWrapper{nameVal: "A", hasLifecycleVal: true},
		"B": &mockServiceWrapper{nameVal: "B", hasLifecycleVal: true},
		"C": &mockServiceWrapper{nameVal: "C", hasLifecycleVal: true},
	}

	order, err := ComputeStartupOrder(graph, services)
	s.Require().NoError(err)

	// Expected: [[C], [A, B]] or [[C], [B, A]]
	s.Equal(2, len(order))
	s.Equal([]string{"C"}, order[0])
	s.ElementsMatch([]string{"A", "B"}, order[1])
}

func (s *LifecycleEngineSuite) TestComputeStartupOrder_Cycle() {
	// Graph: A -> B -> A
	graph := map[string][]string{
		"A": {"B"},
		"B": {"A"},
	}

	services := map[string]serviceWrapper{
		"A": &mockServiceWrapper{nameVal: "A", hasLifecycleVal: true},
		"B": &mockServiceWrapper{nameVal: "B", hasLifecycleVal: true},
	}

	_, err := ComputeStartupOrder(graph, services)
	s.Error(err)
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

	services := map[string]serviceWrapper{
		"A": &mockServiceWrapper{nameVal: "A", hasLifecycleVal: true},
		"B": &mockServiceWrapper{nameVal: "B", hasLifecycleVal: false},
		"C": &mockServiceWrapper{nameVal: "C", hasLifecycleVal: true},
	}

	order, err := ComputeStartupOrder(graph, services)
	s.Require().NoError(err)

	// Expected: [[C], [A]]
	s.Equal(2, len(order))
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
	s.Equal(3, len(shutdownOrder))
	s.Equal([]string{"A"}, shutdownOrder[0])
	s.ElementsMatch([]string{"B", "D"}, shutdownOrder[1])
	s.Equal([]string{"C"}, shutdownOrder[2])
}

// mockServiceWrapper implements serviceWrapper for testing
type mockServiceWrapper struct {
	nameVal         string
	typeNameVal     string
	hasLifecycleVal bool
}

func (m *mockServiceWrapper) name() string                                          { return m.nameVal }
func (m *mockServiceWrapper) typeName() string                                      { return m.typeNameVal }
func (m *mockServiceWrapper) isEager() bool                                         { return false }
func (m *mockServiceWrapper) getInstance(c *Container, chain []string) (any, error) { return nil, nil }
func (m *mockServiceWrapper) start(context.Context) error                           { return nil }
func (m *mockServiceWrapper) stop(context.Context) error                            { return nil }
func (m *mockServiceWrapper) hasLifecycle() bool                                    { return m.hasLifecycleVal }
