package gaz_test

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/spf13/pflag"
	"github.com/stretchr/testify/suite"

	"github.com/petabytecl/gaz"
)

type ConfigManagerSuite struct {
	suite.Suite
}

func TestConfigManagerSuite(t *testing.T) {
	suite.Run(t, new(ConfigManagerSuite))
}

// Reusing TestConfig from config_test.go if available, or redefining here to be safe and self-contained.
type TestManagerConfig struct {
	Host string
	Port int
	DB   struct {
		User string
	}
}

func (c *TestManagerConfig) Default() {
	if c.Host == "" {
		c.Host = "localhost"
	}
	if c.Port == 0 {
		c.Port = 8080
	}
}

func (c *TestManagerConfig) Validate() error {
	if c.Port < 0 {
		return errors.New("port must be positive")
	}
	return nil
}

func (s *ConfigManagerSuite) TestLoadDefaults() {
	var cfg TestManagerConfig
	cm := gaz.NewConfigManager(&cfg)

	err := cm.Load()
	s.Require().NoError(err)

	s.Equal("localhost", cfg.Host)
	s.Equal(8080, cfg.Port)
}

func (s *ConfigManagerSuite) TestLoadEnv() {
	s.Require().NoError(os.Setenv("TEST_APP_HOST", "example.com"))
	s.Require().NoError(os.Setenv("TEST_APP_PORT", "9090"))
	s.Require().NoError(os.Setenv("TEST_APP_DB__USER", "admin"))
	defer func() {
		_ = os.Unsetenv("TEST_APP_HOST")
		_ = os.Unsetenv("TEST_APP_PORT")
		_ = os.Unsetenv("TEST_APP_DB__USER")
	}()

	var cfg TestManagerConfig
	cm := gaz.NewConfigManager(&cfg,
		gaz.WithEnvPrefix("TEST_APP"),
	)

	err := cm.Load()
	s.Require().NoError(err)

	s.Equal("example.com", cfg.Host)
	s.Equal(9090, cfg.Port)
	s.Equal("admin", cfg.DB.User)
}

func (s *ConfigManagerSuite) TestLoadFile() {
	tmpDir := s.T().TempDir()
	configContent := []byte("host: file-host\nport: 7070")
	err := os.WriteFile(filepath.Join(tmpDir, "testconfig.yaml"), configContent, 0o600)
	s.Require().NoError(err)

	var cfg TestManagerConfig
	cm := gaz.NewConfigManager(&cfg,
		gaz.WithName("testconfig"),
		gaz.WithSearchPaths(tmpDir),
	)

	err = cm.Load()
	s.Require().NoError(err)

	s.Equal("file-host", cfg.Host)
	s.Equal(7070, cfg.Port)
}

func (s *ConfigManagerSuite) TestLoadProfile() {
	tmpDir := s.T().TempDir()

	// Base config
	baseContent := []byte("host: base-host\nport: 8080")
	err := os.WriteFile(filepath.Join(tmpDir, "config.yaml"), baseContent, 0o600)
	s.Require().NoError(err)

	// Profile config
	prodContent := []byte("host: prod-host")
	err = os.WriteFile(filepath.Join(tmpDir, "config.prod.yaml"), prodContent, 0o600)
	s.Require().NoError(err)

	s.Require().NoError(os.Setenv("APP_ENV", "prod"))
	defer func() { _ = os.Unsetenv("APP_ENV") }()

	var cfg TestManagerConfig
	cm := gaz.NewConfigManager(&cfg,
		gaz.WithSearchPaths(tmpDir),
		gaz.WithProfileEnv("APP_ENV"),
	)

	err = cm.Load()
	s.Require().NoError(err)

	s.Equal("prod-host", cfg.Host)
	s.Equal(8080, cfg.Port)
}

func (s *ConfigManagerSuite) TestExplicitDefaults() {
	var cfg TestManagerConfig
	cm := gaz.NewConfigManager(&cfg,
		gaz.WithDefaults(map[string]any{
			"host": "default-host",
			"port": 9000,
		}),
	)

	err := cm.Load()
	s.Require().NoError(err)

	s.Equal("default-host", cfg.Host)
	s.Equal(9000, cfg.Port)
}

func (s *ConfigManagerSuite) TestValidation() {
	var cfg TestManagerConfig
	cm := gaz.NewConfigManager(&cfg,
		gaz.WithDefaults(map[string]any{
			"port": -1,
		}),
	)

	err := cm.Load()
	s.Require().Error(err)
	s.Contains(err.Error(), "port must be positive")
}

func (s *ConfigManagerSuite) TestBindFlags() {
	fs := pflag.NewFlagSet("test", pflag.ContinueOnError)
	var hostFlag string
	fs.StringVar(&hostFlag, "host", "flag-default", "host flag")

	// Simulate parsing flags
	err := fs.Parse([]string{"--host", "flag-host"})
	s.Require().NoError(err)

	var cfg TestManagerConfig
	cm := gaz.NewConfigManager(&cfg)

	err = cm.BindFlags(fs)
	s.Require().NoError(err)

	err = cm.Load()
	s.Require().NoError(err)

	s.Equal("flag-host", cfg.Host)
}
