package health

import (
	"testing"

	"github.com/spf13/pflag"
	"github.com/stretchr/testify/suite"
)

// ConfigTestSuite tests the health configuration.
type ConfigTestSuite struct {
	suite.Suite
}

func TestConfigTestSuite(t *testing.T) {
	suite.Run(t, new(ConfigTestSuite))
}

func (s *ConfigTestSuite) TestNamespace() {
	cfg := DefaultConfig()
	s.Equal("health", cfg.Namespace())
}

func (s *ConfigTestSuite) TestFlags_RegistersExpectedFlags() {
	cfg := DefaultConfig()
	fs := pflag.NewFlagSet("test", pflag.ContinueOnError)
	cfg.Flags(fs)

	expected := []string{
		"health-port",
		"health-liveness-path",
		"health-readiness-path",
		"health-startup-path",
	}

	for _, name := range expected {
		flag := fs.Lookup(name)
		s.NotNilf(flag, "Flag %q should be registered", name)
	}
}

func (s *ConfigTestSuite) TestSetDefaults_ZeroValues() {
	cfg := Config{}
	cfg.SetDefaults()

	s.Equal(DefaultPort, cfg.Port)
	s.Equal(DefaultLivenessPath, cfg.LivenessPath)
	s.Equal(DefaultReadinessPath, cfg.ReadinessPath)
	s.Equal(DefaultStartupPath, cfg.StartupPath)
}

func (s *ConfigTestSuite) TestSetDefaults_PreservesExistingValues() {
	cfg := Config{
		Port:          8080,
		LivenessPath:  "/custom-live",
		ReadinessPath: "/custom-ready",
		StartupPath:   "/custom-startup",
	}
	cfg.SetDefaults()

	s.Equal(8080, cfg.Port)
	s.Equal("/custom-live", cfg.LivenessPath)
	s.Equal("/custom-ready", cfg.ReadinessPath)
	s.Equal("/custom-startup", cfg.StartupPath)
}

func (s *ConfigTestSuite) TestValidate_ValidConfig() {
	cfg := DefaultConfig()
	err := cfg.Validate()
	s.NoError(err)
}

func (s *ConfigTestSuite) TestValidate_InvalidPortZero() {
	cfg := DefaultConfig()
	cfg.Port = 0
	err := cfg.Validate()
	s.Require().Error(err)
	s.Contains(err.Error(), "port")
}

func (s *ConfigTestSuite) TestValidate_InvalidPortNegative() {
	cfg := DefaultConfig()
	cfg.Port = -1
	err := cfg.Validate()
	s.Require().Error(err)
	s.Contains(err.Error(), "port")
}

func (s *ConfigTestSuite) TestValidate_InvalidPortTooHigh() {
	cfg := DefaultConfig()
	cfg.Port = 70000
	err := cfg.Validate()
	s.Require().Error(err)
	s.Contains(err.Error(), "port")
}
