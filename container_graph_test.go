package gaz

// TestGraph_Storage tests the internal graph storage mechanisms.
func (s *ContainerSuite) TestGraph_Storage() {
	c := NewContainer()

	// Manually record dependencies to test storage
	c.recordDependency("parent", "child1")
	c.recordDependency("parent", "child2")
	c.recordDependency("other", "child3")

	graph := c.getGraph()

	s.Contains(graph, "parent")
	s.Contains(graph, "other")
	s.Equal([]string{"child1", "child2"}, graph["parent"])
	s.Equal([]string{"child3"}, graph["other"])

	// Verify deep copy
	graph["parent"][0] = "modified"
	graph2 := c.getGraph()
	s.Equal("child1", graph2["parent"][0], "getGraph should return a deep copy")
}

func (s *ContainerSuite) TestGraph_CaptureDependencies() {
	c := NewContainer()

	type ServiceB struct{}
	type ServiceA struct{ B *ServiceB }

	err := For[*ServiceB](c).Provider(func(_ *Container) (*ServiceB, error) {
		return &ServiceB{}, nil
	})
	s.Require().NoError(err)

	err = For[*ServiceA](c).Provider(func(c *Container) (*ServiceA, error) {
		b, resolveErr := Resolve[*ServiceB](c)
		if resolveErr != nil {
			return nil, resolveErr
		}
		return &ServiceA{B: b}, nil
	})
	s.Require().NoError(err)

	// Resolve A, which resolves B
	_, err = Resolve[*ServiceA](c)
	s.Require().NoError(err)

	graph := c.getGraph()

	// Check that A -> B is recorded
	// Note: We need the exact string names used by the container
	// Resolve[*ServiceA] uses TypeName[*ServiceA]() which is likely "*gaz.ServiceA" or similar
	// Let's resolve the names dynamically to be safe, or check contains

	// Helper to get name
	nameA := TypeName[*ServiceA]()
	nameB := TypeName[*ServiceB]()

	s.Contains(graph, nameA)
	s.Contains(graph[nameA], nameB)
}
