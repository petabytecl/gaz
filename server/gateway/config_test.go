package gateway

import (
	"testing"

	"github.com/stretchr/testify/suite"
)

// ConfigTestSuite tests gateway configuration.
type ConfigTestSuite struct {
	suite.Suite
}

func TestConfigTestSuite(t *testing.T) {
	suite.Run(t, new(ConfigTestSuite))
}

func (s *ConfigTestSuite) TestDefaultConfig() {
	cfg := DefaultConfig()

	s.Require().Equal(DefaultPort, cfg.Port)
	s.Require().Equal(DefaultGRPCTarget, cfg.GRPCTarget)
	s.Require().NotNil(cfg.CORS)
	// Default is prod mode (devMode=false).
	s.Require().Empty(cfg.CORS.AllowedOrigins, "Prod mode should have empty allowed origins")
	s.Require().True(cfg.CORS.AllowCredentials, "Prod mode should allow credentials")
}

func (s *ConfigTestSuite) TestDefaultCORSConfig_DevMode() {
	cfg := DefaultCORSConfig(true)

	s.Require().Equal([]string{"*"}, cfg.AllowedOrigins, "Dev mode should allow all origins")
	s.Require().Equal([]string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"}, cfg.AllowedMethods)
	s.Require().Equal([]string{"*"}, cfg.AllowedHeaders, "Dev mode should allow all headers")
	s.Require().Empty(cfg.ExposedHeaders)
	s.Require().False(cfg.AllowCredentials, "Cannot use * with credentials")
	s.Require().Equal(DefaultCORSMaxAge, cfg.MaxAge)
}

func (s *ConfigTestSuite) TestDefaultCORSConfig_ProdMode() {
	cfg := DefaultCORSConfig(false)

	s.Require().Empty(cfg.AllowedOrigins, "Prod mode requires explicit origins")
	s.Require().Equal([]string{"GET", "POST", "PUT", "PATCH", "DELETE"}, cfg.AllowedMethods)
	s.Require().Equal([]string{"Authorization", "Content-Type", "X-Request-ID"}, cfg.AllowedHeaders)
	s.Require().Equal([]string{"X-Request-ID"}, cfg.ExposedHeaders)
	s.Require().True(cfg.AllowCredentials, "Prod mode enables credentials")
	s.Require().Equal(DefaultCORSMaxAge, cfg.MaxAge)
}

func (s *ConfigTestSuite) TestConfig_SetDefaults() {
	cfg := Config{}
	cfg.SetDefaults()

	s.Require().Equal(DefaultPort, cfg.Port, "Port should default to DefaultPort")
	s.Require().Equal(DefaultGRPCTarget, cfg.GRPCTarget, "GRPCTarget should default to DefaultGRPCTarget")
}

func (s *ConfigTestSuite) TestConfig_SetDefaults_PreservesExisting() {
	cfg := Config{
		Port:       9000,
		GRPCTarget: "custom:8080",
	}
	cfg.SetDefaults()

	s.Require().Equal(9000, cfg.Port, "Existing port should be preserved")
	s.Require().Equal("custom:8080", cfg.GRPCTarget, "Existing target should be preserved")
}

func (s *ConfigTestSuite) TestConfig_Validate_Valid() {
	cfg := DefaultConfig()
	err := cfg.Validate()

	s.Require().NoError(err)
}

func (s *ConfigTestSuite) TestConfig_Validate_InvalidPort_Zero() {
	cfg := DefaultConfig()
	cfg.Port = 0
	err := cfg.Validate()

	s.Require().Error(err)
	s.Require().Contains(err.Error(), "port")
}

func (s *ConfigTestSuite) TestConfig_Validate_InvalidPort_Negative() {
	cfg := DefaultConfig()
	cfg.Port = -1
	err := cfg.Validate()

	s.Require().Error(err)
	s.Require().Contains(err.Error(), "port")
}

func (s *ConfigTestSuite) TestConfig_Validate_InvalidPort_TooHigh() {
	cfg := DefaultConfig()
	cfg.Port = 70000
	err := cfg.Validate()

	s.Require().Error(err)
	s.Require().Contains(err.Error(), "port")
}

func (s *ConfigTestSuite) TestConfig_Validate_ValidPort_Boundary() {
	// Test boundary values.
	testCases := []struct {
		name  string
		port  int
		valid bool
	}{
		{"port 1 is valid", 1, true},
		{"port 65535 is valid", 65535, true},
		{"port 65536 is invalid", 65536, false},
	}

	for _, tc := range testCases {
		s.Run(tc.name, func() {
			cfg := DefaultConfig()
			cfg.Port = tc.port
			err := cfg.Validate()
			if tc.valid {
				s.Require().NoError(err)
			} else {
				s.Require().Error(err)
			}
		})
	}
}

func (s *ConfigTestSuite) TestConstants() {
	// Verify constants are defined.
	s.Require().Equal(8080, DefaultPort)
	s.Require().Equal("localhost:50051", DefaultGRPCTarget)
	s.Require().Equal(86400, DefaultCORSMaxAge)
}
