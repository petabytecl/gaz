package gaz

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

func (s *ContainerSuite) TestNewContainer() {
	c := NewContainer()
	s.Require().NotNil(c)
	// Container is now an alias to di.Container - check via public API
	s.Empty(c.List(), "New container should have no services")
}

func (s *ContainerSuite) TestNewContainerReturnsDistinctInstances() {
	c1 := NewContainer()
	c2 := NewContainer()
	s.NotSame(c1, c2, "NewContainer() should return distinct instances")
}

// =============================================================================
// Build() Tests
// =============================================================================

func (s *ContainerSuite) TestBuild_Idempotent() {
	c := NewContainer()
	s.Require().NoError(c.Build())
	s.Require().NoError(c.Build()) // second call also succeeds
}

func (s *ContainerSuite) TestBuild_InstantiatesEagerServices() {
	c := NewContainer()
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
	c := NewContainer()
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
	c := NewContainer()
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
// DI-01: Register with generics
// =============================================================================

func (s *ContainerSuite) TestDI01_RegisterWithGenerics() {
	c := NewContainer()
	err := For[*testDatabase](c).Provider(func(_ *Container) (*testDatabase, error) {
		return &testDatabase{}, nil
	})
	s.Require().NoError(err)

	// Verify service is registered
	db, err := Resolve[*testDatabase](c)
	s.Require().NoError(err)
	s.NotNil(db)
}

// =============================================================================
// DI-02: Lazy instantiation by default
// =============================================================================

func (s *ContainerSuite) TestDI02_LazyInstantiation() {
	c := NewContainer()
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

// =============================================================================
// DI-03: Error propagation with chain context
// =============================================================================

func (s *ContainerSuite) TestDI03_ErrorPropagation() {
	c := NewContainer()
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

// =============================================================================
// DI-04: Named implementations
// =============================================================================

func (s *ContainerSuite) TestDI04_NamedImplementations() {
	c := NewContainer()
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

// =============================================================================
// DI-05: Struct field injection
// =============================================================================

func (s *ContainerSuite) TestDI05_StructFieldInjection() {
	c := NewContainer()
	s.Require().NoError(For[*testInjectDB](c).Instance(&testInjectDB{}))
	err := For[*testHandler](c).Provider(func(_ *Container) (*testHandler, error) {
		return &testHandler{}, nil
	})
	s.Require().NoError(err)

	h, err := Resolve[*testHandler](c)
	s.Require().NoError(err)
	s.NotNil(h.DB, "DB should be injected")
}

// =============================================================================
// DI-06: Override for testing
// =============================================================================

func (s *ContainerSuite) TestDI06_Override() {
	c := NewContainer()
	s.Require().
		NoError(For[*testOverrideService](c).Instance(&testOverrideService{name: "original"}))
	s.Require().
		NoError(For[*testOverrideService](c).Replace().Instance(&testOverrideService{name: "mock"}))

	svc, err := Resolve[*testOverrideService](c)
	s.Require().NoError(err)
	s.Equal("mock", svc.name)
}

// =============================================================================
// DI-07: Transient services
// =============================================================================

func (s *ContainerSuite) TestDI07_TransientServices() {
	c := NewContainer()
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

// =============================================================================
// DI-08: Eager services
// =============================================================================

func (s *ContainerSuite) TestDI08_EagerServices() {
	c := NewContainer()
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

// =============================================================================
// DI-09: Circular dependency detection
// =============================================================================

func (s *ContainerSuite) TestDI09_CycleDetection() {
	c := NewContainer()
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
// Integration Tests
// =============================================================================

func (s *ContainerSuite) TestIntegration_AllRequirements() {
	// This test demonstrates a realistic DI setup using all 9 requirements
	c := NewContainer()

	// DI-01: Register with generics
	// DI-02: Lazy by default (Config is lazy)
	s.Require().NoError(For[*testAppConfig](c).Instance(&testAppConfig{
		dbHost: "localhost",
		dbPort: 5432,
	}))

	// DI-04: Named implementations
	err := For[*testAppDB](c).Named("primary").Provider(func(c *Container) (*testAppDB, error) {
		cfg, resolveErr := Resolve[*testAppConfig](c)
		if resolveErr != nil {
			return nil, resolveErr
		}
		return &testAppDB{host: cfg.dbHost, port: cfg.dbPort, role: "primary"}, nil
	})
	s.Require().NoError(err)

	err = For[*testAppDB](c).Named("replica").Provider(func(c *Container) (*testAppDB, error) {
		cfg, resolveErr := Resolve[*testAppConfig](c)
		if resolveErr != nil {
			return nil, resolveErr
		}
		return &testAppDB{host: cfg.dbHost, port: cfg.dbPort + 1, role: "replica"}, nil
	})
	s.Require().NoError(err)

	// DI-08: Eager service
	eagerStarted := false
	err = For[*testConnectionPool](
		c,
	).Eager().
		Provider(func(c *Container) (*testConnectionPool, error) {
			primary, resolveErr := Resolve[*testAppDB](c, Named("primary"))
			if resolveErr != nil {
				return nil, resolveErr
			}
			eagerStarted = true
			return &testConnectionPool{db: primary, poolSize: 10}, nil
		})
	s.Require().NoError(err)

	// DI-07: Transient service
	requestCounter := 0
	err = For[*testAppRequest](c).Transient().Provider(func(_ *Container) (*testAppRequest, error) {
		requestCounter++
		return &testAppRequest{id: requestCounter}, nil
	})
	s.Require().NoError(err)

	// DI-05: Struct field injection
	err = For[*testAppHandler](c).Provider(func(_ *Container) (*testAppHandler, error) {
		return &testAppHandler{}, nil
	})
	s.Require().NoError(err)

	// Before Build - eager service not started
	s.False(eagerStarted, "eager service should not start before Build()")

	// DI-08: Build instantiates eager services
	s.Require().NoError(c.Build())

	s.True(eagerStarted, "eager service should start at Build()")

	// DI-04: Named resolution
	primary, err := Resolve[*testAppDB](c, Named("primary"))
	s.Require().NoError(err)
	replica, err := Resolve[*testAppDB](c, Named("replica"))
	s.Require().NoError(err)
	s.Equal("primary", primary.role)
	s.Equal("replica", replica.role)

	// DI-02: Lazy - already resolved via eager dependency
	// DI-05: Struct field injection
	handler, err := Resolve[*testAppHandler](c)
	s.Require().NoError(err)
	s.Require().NotNil(handler.Pool, "pool should be injected")
	s.Equal(10, handler.Pool.poolSize)

	// DI-07: Transient - new instance each time
	req1, err := Resolve[*testAppRequest](c)
	s.Require().NoError(err)
	req2, err := Resolve[*testAppRequest](c)
	s.Require().NoError(err)
	s.NotEqual(req1.id, req2.id, "transient should create new instances")
}

func (s *ContainerSuite) TestIntegration_ErrorChainContext() {
	c := NewContainer()

	// Set up a chain: Handler -> Service -> Repository -> Database (fails)
	err := For[*testChainDB](c).Provider(func(_ *Container) (*testChainDB, error) {
		return nil, errors.New("cannot connect to database")
	})
	s.Require().NoError(err)

	err = For[*testChainRepo](c).Provider(func(c *Container) (*testChainRepo, error) {
		db, resolveErr := Resolve[*testChainDB](c)
		if resolveErr != nil {
			return nil, resolveErr
		}
		return &testChainRepo{db: db}, nil
	})
	s.Require().NoError(err)

	err = For[*testChainService](c).Provider(func(c *Container) (*testChainService, error) {
		repo, resolveErr := Resolve[*testChainRepo](c)
		if resolveErr != nil {
			return nil, resolveErr
		}
		return &testChainService{repo: repo}, nil
	})
	s.Require().NoError(err)

	err = For[*testChainHandler](c).Provider(func(c *Container) (*testChainHandler, error) {
		svc, resolveErr := Resolve[*testChainService](c)
		if resolveErr != nil {
			return nil, resolveErr
		}
		return &testChainHandler{svc: svc}, nil
	})
	s.Require().NoError(err)

	_, err = Resolve[*testChainHandler](c)
	s.Require().Error(err)

	// DI-03: Error should contain full chain context
	errStr := err.Error()
	expectedParts := []string{
		"testChainHandler",
		"testChainService",
		"testChainRepo",
		"testChainDB",
		"cannot connect to database",
	}
	for _, part := range expectedParts {
		s.Contains(errStr, part)
	}
}

// =============================================================================
// Test Helper Types
// =============================================================================

type (
	testEagerPool      struct{ id int }
	testFailingService struct{}
	testDatabase       struct{}
	testLazyService    struct{}
	testDB             struct{}
	testRepo           struct{ db *testDB }
	testNamedDB        struct{ name string }
	testInjectDB       struct{}
	testHandler        struct {
		DB *testInjectDB `gaz:"inject"`
	}
)

type (
	testOverrideService struct{ name string }
	testRequest         struct{ id int }
	testPool            struct{}
	testCycleA          struct{ b *testCycleB }
	testCycleB          struct{ a *testCycleA }
)

// Integration test types.
type testAppConfig struct {
	dbHost string
	dbPort int
}
type testAppDB struct {
	host string
	port int
	role string
}
type testConnectionPool struct {
	db       *testAppDB
	poolSize int
}
type (
	testAppRequest struct{ id int }
	testAppHandler struct {
		Pool *testConnectionPool `gaz:"inject"`
	}
)

// Error chain test types.
type (
	testChainDB      struct{}
	testChainRepo    struct{ db *testChainDB }
	testChainService struct{ repo *testChainRepo }
	testChainHandler struct{ svc *testChainService }
)
