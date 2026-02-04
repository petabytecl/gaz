package gateway

import (
	"testing"

	"github.com/stretchr/testify/suite"

	"github.com/petabytecl/gaz"
	"github.com/petabytecl/gaz/di"
)

// ModuleTestSuite tests Gateway module registration.
type ModuleTestSuite struct {
	suite.Suite
}

func TestModuleTestSuite(t *testing.T) {
	suite.Run(t, new(ModuleTestSuite))
}

func (s *ModuleTestSuite) TestNewModule_Defaults() {
	app := gaz.New()

	module := NewModule()
	err := module.Apply(app)
	s.Require().NoError(err)

	err = app.Build()
	s.Require().NoError(err)

	c := app.Container()

	// Verify Gateway was registered.
	s.Require().True(di.Has[*Gateway](c))

	// Verify Config was registered.
	s.Require().True(di.Has[Config](c))

	cfg, err := di.Resolve[Config](c)
	s.Require().NoError(err)
	s.Require().Equal(DefaultPort, cfg.Port)
}

func (s *ModuleTestSuite) TestNewModule_RegistersConfig() {
	app := gaz.New()

	module := NewModule()
	err := module.Apply(app)
	s.Require().NoError(err)

	err = app.Build()
	s.Require().NoError(err)

	c := app.Container()
	s.Require().True(di.Has[Config](c))
}

func (s *ModuleTestSuite) TestNewModule_RegistersGateway() {
	app := gaz.New()

	module := NewModule()
	err := module.Apply(app)
	s.Require().NoError(err)

	err = app.Build()
	s.Require().NoError(err)

	c := app.Container()
	s.Require().True(di.Has[*Gateway](c))
}
