package gaz_test

import (
	"testing"

	"github.com/stretchr/testify/suite"

	"github.com/petabytecl/gaz"
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

// RegistrationSuite tests the registration API (For[T], fluent builder).
type RegistrationSuite struct {
	suite.Suite
}

func TestRegistrationSuite(t *testing.T) {
	suite.Run(t, new(RegistrationSuite))
}

func (s *RegistrationSuite) TestFor_Provider_RegistersService() {
	c := gaz.NewContainer()

	err := gaz.For[*testService](c).Provider(func(_ *gaz.Container) (*testService, error) {
		return &testService{id: 42}, nil
	})

	s.NoError(err)
}

func (s *RegistrationSuite) TestFor_ProviderFunc_RegistersService() {
	c := gaz.NewContainer()

	err := gaz.For[*testService](c).ProviderFunc(func(_ *gaz.Container) *testService {
		return &testService{id: 42}
	})

	s.NoError(err)
}

func (s *RegistrationSuite) TestFor_Instance_RegistersValue() {
	c := gaz.NewContainer()

	cfg := &testConfig{value: "test-value"}
	err := gaz.For[*testConfig](c).Instance(cfg)

	s.NoError(err)
}

func (s *RegistrationSuite) TestFor_Duplicate_AllowsMultiple() {
	c := gaz.NewContainer()

	// First registration should succeed
	err := gaz.For[*testService](c).Provider(func(_ *gaz.Container) (*testService, error) {
		return &testService{id: 1}, nil
	})
	s.Require().NoError(err, "first registration failed")

	// Second registration of same type should succeed (append)
	err = gaz.For[*testService](c).Provider(func(_ *gaz.Container) (*testService, error) {
		return &testService{id: 2}, nil
	})
	s.Require().NoError(err, "second registration failed")

	// Resolution should be ambiguous
	_, err = gaz.Resolve[*testService](c)
	s.Require().ErrorIs(err, gaz.ErrDIAmbiguous)
}

func (s *RegistrationSuite) TestFor_Duplicate_Instance_AllowsMultiple() {
	c := gaz.NewContainer()

	// First registration should succeed
	err := gaz.For[*testConfig](c).Instance(&testConfig{value: "first"})
	s.Require().NoError(err, "first registration failed")

	// Second registration of same type should succeed (append)
	err = gaz.For[*testConfig](c).Instance(&testConfig{value: "second"})
	s.Require().NoError(err, "second registration failed")

	// Resolution should be ambiguous
	_, err = gaz.Resolve[*testConfig](c)
	s.Require().ErrorIs(err, gaz.ErrDIAmbiguous)
}

func (s *RegistrationSuite) TestFor_Replace_AllowsOverwrite() {
	c := gaz.NewContainer()

	// First registration
	err := gaz.For[*testService](c).Provider(func(_ *gaz.Container) (*testService, error) {
		return &testService{id: 1}, nil
	})
	s.Require().NoError(err, "first registration failed")

	// Replace() should allow overwriting
	err = gaz.For[*testService](c).Replace().Provider(func(_ *gaz.Container) (*testService, error) {
		return &testService{id: 2}, nil
	})
	s.NoError(err, "expected no error with Replace()")
}

func (s *RegistrationSuite) TestFor_Replace_Instance_AllowsOverwrite() {
	c := gaz.NewContainer()

	// First registration
	err := gaz.For[*testConfig](c).Instance(&testConfig{value: "first"})
	s.Require().NoError(err, "first registration failed")

	// Replace() should allow overwriting with Instance
	err = gaz.For[*testConfig](c).Replace().Instance(&testConfig{value: "replaced"})
	s.NoError(err, "expected no error with Replace()")
}

func (s *RegistrationSuite) TestFor_Named_CreatesSeparateEntry() {
	c := gaz.NewContainer()

	// Register "primary" named DB
	err := gaz.For[*testDB](c).Named("primary").Provider(func(_ *gaz.Container) (*testDB, error) {
		return &testDB{name: "primary"}, nil
	})
	s.Require().NoError(err, "primary registration failed")

	// Register "replica" named DB - should not conflict
	err = gaz.For[*testDB](c).Named("replica").Provider(func(_ *gaz.Container) (*testDB, error) {
		return &testDB{name: "replica"}, nil
	})
	s.NoError(err, "expected no error for differently named services")
}

func (s *RegistrationSuite) TestFor_Named_DuplicateSameName_AllowsMultiple() {
	c := gaz.NewContainer()

	// Register "primary" named DB
	err := gaz.For[*testDB](c).Named("primary").Provider(func(_ *gaz.Container) (*testDB, error) {
		return &testDB{name: "primary"}, nil
	})
	s.Require().NoError(err, "first registration failed")

	// Register another "primary" - should succeed
	err = gaz.For[*testDB](c).Named("primary").Provider(func(_ *gaz.Container) (*testDB, error) {
		return &testDB{name: "primary-2"}, nil
	})
	s.Require().NoError(err, "second registration failed")

	// Resolution should be ambiguous
	_, err = gaz.Resolve[*testDB](c, gaz.Named("primary"))
	s.Require().ErrorIs(err, gaz.ErrDIAmbiguous)
}

func (s *RegistrationSuite) TestFor_Transient_CreatesTransientService() {
	c := gaz.NewContainer()

	// Registration with Transient() should succeed
	err := gaz.For[*testService](
		c,
	).Transient().
		Provider(func(_ *gaz.Container) (*testService, error) {
			return &testService{id: 99}, nil
		})
	s.NoError(err)
	// Note: Verification of transient behavior (new instance per resolve) is tested in resolution tests
}

func (s *RegistrationSuite) TestFor_Eager_CreatesEagerService() {
	c := gaz.NewContainer()

	// Registration with Eager() should succeed
	err := gaz.For[*testService](c).Eager().Provider(func(_ *gaz.Container) (*testService, error) {
		return &testService{id: 100}, nil
	})
	s.NoError(err)
	// Note: Verification of eager behavior (instantiate at Build) is tested in Build tests
}

func (s *RegistrationSuite) TestFor_ChainedOptions_Work() {
	c := gaz.NewContainer()

	// All options can be chained together
	err := gaz.For[*testDB](c).
		Named("analytics").
		Eager().
		Replace(). // Replace() on first registration is a no-op
		Provider(func(_ *gaz.Container) (*testDB, error) {
			return &testDB{name: "analytics"}, nil
		})
	s.NoError(err, "expected no error with chained options")
}
