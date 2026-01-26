package gaz

import (
	"errors"
	"strings"
	"testing"
)

func TestNew(t *testing.T) {
	c := New()
	if c == nil {
		t.Fatal("New() returned nil")
	}
	if c.services == nil {
		t.Fatal("New() did not initialize services map")
	}
	if c.built {
		t.Fatal("New container should not be built")
	}
}

func TestNewReturnsDistinctInstances(t *testing.T) {
	c1 := New()
	c2 := New()
	if c1 == c2 {
		t.Fatal("New() should return distinct instances")
	}
}

// =============================================================================
// Build() Tests
// =============================================================================

func TestBuild_Idempotent(t *testing.T) {
	c := New()
	err := c.Build()
	if err != nil {
		t.Fatalf("first Build() failed: %v", err)
	}
	err = c.Build()
	if err != nil {
		t.Fatalf("second Build() failed: %v", err)
	}
}

func TestBuild_InstantiatesEagerServices(t *testing.T) {
	c := New()
	instantiated := false
	For[*testEagerPool](c).Eager().Provider(func(c *Container) (*testEagerPool, error) {
		instantiated = true
		return &testEagerPool{}, nil
	})

	if instantiated {
		t.Error("should not instantiate before Build()")
	}

	err := c.Build()
	if err != nil {
		t.Fatalf("Build() failed: %v", err)
	}

	if !instantiated {
		t.Error("should instantiate at Build()")
	}
}

func TestBuild_EagerError_PropagatesWithContext(t *testing.T) {
	c := New()
	For[*testFailingService](c).Eager().Provider(func(c *Container) (*testFailingService, error) {
		return nil, errors.New("startup failed")
	})

	err := c.Build()
	if err == nil {
		t.Fatal("expected error from Build()")
	}
	if !strings.Contains(err.Error(), "testFailingService") {
		t.Errorf("error should contain service name: %v", err)
	}
	if !strings.Contains(err.Error(), "startup failed") {
		t.Errorf("error should contain root cause: %v", err)
	}
}

func TestBuild_ResolveAfterBuild_ReturnsCachedEagerService(t *testing.T) {
	c := New()
	callCount := 0
	For[*testEagerPool](c).Eager().Provider(func(c *Container) (*testEagerPool, error) {
		callCount++
		return &testEagerPool{id: callCount}, nil
	})

	c.Build()

	// Resolve should return cached instance
	pool1, _ := Resolve[*testEagerPool](c)
	pool2, _ := Resolve[*testEagerPool](c)

	if pool1.id != 1 {
		t.Errorf("expected id 1, got %d", pool1.id)
	}
	if pool1 != pool2 {
		t.Error("should return same cached instance")
	}
	if callCount != 1 {
		t.Errorf("provider should be called exactly once, got %d", callCount)
	}
}

// =============================================================================
// DI-01: Register with generics
// =============================================================================

func TestDI01_RegisterWithGenerics(t *testing.T) {
	c := New()
	err := For[*testDatabase](c).Provider(func(c *Container) (*testDatabase, error) {
		return &testDatabase{}, nil
	})
	if err != nil {
		t.Fatalf("registration failed: %v", err)
	}

	// Verify service is registered
	db, err := Resolve[*testDatabase](c)
	if err != nil {
		t.Fatalf("resolution failed: %v", err)
	}
	if db == nil {
		t.Error("resolved nil database")
	}
}

// =============================================================================
// DI-02: Lazy instantiation by default
// =============================================================================

func TestDI02_LazyInstantiation(t *testing.T) {
	c := New()
	instantiated := false
	For[*testLazyService](c).Provider(func(c *Container) (*testLazyService, error) {
		instantiated = true
		return &testLazyService{}, nil
	})

	if instantiated {
		t.Error("should not instantiate before resolve")
	}

	_, _ = Resolve[*testLazyService](c)
	if !instantiated {
		t.Error("should instantiate on first resolve")
	}
}

// =============================================================================
// DI-03: Error propagation with chain context
// =============================================================================

func TestDI03_ErrorPropagation(t *testing.T) {
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
	if err == nil {
		t.Fatal("expected error")
	}
	// Error should contain chain context
	errStr := err.Error()
	if !strings.Contains(errStr, "testRepo") || !strings.Contains(errStr, "testDB") {
		t.Errorf("error should contain dependency context: %v", err)
	}
	if !strings.Contains(errStr, "connection failed") {
		t.Errorf("error should contain root cause: %v", err)
	}
}

// =============================================================================
// DI-04: Named implementations
// =============================================================================

func TestDI04_NamedImplementations(t *testing.T) {
	c := New()
	For[*testNamedDB](c).Named("primary").Instance(&testNamedDB{name: "primary"})
	For[*testNamedDB](c).Named("replica").Instance(&testNamedDB{name: "replica"})

	primary, err := Resolve[*testNamedDB](c, Named("primary"))
	if err != nil {
		t.Fatalf("failed to resolve primary: %v", err)
	}
	replica, err := Resolve[*testNamedDB](c, Named("replica"))
	if err != nil {
		t.Fatalf("failed to resolve replica: %v", err)
	}

	if primary.name != "primary" {
		t.Errorf("expected primary, got %s", primary.name)
	}
	if replica.name != "replica" {
		t.Errorf("expected replica, got %s", replica.name)
	}
	if primary == replica {
		t.Error("should be different instances")
	}
}

// =============================================================================
// DI-05: Struct field injection
// =============================================================================

func TestDI05_StructFieldInjection(t *testing.T) {
	c := New()
	For[*testInjectDB](c).Instance(&testInjectDB{})
	For[*testHandler](c).Provider(func(c *Container) (*testHandler, error) {
		return &testHandler{}, nil
	})

	h, err := Resolve[*testHandler](c)
	if err != nil {
		t.Fatal(err)
	}
	if h.DB == nil {
		t.Error("DB should be injected")
	}
}

// =============================================================================
// DI-06: Override for testing
// =============================================================================

func TestDI06_Override(t *testing.T) {
	c := New()
	For[*testOverrideService](c).Instance(&testOverrideService{name: "original"})
	For[*testOverrideService](c).Replace().Instance(&testOverrideService{name: "mock"})

	s, _ := Resolve[*testOverrideService](c)
	if s.name != "mock" {
		t.Errorf("expected mock, got %s", s.name)
	}
}

// =============================================================================
// DI-07: Transient services
// =============================================================================

func TestDI07_TransientServices(t *testing.T) {
	c := New()
	counter := 0
	For[*testRequest](c).Transient().Provider(func(c *Container) (*testRequest, error) {
		counter++
		return &testRequest{id: counter}, nil
	})

	r1, _ := Resolve[*testRequest](c)
	r2, _ := Resolve[*testRequest](c)

	if r1.id == r2.id {
		t.Error("should be different instances")
	}
	if r1.id != 1 || r2.id != 2 {
		t.Errorf("expected ids 1 and 2, got %d and %d", r1.id, r2.id)
	}
}

// =============================================================================
// DI-08: Eager services
// =============================================================================

func TestDI08_EagerServices(t *testing.T) {
	c := New()
	instantiated := false
	For[*testPool](c).Eager().Provider(func(c *Container) (*testPool, error) {
		instantiated = true
		return &testPool{}, nil
	})

	if instantiated {
		t.Error("should not instantiate before Build")
	}

	c.Build()

	if !instantiated {
		t.Error("should instantiate at Build")
	}
}

// =============================================================================
// DI-09: Circular dependency detection
// =============================================================================

func TestDI09_CycleDetection(t *testing.T) {
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
	if !errors.Is(err, ErrCycle) {
		t.Errorf("expected ErrCycle, got: %v", err)
	}
}

// =============================================================================
// Integration Tests
// =============================================================================

func TestIntegration_AllRequirements(t *testing.T) {
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
	if eagerStarted {
		t.Error("eager service should not start before Build()")
	}

	// DI-08: Build instantiates eager services
	if err := c.Build(); err != nil {
		t.Fatalf("Build() failed: %v", err)
	}

	if !eagerStarted {
		t.Error("eager service should start at Build()")
	}

	// DI-04: Named resolution
	primary, _ := Resolve[*testAppDB](c, Named("primary"))
	replica, _ := Resolve[*testAppDB](c, Named("replica"))
	if primary.role != "primary" || replica.role != "replica" {
		t.Error("named resolution failed")
	}

	// DI-02: Lazy - already resolved via eager dependency
	// DI-05: Struct field injection
	handler, err := Resolve[*testAppHandler](c)
	if err != nil {
		t.Fatalf("failed to resolve handler: %v", err)
	}
	if handler.Pool == nil {
		t.Error("pool should be injected")
	}
	if handler.Pool.poolSize != 10 {
		t.Error("wrong pool injected")
	}

	// DI-07: Transient - new instance each time
	req1, _ := Resolve[*testAppRequest](c)
	req2, _ := Resolve[*testAppRequest](c)
	if req1.id == req2.id {
		t.Error("transient should create new instances")
	}
}

func TestIntegration_ErrorChainContext(t *testing.T) {
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
	if err == nil {
		t.Fatal("expected error")
	}

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
		if !strings.Contains(errStr, part) {
			t.Errorf("error should contain '%s': %v", part, err)
		}
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
