package di

import (
	"context"
	"testing"

	"github.com/stretchr/testify/suite"
)

// =============================================================================
// RegistrationSuite
// =============================================================================

type RegistrationSuite struct {
	suite.Suite
}

func TestRegistrationSuite(t *testing.T) {
	suite.Run(t, new(RegistrationSuite))
}

// testRegService is a simple test service type.
type testRegService struct {
	id int
}

// testRegConfig is a simple configuration type for testing.
type testRegConfig struct {
	value string
}

// testRegDB simulates a database connection for named service tests.
type testRegDB struct {
	name string
}

// =============================================================================
// For[T]() Tests
// =============================================================================

func (s *RegistrationSuite) TestFor_Provider_RegistersService() {
	c := New()

	err := For[*testRegService](c).Provider(func(_ *Container) (*testRegService, error) {
		return &testRegService{id: 42}, nil
	})

	s.NoError(err)
}

func (s *RegistrationSuite) TestFor_ProviderFunc_RegistersService() {
	c := New()

	err := For[*testRegService](c).ProviderFunc(func(_ *Container) *testRegService {
		return &testRegService{id: 42}
	})

	s.NoError(err)
}

func (s *RegistrationSuite) TestFor_Instance_RegistersValue() {
	c := New()

	cfg := &testRegConfig{value: "test-value"}
	err := For[*testRegConfig](c).Instance(cfg)

	s.NoError(err)
}

func (s *RegistrationSuite) TestFor_Duplicate_ReturnsError() {
	c := New()

	// First registration should succeed
	err := For[*testRegService](c).Provider(func(_ *Container) (*testRegService, error) {
		return &testRegService{id: 1}, nil
	})
	s.Require().NoError(err, "first registration failed")

	// Second registration of same type should return ErrDuplicate
	err = For[*testRegService](c).Provider(func(_ *Container) (*testRegService, error) {
		return &testRegService{id: 2}, nil
	})
	s.Require().ErrorIs(err, ErrDuplicate)
}

func (s *RegistrationSuite) TestFor_Duplicate_Instance_ReturnsError() {
	c := New()

	// First registration should succeed
	err := For[*testRegConfig](c).Instance(&testRegConfig{value: "first"})
	s.Require().NoError(err, "first registration failed")

	// Second registration of same type should return ErrDuplicate
	err = For[*testRegConfig](c).Instance(&testRegConfig{value: "second"})
	s.Require().ErrorIs(err, ErrDuplicate)
}

// =============================================================================
// Named() Tests
// =============================================================================

func (s *RegistrationSuite) TestFor_Named_CreatesSeparateEntry() {
	c := New()

	// Register "primary" named DB
	err := For[*testRegDB](c).Named("primary").Provider(func(_ *Container) (*testRegDB, error) {
		return &testRegDB{name: "primary"}, nil
	})
	s.Require().NoError(err, "primary registration failed")

	// Register "replica" named DB - should not conflict
	err = For[*testRegDB](c).Named("replica").Provider(func(_ *Container) (*testRegDB, error) {
		return &testRegDB{name: "replica"}, nil
	})
	s.NoError(err, "expected no error for differently named services")
}

func (s *RegistrationSuite) TestFor_Named_DuplicateSameName_ReturnsError() {
	c := New()

	// Register "primary" named DB
	err := For[*testRegDB](c).Named("primary").Provider(func(_ *Container) (*testRegDB, error) {
		return &testRegDB{name: "primary"}, nil
	})
	s.Require().NoError(err, "first registration failed")

	// Register another "primary" - should return ErrDuplicate
	err = For[*testRegDB](c).Named("primary").Provider(func(_ *Container) (*testRegDB, error) {
		return &testRegDB{name: "primary-2"}, nil
	})
	s.Require().ErrorIs(err, ErrDuplicate)
}

// =============================================================================
// Transient() Tests
// =============================================================================

func (s *RegistrationSuite) TestFor_Transient_CreatesTransientService() {
	c := New()

	// Registration with Transient() should succeed
	err := For[*testRegService](
		c,
	).Transient().
		Provider(func(_ *Container) (*testRegService, error) {
			return &testRegService{id: 99}, nil
		})
	s.NoError(err)

	// Verify transient behavior
	svc1, err := Resolve[*testRegService](c)
	s.Require().NoError(err)
	svc2, err := Resolve[*testRegService](c)
	s.Require().NoError(err)

	// Should be different instances
	s.NotSame(svc1, svc2, "transient should create new instances each time")
}

// =============================================================================
// Eager() Tests
// =============================================================================

func (s *RegistrationSuite) TestFor_Eager_CreatesEagerService() {
	c := New()

	// Registration with Eager() should succeed
	instantiated := false
	err := For[*testRegService](c).Eager().Provider(func(_ *Container) (*testRegService, error) {
		instantiated = true
		return &testRegService{id: 100}, nil
	})
	s.NoError(err)

	// Should not instantiate until Build()
	s.False(instantiated, "should not instantiate before Build")

	// Verify eager behavior at Build()
	s.Require().NoError(c.Build())
	s.True(instantiated, "should instantiate at Build")
}

// =============================================================================
// Replace() Tests
// =============================================================================

func (s *RegistrationSuite) TestFor_Replace_AllowsOverwrite() {
	c := New()

	// First registration
	err := For[*testRegService](c).Provider(func(_ *Container) (*testRegService, error) {
		return &testRegService{id: 1}, nil
	})
	s.Require().NoError(err, "first registration failed")

	// Replace() should allow overwriting
	err = For[*testRegService](c).Replace().Provider(func(_ *Container) (*testRegService, error) {
		return &testRegService{id: 2}, nil
	})
	s.NoError(err, "expected no error with Replace()")

	// Verify replaced service
	svc, err := Resolve[*testRegService](c)
	s.Require().NoError(err)
	s.Equal(2, svc.id, "should get replaced service")
}

func (s *RegistrationSuite) TestFor_Replace_Instance_AllowsOverwrite() {
	c := New()

	// First registration
	err := For[*testRegConfig](c).Instance(&testRegConfig{value: "first"})
	s.Require().NoError(err, "first registration failed")

	// Replace() should allow overwriting with Instance
	err = For[*testRegConfig](c).Replace().Instance(&testRegConfig{value: "replaced"})
	s.NoError(err, "expected no error with Replace()")

	// Verify replaced service
	cfg, err := Resolve[*testRegConfig](c)
	s.Require().NoError(err)
	s.Equal("replaced", cfg.value, "should get replaced instance")
}

// =============================================================================
// OnStart()/OnStop() Hooks Tests
// =============================================================================

func (s *RegistrationSuite) TestFor_OnStart_HookCalled() {
	c := New()

	startCalled := false
	err := For[*testRegService](c).
		OnStart(func(_ context.Context, _ *testRegService) error {
			startCalled = true
			return nil
		}).
		Provider(func(_ *Container) (*testRegService, error) {
			return &testRegService{id: 1}, nil
		})
	s.Require().NoError(err)

	// Resolve to build
	_, err = Resolve[*testRegService](c)
	s.Require().NoError(err)

	// Access wrapper and call start
	wrapper, found := c.GetService(TypeName[*testRegService]())
	s.Require().True(found)

	err = wrapper.Start(context.Background())
	s.Require().NoError(err)
	s.True(startCalled, "OnStart hook should have been called")
}

func (s *RegistrationSuite) TestFor_OnStop_HookCalled() {
	c := New()

	stopCalled := false
	err := For[*testRegService](c).
		OnStop(func(_ context.Context, _ *testRegService) error {
			stopCalled = true
			return nil
		}).
		Provider(func(_ *Container) (*testRegService, error) {
			return &testRegService{id: 1}, nil
		})
	s.Require().NoError(err)

	// Resolve to build
	_, err = Resolve[*testRegService](c)
	s.Require().NoError(err)

	// Access wrapper and call stop
	wrapper, found := c.GetService(TypeName[*testRegService]())
	s.Require().True(found)

	err = wrapper.Stop(context.Background())
	s.Require().NoError(err)
	s.True(stopCalled, "OnStop hook should have been called")
}

func (s *RegistrationSuite) TestFor_BothHooks_CalledCorrectly() {
	c := New()

	callOrder := make([]string, 0)
	err := For[*testRegService](c).
		OnStart(func(_ context.Context, _ *testRegService) error {
			callOrder = append(callOrder, "start")
			return nil
		}).
		OnStop(func(_ context.Context, _ *testRegService) error {
			callOrder = append(callOrder, "stop")
			return nil
		}).
		Provider(func(_ *Container) (*testRegService, error) {
			return &testRegService{id: 1}, nil
		})
	s.Require().NoError(err)

	// Resolve to build
	_, err = Resolve[*testRegService](c)
	s.Require().NoError(err)

	// Access wrapper
	wrapper, found := c.GetService(TypeName[*testRegService]())
	s.Require().True(found)

	// Call start and stop
	s.Require().NoError(wrapper.Start(context.Background()))
	s.Require().NoError(wrapper.Stop(context.Background()))

	s.Equal([]string{"start", "stop"}, callOrder)
}

// =============================================================================
// Chained Options Tests
// =============================================================================

func (s *RegistrationSuite) TestFor_ChainedOptions_Work() {
	c := New()

	// All options can be chained together
	err := For[*testRegDB](c).
		Named("analytics").
		Eager().
		Replace(). // Replace() on first registration is a no-op
		Provider(func(_ *Container) (*testRegDB, error) {
			return &testRegDB{name: "analytics"}, nil
		})
	s.NoError(err, "expected no error with chained options")
}

// =============================================================================
// Provider Error Tests
// =============================================================================

func (s *RegistrationSuite) TestFor_Provider_ReturnsProviderError() {
	c := New()

	providerErr := ErrInvalidProvider

	err := For[*testRegService](c).Provider(func(_ *Container) (*testRegService, error) {
		return nil, providerErr
	})
	s.Require().NoError(err, "registration should succeed")

	// Error occurs on resolution
	_, resolveErr := Resolve[*testRegService](c)
	s.Require().ErrorIs(resolveErr, providerErr)
}
