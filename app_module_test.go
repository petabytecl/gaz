package gaz

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/suite"
)

type ModuleSuite struct {
	suite.Suite
}

func TestModuleSuite(t *testing.T) {
	suite.Run(t, new(ModuleSuite))
}

// moduleTestDB is a test helper type for module tests.
type moduleTestDB struct {
	name string
}

func (s *ModuleSuite) TestModuleRegistersProviders() {
	app := New()

	app.Module("database",
		func(c *Container) error {
			return For[*moduleTestDB](c).ProviderFunc(func(_ *Container) *moduleTestDB {
				return &moduleTestDB{name: "test"}
			})
		},
	)

	err := app.Build()
	s.Require().NoError(err)

	db, resolveErr := Resolve[*moduleTestDB](app.container)
	s.Require().NoError(resolveErr)
	s.Equal("test", db.name)
}

func (s *ModuleSuite) TestModuleDuplicateNameError() {
	app := New()

	app.Module("database",
		func(_ *Container) error { return nil },
	).Module("database", // Duplicate!
		func(_ *Container) error { return nil },
	)

	err := app.Build()
	s.Require().Error(err)
	s.Require().ErrorIs(err, ErrModuleDuplicate)
	s.Contains(err.Error(), "database")
}

func (s *ModuleSuite) TestModuleAfterBuildPanics() {
	app := New()
	s.Require().NoError(app.Build())

	s.Panics(func() {
		app.Module("late", func(_ *Container) error { return nil })
	})
}

func (s *ModuleSuite) TestModuleErrorsAggregated() {
	app := New()

	// Modules returning explicit errors
	app.Module("first",
		func(c *Container) error {
			return errors.New("error in first")
		},
	).Module("second",
		func(c *Container) error {
			return errors.New("error in second")
		},
	)

	err := app.Build()
	s.Require().Error(err)
	s.Contains(err.Error(), "first")  // Module name in error
	s.Contains(err.Error(), "second") // Module name in error
}

func (s *ModuleSuite) TestModuleChaining() {
	app := New()

	result := app.Module("a", func(_ *Container) error { return nil }).
		Module("b", func(_ *Container) error { return nil }).
		Module("c", func(_ *Container) error { return nil })

	s.Same(app, result) // Chaining returns same app
	s.Require().NoError(app.Build())
}

func (s *ModuleSuite) TestEmptyModule() {
	app := New()

	// Empty module is valid
	app.Module("empty")

	s.Require().NoError(app.Build())
	s.True(app.modules["empty"])
}

func (s *ModuleSuite) TestMultipleModulesWithDifferentNames() {
	app := New()

	app.Module("database",
		func(c *Container) error {
			return For[*moduleTestDB](c).ProviderFunc(func(_ *Container) *moduleTestDB {
				return &moduleTestDB{name: "db"}
			})
		},
	).Module("cache",
		func(c *Container) error {
			return For[*moduleTestCache](c).ProviderFunc(func(_ *Container) *moduleTestCache {
				return &moduleTestCache{size: 100}
			})
		},
	)

	err := app.Build()
	s.Require().NoError(err)

	db, dbErr := Resolve[*moduleTestDB](app.container)
	s.Require().NoError(dbErr)
	s.Equal("db", db.name)

	cache, cacheErr := Resolve[*moduleTestCache](app.container)
	s.Require().NoError(cacheErr)
	s.Equal(100, cache.size)
}

// moduleTestCache is a test helper type for module tests.
type moduleTestCache struct {
	size int
}

func (s *ModuleSuite) TestModuleRegistrationErrorIncludesModuleName() {
	app := New()

	app.Module("mymodule",
		func(_ *Container) error {
			return errors.New("registration failed")
		},
	)

	err := app.Build()
	s.Require().Error(err)
	s.Contains(err.Error(), "mymodule")
	s.Contains(err.Error(), "registration failed")
}
