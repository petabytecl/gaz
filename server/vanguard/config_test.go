package vanguard

import (
	"testing"
	"time"

	"github.com/spf13/pflag"
	"github.com/stretchr/testify/suite"
)

// ConfigTestSuite tests the Vanguard server configuration.
type ConfigTestSuite struct {
	suite.Suite
}

func TestConfigTestSuite(t *testing.T) {
	suite.Run(t, new(ConfigTestSuite))
}

func (s *ConfigTestSuite) TestDefaultConfig() {
	cfg := DefaultConfig()

	s.Equal(DefaultPort, cfg.Port)
	s.Equal(time.Duration(0), cfg.ReadTimeout, "ReadTimeout should be zero for streaming safety")
	s.Equal(time.Duration(0), cfg.WriteTimeout, "WriteTimeout should be zero for streaming safety")
	s.Equal(DefaultReadHeaderTimeout, cfg.ReadHeaderTimeout)
	s.Equal(DefaultIdleTimeout, cfg.IdleTimeout)
	s.True(cfg.Reflection)
	s.True(cfg.HealthEnabled)
	s.False(cfg.DevMode)
}

func (s *ConfigTestSuite) TestNamespace() {
	cfg := DefaultConfig()
	s.Equal("server", cfg.Namespace())
}

func (s *ConfigTestSuite) TestValidateAcceptsStreamingSafeZeroTimeouts() {
	cfg := DefaultConfig()
	cfg.ReadTimeout = 0
	cfg.WriteTimeout = 0

	err := cfg.Validate()
	s.Require().NoError(err, "Validate must accept zero ReadTimeout and WriteTimeout for streaming")
}

func (s *ConfigTestSuite) TestValidateAcceptsExplicitTimeouts() {
	cfg := DefaultConfig()
	cfg.ReadTimeout = 30 * time.Second
	cfg.WriteTimeout = 30 * time.Second

	err := cfg.Validate()
	s.Require().NoError(err)
}

func (s *ConfigTestSuite) TestValidateRejectsInvalidPort() {
	tests := []struct {
		name string
		port int
	}{
		{"zero port", 0},
		{"negative port", -1},
		{"port too high", 65536},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			cfg := DefaultConfig()
			cfg.Port = tt.port
			err := cfg.Validate()
			s.Require().Error(err)
			s.Contains(err.Error(), "port")
		})
	}
}

func (s *ConfigTestSuite) TestValidateRejectsZeroReadHeaderTimeout() {
	cfg := DefaultConfig()
	cfg.ReadHeaderTimeout = 0

	err := cfg.Validate()
	s.Require().Error(err)
	s.Contains(err.Error(), "read_header_timeout")
}

func (s *ConfigTestSuite) TestValidateRejectsNegativeReadHeaderTimeout() {
	cfg := DefaultConfig()
	cfg.ReadHeaderTimeout = -1 * time.Second

	err := cfg.Validate()
	s.Require().Error(err)
	s.Contains(err.Error(), "read_header_timeout")
}

func (s *ConfigTestSuite) TestValidateRejectsZeroIdleTimeout() {
	cfg := DefaultConfig()
	cfg.IdleTimeout = 0

	err := cfg.Validate()
	s.Require().Error(err)
	s.Contains(err.Error(), "idle_timeout")
}

func (s *ConfigTestSuite) TestValidateRejectsNegativeIdleTimeout() {
	cfg := DefaultConfig()
	cfg.IdleTimeout = -1 * time.Second

	err := cfg.Validate()
	s.Require().Error(err)
	s.Contains(err.Error(), "idle_timeout")
}

func (s *ConfigTestSuite) TestSetDefaultsFillsZeroPort() {
	cfg := Config{}
	cfg.SetDefaults()

	s.Equal(DefaultPort, cfg.Port)
}

func (s *ConfigTestSuite) TestSetDefaultsFillsZeroReadHeaderTimeout() {
	cfg := Config{}
	cfg.SetDefaults()

	s.Equal(DefaultReadHeaderTimeout, cfg.ReadHeaderTimeout)
}

func (s *ConfigTestSuite) TestSetDefaultsFillsZeroIdleTimeout() {
	cfg := Config{}
	cfg.SetDefaults()

	s.Equal(DefaultIdleTimeout, cfg.IdleTimeout)
}

func (s *ConfigTestSuite) TestSetDefaultsDoesNotOverrideReadTimeout() {
	cfg := Config{ReadTimeout: 0}
	cfg.SetDefaults()

	s.Equal(time.Duration(0), cfg.ReadTimeout, "SetDefaults must NOT override zero ReadTimeout")
}

func (s *ConfigTestSuite) TestSetDefaultsDoesNotOverrideWriteTimeout() {
	cfg := Config{WriteTimeout: 0}
	cfg.SetDefaults()

	s.Equal(time.Duration(0), cfg.WriteTimeout, "SetDefaults must NOT override zero WriteTimeout")
}

func (s *ConfigTestSuite) TestSetDefaultsPreservesExistingValues() {
	cfg := Config{
		Port:              9090,
		ReadHeaderTimeout: 10 * time.Second,
		IdleTimeout:       60 * time.Second,
	}
	cfg.SetDefaults()

	s.Equal(9090, cfg.Port)
	s.Equal(10*time.Second, cfg.ReadHeaderTimeout)
	s.Equal(60*time.Second, cfg.IdleTimeout)
}

func (s *ConfigTestSuite) TestFlagsRegistersExpectedFlags() {
	cfg := DefaultConfig()
	fs := pflag.NewFlagSet("test", pflag.ContinueOnError)
	cfg.Flags(fs)

	expected := []string{
		"server-port",
		"server-read-header-timeout",
		"server-idle-timeout",
		"server-reflection",
		"server-health-enabled",
		"server-dev-mode",
	}

	for _, name := range expected {
		flag := fs.Lookup(name)
		s.NotNilf(flag, "Flag %q should be registered", name)
	}

	// ReadTimeout and WriteTimeout should NOT have flags.
	s.Nil(fs.Lookup("server-read-timeout"), "server-read-timeout should not be a flag")
	s.Nil(fs.Lookup("server-write-timeout"), "server-write-timeout should not be a flag")
}
