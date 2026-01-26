package gaz_test

import (
	"errors"
	"testing"

	"github.com/petabyte/gaz"
)

// testService is a simple test service type.
type testService struct {
	id int
}

// testConfig is a simple configuration type for testing.
type testConfig struct {
	value string
}

// testDB simulates a database connection for named service tests.
type testDB struct {
	name string
}

func TestFor_Provider_RegistersService(t *testing.T) {
	c := gaz.New()

	err := gaz.For[*testService](c).Provider(func(c *gaz.Container) (*testService, error) {
		return &testService{id: 42}, nil
	})

	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}
}

func TestFor_ProviderFunc_RegistersService(t *testing.T) {
	c := gaz.New()

	err := gaz.For[*testService](c).ProviderFunc(func(c *gaz.Container) *testService {
		return &testService{id: 42}
	})

	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}
}

func TestFor_Instance_RegistersValue(t *testing.T) {
	c := gaz.New()

	cfg := &testConfig{value: "test-value"}
	err := gaz.For[*testConfig](c).Instance(cfg)

	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}
}

func TestFor_Duplicate_ReturnsError(t *testing.T) {
	c := gaz.New()

	// First registration should succeed
	err := gaz.For[*testService](c).Provider(func(c *gaz.Container) (*testService, error) {
		return &testService{id: 1}, nil
	})
	if err != nil {
		t.Fatalf("first registration failed: %v", err)
	}

	// Second registration of same type should return ErrDuplicate
	err = gaz.For[*testService](c).Provider(func(c *gaz.Container) (*testService, error) {
		return &testService{id: 2}, nil
	})
	if !errors.Is(err, gaz.ErrDuplicate) {
		t.Errorf("expected ErrDuplicate, got %v", err)
	}
}

func TestFor_Duplicate_Instance_ReturnsError(t *testing.T) {
	c := gaz.New()

	// First registration should succeed
	err := gaz.For[*testConfig](c).Instance(&testConfig{value: "first"})
	if err != nil {
		t.Fatalf("first registration failed: %v", err)
	}

	// Second registration of same type should return ErrDuplicate
	err = gaz.For[*testConfig](c).Instance(&testConfig{value: "second"})
	if !errors.Is(err, gaz.ErrDuplicate) {
		t.Errorf("expected ErrDuplicate, got %v", err)
	}
}

func TestFor_Replace_AllowsOverwrite(t *testing.T) {
	c := gaz.New()

	// First registration
	err := gaz.For[*testService](c).Provider(func(c *gaz.Container) (*testService, error) {
		return &testService{id: 1}, nil
	})
	if err != nil {
		t.Fatalf("first registration failed: %v", err)
	}

	// Replace() should allow overwriting
	err = gaz.For[*testService](c).Replace().Provider(func(c *gaz.Container) (*testService, error) {
		return &testService{id: 2}, nil
	})
	if err != nil {
		t.Errorf("expected no error with Replace(), got %v", err)
	}
}

func TestFor_Replace_Instance_AllowsOverwrite(t *testing.T) {
	c := gaz.New()

	// First registration
	err := gaz.For[*testConfig](c).Instance(&testConfig{value: "first"})
	if err != nil {
		t.Fatalf("first registration failed: %v", err)
	}

	// Replace() should allow overwriting with Instance
	err = gaz.For[*testConfig](c).Replace().Instance(&testConfig{value: "replaced"})
	if err != nil {
		t.Errorf("expected no error with Replace(), got %v", err)
	}
}

func TestFor_Named_CreatesSeparateEntry(t *testing.T) {
	c := gaz.New()

	// Register "primary" named DB
	err := gaz.For[*testDB](c).Named("primary").Provider(func(c *gaz.Container) (*testDB, error) {
		return &testDB{name: "primary"}, nil
	})
	if err != nil {
		t.Fatalf("primary registration failed: %v", err)
	}

	// Register "replica" named DB - should not conflict
	err = gaz.For[*testDB](c).Named("replica").Provider(func(c *gaz.Container) (*testDB, error) {
		return &testDB{name: "replica"}, nil
	})
	if err != nil {
		t.Errorf("expected no error for differently named services, got %v", err)
	}
}

func TestFor_Named_DuplicateSameName_ReturnsError(t *testing.T) {
	c := gaz.New()

	// Register "primary" named DB
	err := gaz.For[*testDB](c).Named("primary").Provider(func(c *gaz.Container) (*testDB, error) {
		return &testDB{name: "primary"}, nil
	})
	if err != nil {
		t.Fatalf("first registration failed: %v", err)
	}

	// Register another "primary" - should return ErrDuplicate
	err = gaz.For[*testDB](c).Named("primary").Provider(func(c *gaz.Container) (*testDB, error) {
		return &testDB{name: "primary-2"}, nil
	})
	if !errors.Is(err, gaz.ErrDuplicate) {
		t.Errorf("expected ErrDuplicate for same name, got %v", err)
	}
}

func TestFor_Transient_CreatesTransientService(t *testing.T) {
	c := gaz.New()

	// Registration with Transient() should succeed
	err := gaz.For[*testService](c).Transient().Provider(func(c *gaz.Container) (*testService, error) {
		return &testService{id: 99}, nil
	})
	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}
	// Note: Verification of transient behavior (new instance per resolve) is tested in resolution tests
}

func TestFor_Eager_CreatesEagerService(t *testing.T) {
	c := gaz.New()

	// Registration with Eager() should succeed
	err := gaz.For[*testService](c).Eager().Provider(func(c *gaz.Container) (*testService, error) {
		return &testService{id: 100}, nil
	})
	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}
	// Note: Verification of eager behavior (instantiate at Build) is tested in Build tests
}

func TestFor_ChainedOptions_Work(t *testing.T) {
	c := gaz.New()

	// All options can be chained together
	err := gaz.For[*testDB](c).
		Named("analytics").
		Eager().
		Replace(). // Replace() on first registration is a no-op
		Provider(func(c *gaz.Container) (*testDB, error) {
			return &testDB{name: "analytics"}, nil
		})
	if err != nil {
		t.Errorf("expected no error with chained options, got %v", err)
	}
}
