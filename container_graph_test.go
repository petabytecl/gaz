package gaz

// TestGraph_Storage tests the internal graph storage mechanisms via public API.
func (s *ContainerSuite) TestGraph_Storage() {
	c := NewContainer()

	// Register services with dependencies to test graph recording
	type child1 struct{}
	type child2 struct{}
	type child3 struct{}
	type parent struct{}
	type other struct{}

	// Register children
	_ = For[*child1](c).Instance(&child1{})
	_ = For[*child2](c).Instance(&child2{})
	_ = For[*child3](c).Instance(&child3{})

	// Parent depends on child1 and child2
	_ = For[*parent](c).Provider(func(c *Container) (*parent, error) {
		_, _ = Resolve[*child1](c)
		_, _ = Resolve[*child2](c)
		return &parent{}, nil
	})

	// Other depends on child3
	_ = For[*other](c).Provider(func(c *Container) (*other, error) {
		_, _ = Resolve[*child3](c)
		return &other{}, nil
	})

	// Resolve to populate graph
	_, _ = Resolve[*parent](c)
	_, _ = Resolve[*other](c)

	graph := c.GetGraph()

	// Verify graph contains expected dependencies
	parentName := TypeName[*parent]()
	otherName := TypeName[*other]()
	child1Name := TypeName[*child1]()
	child2Name := TypeName[*child2]()
	child3Name := TypeName[*child3]()

	s.Contains(graph, parentName)
	s.Contains(graph, otherName)
	s.Contains(graph[parentName], child1Name)
	s.Contains(graph[parentName], child2Name)
	s.Contains(graph[otherName], child3Name)

	// Verify deep copy
	if len(graph[parentName]) > 0 {
		graph[parentName][0] = "modified"
		graph2 := c.GetGraph()
		s.Equal(child1Name, graph2[parentName][0], "GetGraph should return a deep copy")
	}
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

	graph := c.GetGraph()

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
