package gaz

import (
	"github.com/stretchr/testify/assert"
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
