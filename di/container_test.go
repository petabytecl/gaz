package di

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/suite"
)

// =============================================================================
// ContainerSuite
// =============================================================================

type ContainerSuite struct {
	suite.Suite
}

func TestContainerSuite(t *testing.T) {
	suite.Run(t, new(ContainerSuite))
}

// =============================================================================
// New() Tests
// =============================================================================

func (s *ContainerSuite) TestNew() {
	c := New()
	s.Require().NotNil(c)
	s.Equal(0, len(c.List()), "New container should have no services")
}

func (s *ContainerSuite) TestNew_ReturnsDistinctInstances() {
	c1 := New()
	c2 := New()
	s.NotSame(c1, c2, "New() should return distinct instances")
}

// =============================================================================
// Build() Tests
// =============================================================================

func (s *ContainerSuite) TestBuild_Idempotent() {
	c := New()
	s.Require().NoError(c.Build())
	s.Require().NoError(c.Build()) // second call also succeeds
}

func (s *ContainerSuite) TestBuild_InstantiatesEagerServices() {
	c := New()
	instantiated := false
	err := For[*testEagerPool](c).Eager().Provider(func(_ *Container) (*testEagerPool, error) {
		instantiated = true
		return &testEagerPool{}, nil
	})
	s.Require().NoError(err)

	s.False(instantiated, "should not instantiate before Build()")

	s.Require().NoError(c.Build())

	s.True(instantiated, "should instantiate at Build()")
}

func (s *ContainerSuite) TestBuild_EagerError_PropagatesWithContext() {
	c := New()
	regErr := For[*testFailingService](
		c,
	).Eager().
		Provider(func(_ *Container) (*testFailingService, error) {
			return nil, errors.New("startup failed")
		})
	s.Require().NoError(regErr)

	err := c.Build()
	s.Require().Error(err)
	s.Contains(err.Error(), "testFailingService")
	s.Contains(err.Error(), "startup failed")
}

func (s *ContainerSuite) TestBuild_ResolveAfterBuild_ReturnsCachedEagerService() {
	c := New()
	callCount := 0
	err := For[*testEagerPool](c).Eager().Provider(func(_ *Container) (*testEagerPool, error) {
		callCount++
		return &testEagerPool{id: callCount}, nil
	})
	s.Require().NoError(err)

	s.Require().NoError(c.Build())

	// Resolve should return cached instance
	pool1, err := Resolve[*testEagerPool](c)
	s.Require().NoError(err)
	pool2, err := Resolve[*testEagerPool](c)
	s.Require().NoError(err)

	s.Equal(1, pool1.id)
	s.Same(pool1, pool2, "should return same cached instance")
	s.Equal(1, callCount, "provider should be called exactly once")
}

// =============================================================================
// List() Tests
// =============================================================================

func (s *ContainerSuite) TestList_Empty() {
	c := New()
	list := c.List()
	s.Empty(list, "empty container should have no services")
}

func (s *ContainerSuite) TestList_WithServices() {
	c := New()
	s.Require().NoError(For[*testDatabase](c).Instance(&testDatabase{}))
	s.Require().NoError(For[*testLazyService](c).Instance(&testLazyService{}))

	list := c.List()
	s.Len(list, 2, "should have 2 services")
	s.Contains(list, TypeName[*testDatabase]())
	s.Contains(list, TypeName[*testLazyService]())
}

func (s *ContainerSuite) TestList_Sorted() {
	c := New()
	// Register in reverse alphabetical order
	s.Require().NoError(For[*testDatabase](c).Instance(&testDatabase{}))
	s.Require().NoError(For[*testApp](c).Instance(&testApp{}))
	s.Require().NoError(For[*testLazyService](c).Instance(&testLazyService{}))

	list := c.List()
	// Should be sorted
	for i := 1; i < len(list); i++ {
		s.True(list[i-1] < list[i], "list should be sorted: %s should come before %s", list[i-1], list[i])
	}
}

// =============================================================================
// Has[T]() Tests
// =============================================================================

func (s *ContainerSuite) TestHas_NotRegistered() {
	c := New()
	s.False(Has[*testDatabase](c), "should return false for unregistered type")
}

func (s *ContainerSuite) TestHas_Registered() {
	c := New()
	s.Require().NoError(For[*testDatabase](c).Instance(&testDatabase{}))
	s.True(Has[*testDatabase](c), "should return true for registered type")
}

// =============================================================================
// HasService() Tests
// =============================================================================

func (s *ContainerSuite) TestHasService_NotRegistered() {
	c := New()
	s.False(c.HasService("nonexistent"), "should return false for unregistered name")
}

func (s *ContainerSuite) TestHasService_Registered() {
	c := New()
	s.Require().NoError(For[*testDatabase](c).Instance(&testDatabase{}))
	s.True(c.HasService(TypeName[*testDatabase]()), "should return true for registered name")
}

func (s *ContainerSuite) TestHasService_Named() {
	c := New()
	s.Require().NoError(For[*testNamedDB](c).Named("primary").Instance(&testNamedDB{name: "primary"}))
	s.True(c.HasService("primary"), "should return true for named service")
	s.False(c.HasService(TypeName[*testNamedDB]()), "should return false for type name when using named")
}

// =============================================================================
// ForEachService() Tests
// =============================================================================

func (s *ContainerSuite) TestForEachService_Empty() {
	c := New()
	count := 0
	c.ForEachService(func(_ string, _ ServiceWrapper) {
		count++
	})
	s.Equal(0, count, "should not iterate over empty container")
}

func (s *ContainerSuite) TestForEachService_WithServices() {
	c := New()
	s.Require().NoError(For[*testDatabase](c).Instance(&testDatabase{}))
	s.Require().NoError(For[*testLazyService](c).Instance(&testLazyService{}))

	names := make([]string, 0)
	c.ForEachService(func(name string, svc ServiceWrapper) {
		names = append(names, name)
		s.NotNil(svc, "service wrapper should not be nil")
	})
	s.Len(names, 2, "should iterate over all services")
}

// =============================================================================
// GetService() Tests
// =============================================================================

func (s *ContainerSuite) TestGetService_NotFound() {
	c := New()
	svc, found := c.GetService("nonexistent")
	s.False(found, "should return false for nonexistent service")
	s.Nil(svc, "should return nil for nonexistent service")
}

func (s *ContainerSuite) TestGetService_Found() {
	c := New()
	original := &testDatabase{host: "localhost"}
	s.Require().NoError(For[*testDatabase](c).Instance(original))

	svc, found := c.GetService(TypeName[*testDatabase]())
	s.True(found, "should return true for registered service")
	s.NotNil(svc, "should return service wrapper")
	s.Equal(TypeName[*testDatabase](), svc.Name())
}

// =============================================================================
// GetGraph() Tests
// =============================================================================

func (s *ContainerSuite) TestGetGraph_Empty() {
	c := New()
	graph := c.GetGraph()
	s.Empty(graph, "empty container should have empty graph")
}

func (s *ContainerSuite) TestGetGraph_WithDependencies() {
	c := New()

	type child struct{}
	type parent struct{ c *child }

	s.Require().NoError(For[*child](c).Instance(&child{}))
	err := For[*parent](c).Provider(func(c *Container) (*parent, error) {
		dep, resolveErr := Resolve[*child](c)
		if resolveErr != nil {
			return nil, resolveErr
		}
		return &parent{c: dep}, nil
	})
	s.Require().NoError(err)

	// Resolve parent to populate graph
	_, err = Resolve[*parent](c)
	s.Require().NoError(err)

	graph := c.GetGraph()
	parentName := TypeName[*parent]()
	childName := TypeName[*child]()

	s.Contains(graph, parentName, "graph should contain parent")
	s.Contains(graph[parentName], childName, "parent should depend on child")
}

func (s *ContainerSuite) TestGetGraph_ReturnsDeepCopy() {
	c := New()

	type child struct{}
	type parent struct{ c *child }

	s.Require().NoError(For[*child](c).Instance(&child{}))
	err := For[*parent](c).Provider(func(c *Container) (*parent, error) {
		dep, resolveErr := Resolve[*child](c)
		if resolveErr != nil {
			return nil, resolveErr
		}
		return &parent{c: dep}, nil
	})
	s.Require().NoError(err)

	// Resolve parent to populate graph
	_, err = Resolve[*parent](c)
	s.Require().NoError(err)

	graph1 := c.GetGraph()
	parentName := TypeName[*parent]()

	// Modify the returned graph
	if len(graph1[parentName]) > 0 {
		graph1[parentName][0] = "modified"

		graph2 := c.GetGraph()
		s.NotEqual("modified", graph2[parentName][0], "GetGraph should return deep copy")
	}
}

// =============================================================================
// DI Requirements Tests
// =============================================================================

func (s *ContainerSuite) TestDI01_RegisterWithGenerics() {
	c := New()
	err := For[*testDatabase](c).Provider(func(_ *Container) (*testDatabase, error) {
		return &testDatabase{}, nil
	})
	s.Require().NoError(err)

	// Verify service is registered
	db, err := Resolve[*testDatabase](c)
	s.Require().NoError(err)
	s.NotNil(db)
}

func (s *ContainerSuite) TestDI02_LazyInstantiation() {
	c := New()
	instantiated := false
	err := For[*testLazyService](c).Provider(func(_ *Container) (*testLazyService, error) {
		instantiated = true
		return &testLazyService{}, nil
	})
	s.Require().NoError(err)

	s.False(instantiated, "should not instantiate before resolve")

	_, _ = Resolve[*testLazyService](c)
	s.True(instantiated, "should instantiate on first resolve")
}

func (s *ContainerSuite) TestDI03_ErrorPropagation() {
	c := New()
	err := For[*testDB](c).Provider(func(_ *Container) (*testDB, error) {
		return nil, errors.New("connection failed")
	})
	s.Require().NoError(err)

	err = For[*testRepo](c).Provider(func(c *Container) (*testRepo, error) {
		db, resolveErr := Resolve[*testDB](c)
		if resolveErr != nil {
			return nil, resolveErr
		}
		return &testRepo{db: db}, nil
	})
	s.Require().NoError(err)

	_, err = Resolve[*testRepo](c)
	s.Require().Error(err)
	// Error should contain chain context
	errStr := err.Error()
	s.Contains(errStr, "testRepo")
	s.Contains(errStr, "testDB")
	s.Contains(errStr, "connection failed")
}

func (s *ContainerSuite) TestDI04_NamedImplementations() {
	c := New()
	s.Require().
		NoError(For[*testNamedDB](c).Named("primary").Instance(&testNamedDB{name: "primary"}))
	s.Require().
		NoError(For[*testNamedDB](c).Named("replica").Instance(&testNamedDB{name: "replica"}))

	primary, err := Resolve[*testNamedDB](c, Named("primary"))
	s.Require().NoError(err)
	replica, err := Resolve[*testNamedDB](c, Named("replica"))
	s.Require().NoError(err)

	s.Equal("primary", primary.name)
	s.Equal("replica", replica.name)
	s.NotSame(primary, replica, "should be different instances")
}

func (s *ContainerSuite) TestDI07_TransientServices() {
	c := New()
	counter := 0
	err := For[*testRequest](c).Transient().Provider(func(_ *Container) (*testRequest, error) {
		counter++
		return &testRequest{id: counter}, nil
	})
	s.Require().NoError(err)

	r1, err := Resolve[*testRequest](c)
	s.Require().NoError(err)
	r2, err := Resolve[*testRequest](c)
	s.Require().NoError(err)

	s.NotEqual(r1.id, r2.id, "should be different instances")
	s.Equal(1, r1.id)
	s.Equal(2, r2.id)
}

func (s *ContainerSuite) TestDI08_EagerServices() {
	c := New()
	instantiated := false
	err := For[*testPool](c).Eager().Provider(func(_ *Container) (*testPool, error) {
		instantiated = true
		return &testPool{}, nil
	})
	s.Require().NoError(err)

	s.False(instantiated, "should not instantiate before Build")

	s.Require().NoError(c.Build())

	s.True(instantiated, "should instantiate at Build")
}

func (s *ContainerSuite) TestDI09_CycleDetection() {
	c := New()
	err := For[*testCycleA](c).Provider(func(c *Container) (*testCycleA, error) {
		b, resolveErr := Resolve[*testCycleB](c)
		if resolveErr != nil {
			return nil, resolveErr
		}
		return &testCycleA{b: b}, nil
	})
	s.Require().NoError(err)

	err = For[*testCycleB](c).Provider(func(c *Container) (*testCycleB, error) {
		a, resolveErr := Resolve[*testCycleA](c)
		if resolveErr != nil {
			return nil, resolveErr
		}
		return &testCycleB{a: a}, nil
	})
	s.Require().NoError(err)

	_, err = Resolve[*testCycleA](c)
	s.Require().ErrorIs(err, ErrCycle)
}

// =============================================================================
// Test Helper Types
// =============================================================================

type testApp struct{}
type testEagerPool struct{ id int }
type testFailingService struct{}
type testDatabase struct{ host string }
type testLazyService struct{}
type testDB struct{}
type testRepo struct{ db *testDB }
type testNamedDB struct{ name string }
type testRequest struct{ id int }
type testPool struct{}
type testCycleA struct{ b *testCycleB }
type testCycleB struct{ a *testCycleA }
