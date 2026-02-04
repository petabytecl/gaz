package module_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/spf13/cobra"
	"github.com/stretchr/testify/suite"

	"github.com/petabytecl/gaz"
	configmod "github.com/petabytecl/gaz/config/module"
)

type ConfigModuleSuite struct {
	suite.Suite
	tempDir string
}

func TestConfigModuleSuite(t *testing.T) {
	suite.Run(t, new(ConfigModuleSuite))
}

func (s *ConfigModuleSuite) SetupTest() {
	var err error
	s.tempDir, err = os.MkdirTemp("", "config-module-test-*")
	s.Require().NoError(err)
}

func (s *ConfigModuleSuite) TearDownTest() {
	_ = os.RemoveAll(s.tempDir)
}

func (s *ConfigModuleSuite) TestDefaultConfig() {
	cfg := configmod.DefaultConfig()

	s.Equal("", cfg.ConfigFile, "ConfigFile should be empty by default")
	s.Equal("GAZ", cfg.EnvPrefix, "EnvPrefix should default to GAZ")
	s.True(cfg.Strict, "Strict should default to true")
}

func (s *ConfigModuleSuite) TestConfigNamespace() {
	cfg := configmod.DefaultConfig()
	s.Equal("config", cfg.Namespace())
}

func (s *ConfigModuleSuite) TestConfigSetDefaults() {
	cfg := configmod.Config{} // Zero value
	cfg.SetDefaults()

	s.Equal("GAZ", cfg.EnvPrefix, "EnvPrefix should be set to GAZ")
}

func (s *ConfigModuleSuite) TestConfigValidate_EmptyPath() {
	cfg := configmod.DefaultConfig()
	err := cfg.Validate()
	s.NoError(err, "Empty config file path should not error")
}

func (s *ConfigModuleSuite) TestConfigValidate_ValidPath() {
	// Create temp config file
	configPath := filepath.Join(s.tempDir, "config.yaml")
	err := os.WriteFile(configPath, []byte("key: value"), 0o644)
	s.Require().NoError(err)

	cfg := configmod.Config{ConfigFile: configPath}
	err = cfg.Validate()
	s.NoError(err, "Valid config file path should not error")
}

func (s *ConfigModuleSuite) TestConfigValidate_InvalidPath() {
	cfg := configmod.Config{ConfigFile: "/nonexistent/config.yaml"}
	err := cfg.Validate()
	s.Error(err, "Nonexistent config file should error")
	s.Contains(err.Error(), "not found")
}

func (s *ConfigModuleSuite) TestGetSearchPaths_NoXDG() {
	// Clear XDG env var
	original := os.Getenv("XDG_CONFIG_HOME")
	s.Require().NoError(os.Unsetenv("XDG_CONFIG_HOME"))
	defer func() {
		if original != "" {
			_ = os.Setenv("XDG_CONFIG_HOME", original)
		}
	}()

	cfg := configmod.DefaultConfig()
	paths := cfg.GetSearchPaths("testapp")

	s.Contains(paths, ".", "Should include current directory")
	// Should include ~/.config/testapp if home is available
	if home, err := os.UserHomeDir(); err == nil {
		expectedPath := filepath.Join(home, ".config", "testapp")
		s.Contains(paths, expectedPath, "Should include XDG default path")
	}
}

func (s *ConfigModuleSuite) TestGetSearchPaths_WithXDG() {
	xdgDir := filepath.Join(s.tempDir, "xdg-config")
	s.Require().NoError(os.MkdirAll(xdgDir, 0o755))

	original := os.Getenv("XDG_CONFIG_HOME")
	s.Require().NoError(os.Setenv("XDG_CONFIG_HOME", xdgDir))
	defer func() {
		if original != "" {
			_ = os.Setenv("XDG_CONFIG_HOME", original)
		} else {
			_ = os.Unsetenv("XDG_CONFIG_HOME")
		}
	}()

	cfg := configmod.DefaultConfig()
	paths := cfg.GetSearchPaths("testapp")

	s.Contains(paths, ".", "Should include current directory")
	s.Contains(paths, filepath.Join(xdgDir, "testapp"), "Should include custom XDG path")
}

func (s *ConfigModuleSuite) TestGetSearchPaths_EmptyAppName() {
	cfg := configmod.DefaultConfig()
	paths := cfg.GetSearchPaths("")

	s.Contains(paths, ".", "Should include current directory")
	// Should NOT include XDG path with empty app name
	s.Len(paths, 1, "Should only have current directory with empty app name")
}

func (s *ConfigModuleSuite) TestFlagsRegistered() {
	cmd := &cobra.Command{Use: "test"}
	app := gaz.New(gaz.WithCobra(cmd))
	app.Use(configmod.New())

	// Flags should be registered on persistent flags
	s.NotNil(cmd.PersistentFlags().Lookup("config"), "--config flag should be registered")
	s.NotNil(cmd.PersistentFlags().Lookup("env-prefix"), "--env-prefix flag should be registered")
	s.NotNil(cmd.PersistentFlags().Lookup("config-strict"), "--config-strict flag should be registered")
}

func (s *ConfigModuleSuite) TestFlagsDefaultValues() {
	cmd := &cobra.Command{Use: "test"}
	app := gaz.New(gaz.WithCobra(cmd))
	app.Use(configmod.New())

	// Check default values
	configFlag := cmd.PersistentFlags().Lookup("config")
	s.Equal("", configFlag.DefValue, "--config default should be empty")

	envPrefixFlag := cmd.PersistentFlags().Lookup("env-prefix")
	s.Equal("GAZ", envPrefixFlag.DefValue, "--env-prefix default should be GAZ")

	strictFlag := cmd.PersistentFlags().Lookup("config-strict")
	s.Equal("true", strictFlag.DefValue, "--config-strict default should be true")
}

func (s *ConfigModuleSuite) TestModuleProvideConfig() {
	cmd := &cobra.Command{
		Use: "test",
		RunE: func(cmd *cobra.Command, args []string) error {
			return nil
		},
	}

	app := gaz.New(gaz.WithCobra(cmd))
	app.Use(configmod.New())

	// Build should succeed
	err := app.Build()
	s.NoError(err)

	// Config should be resolvable
	cfg, err := gaz.Resolve[configmod.Config](app.Container())
	s.NoError(err)
	s.Equal("GAZ", cfg.EnvPrefix)
	s.True(cfg.Strict)
}

func (s *ConfigModuleSuite) TestExplicitConfigFileExists() {
	// Create temp config file
	configPath := filepath.Join(s.tempDir, "app.yaml")
	s.Require().NoError(os.WriteFile(configPath, []byte("key: value"), 0o644))

	executed := false
	cmd := &cobra.Command{
		Use: "test",
		RunE: func(cmd *cobra.Command, args []string) error {
			executed = true
			return nil
		},
	}

	app := gaz.New(gaz.WithCobra(cmd))
	app.Use(configmod.New())

	// Set args before Execute
	cmd.SetArgs([]string{"--config", configPath})
	err := cmd.Execute()
	s.NoError(err)
	s.True(executed, "Command should have executed")
}

func (s *ConfigModuleSuite) TestExplicitConfigFileNotExists() {
	executed := false
	cmd := &cobra.Command{
		Use: "test",
		RunE: func(cmd *cobra.Command, args []string) error {
			executed = true
			return nil
		},
	}

	app := gaz.New(gaz.WithCobra(cmd))
	app.Use(configmod.New())

	cmd.SetArgs([]string{"--config", "/nonexistent/config.yaml"})
	err := cmd.Execute()
	s.Error(err, "Should error for nonexistent config file")
	s.Contains(err.Error(), "not found")
	s.False(executed, "Command should not have executed")
}

func (s *ConfigModuleSuite) TestEnvPrefixFlag() {
	cmd := &cobra.Command{
		Use: "test",
		RunE: func(cmd *cobra.Command, args []string) error {
			return nil
		},
	}

	app := gaz.New(gaz.WithCobra(cmd))
	app.Use(configmod.New())

	cmd.SetArgs([]string{"--env-prefix", "MYAPP"})
	err := cmd.Execute()
	s.NoError(err)

	// The env prefix is applied to the config manager, which we can verify
	// by checking the flag value was parsed correctly
	envPrefixFlag := cmd.PersistentFlags().Lookup("env-prefix")
	s.Equal("MYAPP", envPrefixFlag.Value.String())
}

func (s *ConfigModuleSuite) TestConfigStrictFlagFalse() {
	cmd := &cobra.Command{
		Use: "test",
		RunE: func(cmd *cobra.Command, args []string) error {
			return nil
		},
	}

	app := gaz.New(gaz.WithCobra(cmd))
	app.Use(configmod.New())

	cmd.SetArgs([]string{"--config-strict=false"})
	err := cmd.Execute()
	s.NoError(err)

	strictFlag := cmd.PersistentFlags().Lookup("config-strict")
	s.Equal("false", strictFlag.Value.String())
}

func (s *ConfigModuleSuite) TestAutoSearchWithoutConfigFlag() {
	// Create config file in temp directory
	configPath := filepath.Join(s.tempDir, "config.yaml")
	s.Require().NoError(os.WriteFile(configPath, []byte("test: value"), 0o644))

	// Change to temp directory for auto-search
	oldWd, err := os.Getwd()
	s.Require().NoError(err)
	s.Require().NoError(os.Chdir(s.tempDir))
	defer func() { _ = os.Chdir(oldWd) }()

	executed := false
	cmd := &cobra.Command{
		Use: "test",
		RunE: func(cmd *cobra.Command, args []string) error {
			executed = true
			return nil
		},
	}

	app := gaz.New(gaz.WithCobra(cmd))
	app.Use(configmod.New())

	// No --config flag, should use auto-search
	cmd.SetArgs([]string{})
	err = cmd.Execute()
	s.NoError(err)
	s.True(executed, "Command should have executed with auto-search")
}
