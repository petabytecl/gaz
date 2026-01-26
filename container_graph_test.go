package gaz

import (
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestGraph_Storage tests the internal graph storage mechanisms
func (s *ContainerSuite) TestGraph_Storage() {
	c := New()

	// Manually record dependencies to test storage
	c.recordDependency("parent", "child1")
	c.recordDependency("parent", "child2")
	c.recordDependency("other", "child3")

	graph := c.getGraph()

	assert.Contains(s.T(), graph, "parent")
	assert.Contains(s.T(), graph, "other")
	assert.Equal(s.T(), []string{"child1", "child2"}, graph["parent"])
	assert.Equal(s.T(), []string{"child3"}, graph["other"])

	// Verify deep copy
	graph["parent"][0] = "modified"
	graph2 := c.getGraph()
	assert.Equal(s.T(), "child1", graph2["parent"][0], "getGraph should return a deep copy")
}

func (s *ContainerSuite) TestGraph_CaptureDependencies() {
	c := New()

	type ServiceB struct{}
	type ServiceA struct{ B *ServiceB }

	For[*ServiceB](c).Provider(func(c *Container) (*ServiceB, error) {
		return &ServiceB{}, nil
	})

	For[*ServiceA](c).Provider(func(c *Container) (*ServiceA, error) {
		b, err := Resolve[*ServiceB](c)
		if err != nil {
			return nil, err
		}
		return &ServiceA{B: b}, nil
	})

	// Resolve A, which resolves B
	_, err := Resolve[*ServiceA](c)
	require.NoError(s.T(), err)

	graph := c.getGraph()

	// Check that A -> B is recorded
	// Note: We need the exact string names used by the container
	// Resolve[*ServiceA] uses TypeName[*ServiceA]() which is likely "*gaz.ServiceA" or similar
	// Let's resolve the names dynamically to be safe, or check contains

	// Helper to get name
	nameA := TypeName[*ServiceA]()
	nameB := TypeName[*ServiceB]()

	assert.Contains(s.T(), graph, nameA)
	assert.Contains(s.T(), graph[nameA], nameB)
}
