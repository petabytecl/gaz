package gaz

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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

func (s *ContainerSuite) TestNew() {
	c := New()
	require.NotNil(s.T(), c)
	require.NotNil(s.T(), c.services)
	assert.False(s.T(), c.built, "New container should not be built")
}

func (s *ContainerSuite) TestNewReturnsDistinctInstances() {
	c1 := New()
	c2 := New()
	assert.NotSame(s.T(), c1, c2, "New() should return distinct instances")
}

// =============================================================================
// Build() Tests
// =============================================================================

func (s *ContainerSuite) TestBuild_Idempotent() {
	c := New()
	require.NoError(s.T(), c.Build())
	require.NoError(s.T(), c.Build()) // second call also succeeds
}

func (s *ContainerSuite) TestBuild_InstantiatesEagerServices() {
	c := New()
	instantiated := false
	For[*testEagerPool](c).Eager().Provider(func(c *Container) (*testEagerPool, error) {
		instantiated = true
		return &testEagerPool{}, nil
	})

	assert.False(s.T(), instantiated, "should not instantiate before Build()")

	require.NoError(s.T(), c.Build())

	assert.True(s.T(), instantiated, "should instantiate at Build()")
}

func (s *ContainerSuite) TestBuild_EagerError_PropagatesWithContext() {
	c := New()
	For[*testFailingService](c).Eager().Provider(func(c *Container) (*testFailingService, error) {
		return nil, errors.New("startup failed")
	})

	err := c.Build()
	require.Error(s.T(), err)
	assert.Contains(s.T(), err.Error(), "testFailingService")
	assert.Contains(s.T(), err.Error(), "startup failed")
}

func (s *ContainerSuite) TestBuild_ResolveAfterBuild_ReturnsCachedEagerService() {
	c := New()
	callCount := 0
	For[*testEagerPool](c).Eager().Provider(func(c *Container) (*testEagerPool, error) {
		callCount++
		return &testEagerPool{id: callCount}, nil
	})

	require.NoError(s.T(), c.Build())

	// Resolve should return cached instance
	pool1, err := Resolve[*testEagerPool](c)
	require.NoError(s.T(), err)
	pool2, err := Resolve[*testEagerPool](c)
	require.NoError(s.T(), err)

	assert.Equal(s.T(), 1, pool1.id)
	assert.Same(s.T(), pool1, pool2, "should return same cached instance")
	assert.Equal(s.T(), 1, callCount, "provider should be called exactly once")
}

// =============================================================================
// DI-01: Register with generics
// =============================================================================

func (s *ContainerSuite) TestDI01_RegisterWithGenerics() {
	c := New()
	err := For[*testDatabase](c).Provider(func(c *Container) (*testDatabase, error) {
		return &testDatabase{}, nil
	})
	require.NoError(s.T(), err)

	// Verify service is registered
	db, err := Resolve[*testDatabase](c)
	require.NoError(s.T(), err)
	assert.NotNil(s.T(), db)
}

// =============================================================================
// DI-02: Lazy instantiation by default
// =============================================================================

func (s *ContainerSuite) TestDI02_LazyInstantiation() {
	c := New()
	instantiated := false
	For[*testLazyService](c).Provider(func(c *Container) (*testLazyService, error) {
		instantiated = true
		return &testLazyService{}, nil
	})

	assert.False(s.T(), instantiated, "should not instantiate before resolve")

	_, _ = Resolve[*testLazyService](c)
	assert.True(s.T(), instantiated, "should instantiate on first resolve")
}

// =============================================================================
// DI-03: Error propagation with chain context
// =============================================================================

func (s *ContainerSuite) TestDI03_ErrorPropagation() {
	c := New()
	For[*testDB](c).Provider(func(c *Container) (*testDB, error) {
		return nil, errors.New("connection failed")
	})
	For[*testRepo](c).Provider(func(c *Container) (*testRepo, error) {
		db, err := Resolve[*testDB](c)
		if err != nil {
			return nil, err
		}
		return &testRepo{db: db}, nil
	})

	_, err := Resolve[*testRepo](c)
	require.Error(s.T(), err)
	// Error should contain chain context
	errStr := err.Error()
	assert.Contains(s.T(), errStr, "testRepo")
	assert.Contains(s.T(), errStr, "testDB")
	assert.Contains(s.T(), errStr, "connection failed")
}

// =============================================================================
// DI-04: Named implementations
// =============================================================================

func (s *ContainerSuite) TestDI04_NamedImplementations() {
	c := New()
	For[*testNamedDB](c).Named("primary").Instance(&testNamedDB{name: "primary"})
	For[*testNamedDB](c).Named("replica").Instance(&testNamedDB{name: "replica"})

	primary, err := Resolve[*testNamedDB](c, Named("primary"))
	require.NoError(s.T(), err)
	replica, err := Resolve[*testNamedDB](c, Named("replica"))
	require.NoError(s.T(), err)

	assert.Equal(s.T(), "primary", primary.name)
	assert.Equal(s.T(), "replica", replica.name)
	assert.NotSame(s.T(), primary, replica, "should be different instances")
}

// =============================================================================
// DI-05: Struct field injection
// =============================================================================

func (s *ContainerSuite) TestDI05_StructFieldInjection() {
	c := New()
	For[*testInjectDB](c).Instance(&testInjectDB{})
	For[*testHandler](c).Provider(func(c *Container) (*testHandler, error) {
		return &testHandler{}, nil
	})

	h, err := Resolve[*testHandler](c)
	require.NoError(s.T(), err)
	assert.NotNil(s.T(), h.DB, "DB should be injected")
}

// =============================================================================
// DI-06: Override for testing
// =============================================================================

func (s *ContainerSuite) TestDI06_Override() {
	c := New()
	For[*testOverrideService](c).Instance(&testOverrideService{name: "original"})
	For[*testOverrideService](c).Replace().Instance(&testOverrideService{name: "mock"})

	svc, err := Resolve[*testOverrideService](c)
	require.NoError(s.T(), err)
	assert.Equal(s.T(), "mock", svc.name)
}

// =============================================================================
// DI-07: Transient services
// =============================================================================

func (s *ContainerSuite) TestDI07_TransientServices() {
	c := New()
	counter := 0
	For[*testRequest](c).Transient().Provider(func(c *Container) (*testRequest, error) {
		counter++
		return &testRequest{id: counter}, nil
	})

	r1, err := Resolve[*testRequest](c)
	require.NoError(s.T(), err)
	r2, err := Resolve[*testRequest](c)
	require.NoError(s.T(), err)

	assert.NotEqual(s.T(), r1.id, r2.id, "should be different instances")
	assert.Equal(s.T(), 1, r1.id)
	assert.Equal(s.T(), 2, r2.id)
}

// =============================================================================
// DI-08: Eager services
// =============================================================================

func (s *ContainerSuite) TestDI08_EagerServices() {
	c := New()
	instantiated := false
	For[*testPool](c).Eager().Provider(func(c *Container) (*testPool, error) {
		instantiated = true
		return &testPool{}, nil
	})

	assert.False(s.T(), instantiated, "should not instantiate before Build")

	require.NoError(s.T(), c.Build())

	assert.True(s.T(), instantiated, "should instantiate at Build")
}

// =============================================================================
// DI-09: Circular dependency detection
// =============================================================================

func (s *ContainerSuite) TestDI09_CycleDetection() {
	c := New()
	For[*testCycleA](c).Provider(func(c *Container) (*testCycleA, error) {
		b, err := Resolve[*testCycleB](c)
		if err != nil {
			return nil, err
		}
		return &testCycleA{b: b}, nil
	})
	For[*testCycleB](c).Provider(func(c *Container) (*testCycleB, error) {
		a, err := Resolve[*testCycleA](c)
		if err != nil {
			return nil, err
		}
		return &testCycleB{a: a}, nil
	})

	_, err := Resolve[*testCycleA](c)
	assert.ErrorIs(s.T(), err, ErrCycle)
}

// =============================================================================
// Integration Tests
// =============================================================================

func (s *ContainerSuite) TestIntegration_AllRequirements() {
	// This test demonstrates a realistic DI setup using all 9 requirements
	c := New()

	// DI-01: Register with generics
	// DI-02: Lazy by default (Config is lazy)
	For[*testAppConfig](c).Instance(&testAppConfig{
		dbHost: "localhost",
		dbPort: 5432,
	})

	// DI-04: Named implementations
	For[*testAppDB](c).Named("primary").Provider(func(c *Container) (*testAppDB, error) {
		cfg, err := Resolve[*testAppConfig](c)
		if err != nil {
			return nil, err
		}
		return &testAppDB{host: cfg.dbHost, port: cfg.dbPort, role: "primary"}, nil
	})

	For[*testAppDB](c).Named("replica").Provider(func(c *Container) (*testAppDB, error) {
		cfg, err := Resolve[*testAppConfig](c)
		if err != nil {
			return nil, err
		}
		return &testAppDB{host: cfg.dbHost, port: cfg.dbPort + 1, role: "replica"}, nil
	})

	// DI-08: Eager service
	eagerStarted := false
	For[*testConnectionPool](c).Eager().Provider(func(c *Container) (*testConnectionPool, error) {
		primary, err := Resolve[*testAppDB](c, Named("primary"))
		if err != nil {
			return nil, err
		}
		eagerStarted = true
		return &testConnectionPool{db: primary, poolSize: 10}, nil
	})

	// DI-07: Transient service
	requestCounter := 0
	For[*testAppRequest](c).Transient().Provider(func(c *Container) (*testAppRequest, error) {
		requestCounter++
		return &testAppRequest{id: requestCounter}, nil
	})

	// DI-05: Struct field injection
	For[*testAppHandler](c).Provider(func(c *Container) (*testAppHandler, error) {
		return &testAppHandler{}, nil
	})

	// Before Build - eager service not started
	assert.False(s.T(), eagerStarted, "eager service should not start before Build()")

	// DI-08: Build instantiates eager services
	require.NoError(s.T(), c.Build())

	assert.True(s.T(), eagerStarted, "eager service should start at Build()")

	// DI-04: Named resolution
	primary, err := Resolve[*testAppDB](c, Named("primary"))
	require.NoError(s.T(), err)
	replica, err := Resolve[*testAppDB](c, Named("replica"))
	require.NoError(s.T(), err)
	assert.Equal(s.T(), "primary", primary.role)
	assert.Equal(s.T(), "replica", replica.role)

	// DI-02: Lazy - already resolved via eager dependency
	// DI-05: Struct field injection
	handler, err := Resolve[*testAppHandler](c)
	require.NoError(s.T(), err)
	require.NotNil(s.T(), handler.Pool, "pool should be injected")
	assert.Equal(s.T(), 10, handler.Pool.poolSize)

	// DI-07: Transient - new instance each time
	req1, err := Resolve[*testAppRequest](c)
	require.NoError(s.T(), err)
	req2, err := Resolve[*testAppRequest](c)
	require.NoError(s.T(), err)
	assert.NotEqual(s.T(), req1.id, req2.id, "transient should create new instances")
}

func (s *ContainerSuite) TestIntegration_ErrorChainContext() {
	c := New()

	// Set up a chain: Handler -> Service -> Repository -> Database (fails)
	For[*testChainDB](c).Provider(func(c *Container) (*testChainDB, error) {
		return nil, errors.New("cannot connect to database")
	})

	For[*testChainRepo](c).Provider(func(c *Container) (*testChainRepo, error) {
		db, err := Resolve[*testChainDB](c)
		if err != nil {
			return nil, err
		}
		return &testChainRepo{db: db}, nil
	})

	For[*testChainService](c).Provider(func(c *Container) (*testChainService, error) {
		repo, err := Resolve[*testChainRepo](c)
		if err != nil {
			return nil, err
		}
		return &testChainService{repo: repo}, nil
	})

	For[*testChainHandler](c).Provider(func(c *Container) (*testChainHandler, error) {
		svc, err := Resolve[*testChainService](c)
		if err != nil {
			return nil, err
		}
		return &testChainHandler{svc: svc}, nil
	})

	_, err := Resolve[*testChainHandler](c)
	require.Error(s.T(), err)

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
		assert.Contains(s.T(), errStr, part)
	}
}

// =============================================================================
// Test Helper Types
// =============================================================================

type testEagerPool struct{ id int }
type testFailingService struct{}
type testDatabase struct{}
type testLazyService struct{}
type testDB struct{}
type testRepo struct{ db *testDB }
type testNamedDB struct{ name string }
type testInjectDB struct{}
type testHandler struct {
	DB *testInjectDB `gaz:"inject"`
}
type testOverrideService struct{ name string }
type testRequest struct{ id int }
type testPool struct{}
type testCycleA struct{ b *testCycleB }
type testCycleB struct{ a *testCycleA }

// Integration test types
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
type testAppRequest struct{ id int }
type testAppHandler struct {
	Pool *testConnectionPool `gaz:"inject"`
}

// Error chain test types
type testChainDB struct{}
type testChainRepo struct{ db *testChainDB }
type testChainService struct{ repo *testChainRepo }
type testChainHandler struct{ svc *testChainService }
