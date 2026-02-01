package gaz

import (
	"testing"

	"github.com/stretchr/testify/suite"
)

type AppUseSuite struct {
	suite.Suite
}

func TestAppUseSuite(t *testing.T) {
	suite.Run(t, new(AppUseSuite))
}

// appUseTestService is a test helper type.
type appUseTestService struct {
	name string
}

func (s *AppUseSuite) TestApp_Use_AppliesModule() {
	applied := false
	m := NewModule("test").
		Provide(func(c *Container) error {
			applied = true
			return nil
		}).
		Build()

	New().Use(m)
	s.True(applied)
}

func (s *AppUseSuite) TestApp_Use_Chainable() {
	m1 := NewModule("m1").Build()
	m2 := NewModule("m2").Build()
	m3 := NewModule("m3").Build()

	app := New()
	result := app.Use(m1).Use(m2).Use(m3)

	s.Same(app, result) // Chaining returns same app
	s.Require().NoError(app.Build())
}

func (s *AppUseSuite) TestApp_Use_DuplicateModuleReturnsError() {
	m := NewModule("test").Build()
	app := New().Use(m).Use(m) // same module name twice

	err := app.Build()
	s.Require().Error(err)
	s.ErrorIs(err, ErrModuleDuplicate)
	s.Contains(err.Error(), "test")
}

func (s *AppUseSuite) TestApp_Use_PanicsAfterBuild() {
	m := NewModule("test").Build()
	app := New()
	s.Require().NoError(app.Build())

	s.Panics(func() { app.Use(m) })
}

func (s *AppUseSuite) TestApp_Use_RegistersProviders() {
	m := NewModule("test").
		Provide(func(c *Container) error {
			return For[*appUseTestService](c).ProviderFunc(func(_ *Container) *appUseTestService {
				return &appUseTestService{name: "from-module"}
			})
		}).
		Build()

	app := New()
	app.Use(m)

	err := app.Build()
	s.Require().NoError(err)

	svc, resolveErr := Resolve[*appUseTestService](app.container)
	s.Require().NoError(resolveErr)
	s.Equal("from-module", svc.name)
}

func (s *AppUseSuite) TestApp_Use_TracksModuleNames() {
	m1 := NewModule("database").Build()
	m2 := NewModule("cache").Build()

	app := New()
	app.Use(m1).Use(m2)

	s.True(app.modules["database"])
	s.True(app.modules["cache"])
}

func (s *AppUseSuite) TestApp_Use_ModuleWithChildModules() {
	var order []string

	child := NewModule("child").
		Provide(func(c *Container) error {
			order = append(order, "child")
			return nil
		}).
		Build()

	parent := NewModule("parent").
		Use(child).
		Provide(func(c *Container) error {
			order = append(order, "parent")
			return nil
		}).
		Build()

	app := New()
	app.Use(parent)

	// Child should be applied before parent
	s.Equal([]string{"child", "parent"}, order)
	// Both module names should be tracked
	s.True(app.modules["parent"])
	s.True(app.modules["child"])
}

func (s *AppUseSuite) TestApp_Use_ModuleErrorCollected() {
	// Register something first to cause a duplicate error
	app := New()
	err := For[*appUseTestService](app.container).ProviderFunc(func(_ *Container) *appUseTestService {
		return &appUseTestService{name: "existing"}
	})
	s.Require().NoError(err)

	// Module tries to register same type
	m := NewModule("conflict").
		Provide(func(c *Container) error {
			return For[*appUseTestService](c).ProviderFunc(func(_ *Container) *appUseTestService {
				return &appUseTestService{name: "conflict"}
			})
		}).
		Build()

	app.Use(m)

	buildErr := app.Build()
	s.Require().Error(buildErr)
	s.Contains(buildErr.Error(), "conflict")
}
