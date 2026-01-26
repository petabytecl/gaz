package gaz_test

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/suite"

	"github.com/petabytecl/gaz"
)

type ConfigSuite struct {
	suite.Suite
}

func TestConfigSuite(t *testing.T) {
	suite.Run(t, new(ConfigSuite))
}

type TestConfig struct {
	Host string
	Port int
}

func (c *TestConfig) Default() {
	if c.Host == "" {
		c.Host = "localhost"
	}
	if c.Port == 0 {
		c.Port = 8080
	}
}

func (c *TestConfig) Validate() error {
	if c.Port < 0 {
		return errors.New("port must be positive")
	}
	return nil
}

func (s *ConfigSuite) TestDefaults() {
	var cfg TestConfig
	app := gaz.New().WithConfig(&cfg)

	err := app.Build()
	s.Require().NoError(err)

	s.Equal("localhost", cfg.Host)
	s.Equal(8080, cfg.Port)
}

func (s *ConfigSuite) TestEnvVars() {
	s.Require().NoError(os.Setenv("TEST_APP_HOST", "example.com"))
	s.Require().NoError(os.Setenv("TEST_APP_PORT", "9090"))
	defer func() {
		_ = os.Unsetenv("TEST_APP_HOST")
		_ = os.Unsetenv("TEST_APP_PORT")
	}()

	var cfg TestConfig
	app := gaz.New().WithConfig(&cfg, gaz.WithEnvPrefix("TEST_APP"))

	err := app.Build()
	s.Require().NoError(err)

	s.Equal("example.com", cfg.Host)
	s.Equal(9090, cfg.Port)
}

func (s *ConfigSuite) TestValidation() {
	s.Require().NoError(os.Setenv("TEST_APP_PORT", "-1"))
	defer func() { _ = os.Unsetenv("TEST_APP_PORT") }()

	var cfg TestConfig
	app := gaz.New().WithConfig(&cfg, gaz.WithEnvPrefix("TEST_APP"))

	err := app.Build()
	s.Require().Error(err)
	s.Contains(err.Error(), "port must be positive")
}

func (s *ConfigSuite) TestInjection() {
	var cfg TestConfig
	var injectedCfg *TestConfig

	app := gaz.New().
		WithConfig(&cfg).
		ProvideSingleton(func(c *gaz.Container) string {
			// This service depends on config
			conf, _ := gaz.Resolve[*TestConfig](c)
			injectedCfg = conf
			return "done"
		})

	err := app.Build()
	s.Require().NoError(err)

	// Trigger resolution
	_, err = gaz.Resolve[string](app.Container())
	s.Require().NoError(err)

	s.NotNil(injectedCfg)
	s.Same(&cfg, injectedCfg)
}

func (s *ConfigSuite) TestProfiles() {
	// Create temp dir for config files
	tmpDir := s.T().TempDir()

	// Write base config
	baseConfig := []byte("host: localhost\nport: 8080")
	err := os.WriteFile(filepath.Join(tmpDir, "config.yaml"), baseConfig, 0o600)
	s.Require().NoError(err)

	// Write profile config
	prodConfig := []byte("host: prod-host\n") // Overrides host, keeps port
	err = os.WriteFile(filepath.Join(tmpDir, "config.prod.yaml"), prodConfig, 0o600)
	s.Require().NoError(err)

	s.Require().NoError(os.Setenv("TEST_ENV", "prod"))
	defer func() { _ = os.Unsetenv("TEST_ENV") }()

	var cfg TestConfig
	app := gaz.New().WithConfig(&cfg,
		gaz.WithSearchPaths(tmpDir),
		gaz.WithProfileEnv("TEST_ENV"),
	)

	err = app.Build()
	s.Require().NoError(err)

	s.Equal("prod-host", cfg.Host)
	s.Equal(8080, cfg.Port) // Preserved from base
}
