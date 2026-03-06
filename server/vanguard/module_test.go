package vanguard

import (
	"testing"

	"github.com/stretchr/testify/suite"
)

// ModuleTestSuite tests the Vanguard module registration.
type ModuleTestSuite struct {
	suite.Suite
}

func TestModuleTestSuite(t *testing.T) {
	suite.Run(t, new(ModuleTestSuite))
}

func (s *ModuleTestSuite) TestNewModuleCreatesModule() {
	mod := NewModule()
	s.Require().NotNil(mod)
}

func (s *ModuleTestSuite) TestNewModuleName() {
	mod := NewModule()
	s.Equal("vanguard", mod.Name())
}

func (s *ModuleTestSuite) TestProvideConfigDefaultValues() {
	cfg := DefaultConfig()
	s.Equal(DefaultPort, cfg.Port)
	s.Equal("server", cfg.Namespace())
	s.True(cfg.Reflection)
	s.True(cfg.HealthEnabled)
	s.False(cfg.DevMode)
}
