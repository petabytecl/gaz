package gaz

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

type ModuleBuilderSuite struct {
	suite.Suite
}

func TestModuleBuilderSuite(t *testing.T) {
	suite.Run(t, new(ModuleBuilderSuite))
}

// moduleBuilderTestService is a test helper type.
type moduleBuilderTestService struct {
	name string
}

// moduleBuilderTestCache is a test helper type.
type moduleBuilderTestCache struct {
	size int
}

func (s *ModuleBuilderSuite) TestNewModule_ReturnsBuilder() {
	mb := NewModule("test")
	s.Require().NotNil(mb)
}

func (s *ModuleBuilderSuite) TestModuleBuilder_Provide_Chainable() {
	mb := NewModule("test").Provide(func(c *Container) error { return nil })
	s.Require().NotNil(mb)
}

func (s *ModuleBuilderSuite) TestModuleBuilder_Build_ReturnsModule() {
	m := NewModule("test").Build()
	s.Require().NotNil(m)
	s.Equal("test", m.Name())
}

func (s *ModuleBuilderSuite) TestModuleBuilder_Provide_RegistersProviders() {
	m := NewModule("test").
		Provide(func(c *Container) error {
			return For[*moduleBuilderTestService](c).ProviderFunc(func(_ *Container) *moduleBuilderTestService {
				return &moduleBuilderTestService{name: "test"}
			})
		}).
		Build()

	app := New()
	app.Use(m)

	err := app.Build()
	s.Require().NoError(err)

	svc, resolveErr := Resolve[*moduleBuilderTestService](app.container)
	s.Require().NoError(resolveErr)
	s.Equal("test", svc.name)
}

func (s *ModuleBuilderSuite) TestModuleBuilder_MultipleProviders() {
	m := NewModule("test").
		Provide(
			func(c *Container) error {
				return For[*moduleBuilderTestService](c).ProviderFunc(func(_ *Container) *moduleBuilderTestService {
					return &moduleBuilderTestService{name: "svc"}
				})
			},
			func(c *Container) error {
				return For[*moduleBuilderTestCache](c).ProviderFunc(func(_ *Container) *moduleBuilderTestCache {
					return &moduleBuilderTestCache{size: 100}
				})
			},
		).
		Build()

	app := New()
	app.Use(m)

	err := app.Build()
	s.Require().NoError(err)

	svc, svcErr := Resolve[*moduleBuilderTestService](app.container)
	s.Require().NoError(svcErr)
	s.Equal("svc", svc.name)

	cache, cacheErr := Resolve[*moduleBuilderTestCache](app.container)
	s.Require().NoError(cacheErr)
	s.Equal(100, cache.size)
}

func (s *ModuleBuilderSuite) TestModuleBuilder_Use_BundlesChildModule() {
	child := NewModule("child").
		Provide(func(c *Container) error {
			return For[*moduleBuilderTestService](c).ProviderFunc(func(_ *Container) *moduleBuilderTestService {
				return &moduleBuilderTestService{name: "from-child"}
			})
		}).
		Build()

	parent := NewModule("parent").Use(child).Build()

	app := New()
	app.Use(parent)

	err := app.Build()
	s.Require().NoError(err)

	// Child's provider should be registered
	svc, resolveErr := Resolve[*moduleBuilderTestService](app.container)
	s.Require().NoError(resolveErr)
	s.Equal("from-child", svc.name)
}

func (s *ModuleBuilderSuite) TestModuleBuilder_Apply_AppliesChildModulesFirst() {
	// Track order of application
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

	s.Equal([]string{"child", "parent"}, order)
}

func (s *ModuleBuilderSuite) TestModuleBuilder_MultipleChildModules() {
	var order []string

	child1 := NewModule("child1").
		Provide(func(c *Container) error {
			order = append(order, "child1")
			return nil
		}).
		Build()

	child2 := NewModule("child2").
		Provide(func(c *Container) error {
			order = append(order, "child2")
			return nil
		}).
		Build()

	parent := NewModule("parent").
		Use(child1).
		Use(child2).
		Provide(func(c *Container) error {
			order = append(order, "parent")
			return nil
		}).
		Build()

	app := New()
	app.Use(parent)

	// Children should be applied in order before parent
	s.Equal([]string{"child1", "child2", "parent"}, order)
}

func (s *ModuleBuilderSuite) TestModuleBuilder_NestedChildModules() {
	var order []string

	grandchild := NewModule("grandchild").
		Provide(func(c *Container) error {
			order = append(order, "grandchild")
			return nil
		}).
		Build()

	child := NewModule("child").
		Use(grandchild).
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

	// Should apply deepest first
	s.Equal([]string{"grandchild", "child", "parent"}, order)
}

func (s *ModuleBuilderSuite) TestModuleBuilder_ProviderError() {
	m := NewModule("test").
		Provide(func(c *Container) error {
			// First registration succeeds
			return For[*moduleBuilderTestService](c).ProviderFunc(func(_ *Container) *moduleBuilderTestService {
				return &moduleBuilderTestService{name: "first"}
			})
		}).
		Build()

	m2 := NewModule("test2").
		Provide(func(c *Container) error {
			// Second registration of same type should fail
			return For[*moduleBuilderTestService](c).ProviderFunc(func(_ *Container) *moduleBuilderTestService {
				return &moduleBuilderTestService{name: "second"}
			})
		}).
		Build()

	app := New()
	app.Use(m).Use(m2)

	err := app.Build()
	s.Require().Error(err)
	s.ErrorIs(err, ErrDuplicate)
}

func TestNewModule_ReturnsBuilder(t *testing.T) {
	mb := NewModule("test")
	require.NotNil(t, mb)
}

func TestModuleBuilder_Provide_Chainable(t *testing.T) {
	mb := NewModule("test").Provide(func(c *Container) error { return nil })
	require.NotNil(t, mb)
}

func TestModuleBuilder_Build_ReturnsModule(t *testing.T) {
	m := NewModule("test").Build()
	require.NotNil(t, m)
	assert.Equal(t, "test", m.Name())
}
