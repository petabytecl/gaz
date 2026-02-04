package module_test

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"github.com/stretchr/testify/suite"

	"github.com/petabytecl/gaz"
	"github.com/petabytecl/gaz/logger"
	loggermod "github.com/petabytecl/gaz/logger/module"
)

type LoggerModuleTestSuite struct {
	suite.Suite
}

func TestLoggerModuleTestSuite(t *testing.T) {
	suite.Run(t, new(LoggerModuleTestSuite))
}

func (s *LoggerModuleTestSuite) TestModuleRegistration() {
	rootCmd := &cobra.Command{Use: "test", RunE: func(_ *cobra.Command, _ []string) error { return nil }}
	app := gaz.New(gaz.WithCobra(rootCmd))
	app.Use(loggermod.New())

	err := app.Build()
	s.Require().NoError(err)

	// Config should be resolvable
	cfg, err := gaz.Resolve[logger.Config](app.Container())
	s.Require().NoError(err)

	// Check defaults
	s.Equal("text", cfg.Format)
	s.Equal("stdout", cfg.Output)
	s.Equal("info", cfg.LevelName())
}

func (s *LoggerModuleTestSuite) TestDefaultConfig() {
	cfg := logger.DefaultConfig()

	s.Equal("info", cfg.LevelName())
	s.Equal("text", cfg.Format)
	s.Equal("stdout", cfg.Output)
	s.False(cfg.AddSource)
}

func (s *LoggerModuleTestSuite) TestConfigFlags() {
	cfg := logger.DefaultConfig()
	fs := pflag.NewFlagSet("test", pflag.ContinueOnError)
	cfg.Flags(fs)

	// Verify flags registered
	s.NotNil(fs.Lookup("log-level"))
	s.NotNil(fs.Lookup("log-format"))
	s.NotNil(fs.Lookup("log-output"))
	s.NotNil(fs.Lookup("log-add-source"))

	// Parse custom values
	err := fs.Parse([]string{
		"--log-level=debug",
		"--log-format=json",
		"--log-output=stderr",
		"--log-add-source",
	})
	s.Require().NoError(err)

	// Validate to convert levelName to Level
	err = cfg.Validate()
	s.Require().NoError(err)

	s.Equal("json", cfg.Format)
	s.Equal("stderr", cfg.Output)
	s.True(cfg.AddSource)
}

func (s *LoggerModuleTestSuite) TestConfigValidation_InvalidLevel() {
	cfg := logger.DefaultConfig()
	fs := pflag.NewFlagSet("test", pflag.ContinueOnError)
	cfg.Flags(fs)
	_ = fs.Parse([]string{"--log-level=trace"})

	err := cfg.Validate()
	s.Error(err)
	s.Contains(err.Error(), "invalid log level")
	s.Contains(err.Error(), "trace")
}

func (s *LoggerModuleTestSuite) TestConfigValidation_InvalidFormat() {
	cfg := logger.DefaultConfig()
	fs := pflag.NewFlagSet("test", pflag.ContinueOnError)
	cfg.Flags(fs)
	_ = fs.Parse([]string{"--log-format=yaml"})

	err := cfg.Validate()
	s.Error(err)
	s.Contains(err.Error(), "invalid log format")
	s.Contains(err.Error(), "yaml")
}

func (s *LoggerModuleTestSuite) TestOutputStdout() {
	cfg := logger.DefaultConfig()
	cfg.Output = "stdout"
	_ = cfg.Validate()

	log := logger.NewLogger(&cfg)
	s.NotNil(log)
}

func (s *LoggerModuleTestSuite) TestOutputStderr() {
	cfg := logger.DefaultConfig()
	cfg.Output = "stderr"
	_ = cfg.Validate()

	log := logger.NewLogger(&cfg)
	s.NotNil(log)
}

func (s *LoggerModuleTestSuite) TestOutputFile() {
	dir := s.T().TempDir()
	path := filepath.Join(dir, "test.log")

	cfg := logger.DefaultConfig()
	cfg.Output = path
	_ = cfg.Validate()

	log := logger.NewLogger(&cfg)
	s.NotNil(log)

	// Write a log entry
	log.Info("test message")

	// Verify file exists and has content
	content, err := os.ReadFile(path)
	s.Require().NoError(err)
	s.Contains(string(content), "test message")
}

func (s *LoggerModuleTestSuite) TestOutputFileFallback() {
	cfg := logger.DefaultConfig()
	cfg.Output = "/nonexistent/directory/test.log"
	_ = cfg.Validate()

	// Should not panic, should fall back to stdout
	log := logger.NewLogger(&cfg)
	s.NotNil(log)
}

func (s *LoggerModuleTestSuite) TestNewLoggerWithWriter() {
	var buf bytes.Buffer
	cfg := logger.DefaultConfig()
	cfg.Format = "json"
	_ = cfg.Validate()

	log := logger.NewLoggerWithWriter(&cfg, &buf)
	log.Info("test message")

	s.Contains(buf.String(), "test message")
}

func (s *LoggerModuleTestSuite) TestAllLevels() {
	levels := []string{"debug", "info", "warn", "error"}
	for _, level := range levels {
		cfg := logger.DefaultConfig()
		fs := pflag.NewFlagSet("test", pflag.ContinueOnError)
		cfg.Flags(fs)
		_ = fs.Parse([]string{"--log-level=" + level})

		err := cfg.Validate()
		s.NoError(err, "level %s should be valid", level)
	}
}

func (s *LoggerModuleTestSuite) TestConfigNamespace() {
	cfg := logger.DefaultConfig()
	s.Equal("log", cfg.Namespace())
}

func (s *LoggerModuleTestSuite) TestConfigSetDefaults() {
	cfg := logger.Config{}
	cfg.SetDefaults()

	s.Equal("text", cfg.Format)
	s.Equal("stdout", cfg.Output)
}

func (s *LoggerModuleTestSuite) TestModuleWithFlags() {
	rootCmd := &cobra.Command{
		Use: "test",
		RunE: func(_ *cobra.Command, _ []string) error {
			return nil
		},
	}

	app := gaz.New(gaz.WithCobra(rootCmd))
	app.Use(loggermod.New())

	// Parse flags before build
	rootCmd.SetArgs([]string{"--log-level=debug", "--log-format=json"})
	err := rootCmd.Execute()
	s.Require().NoError(err)

	// Config should reflect parsed flags
	cfg, err := gaz.Resolve[logger.Config](app.Container())
	s.Require().NoError(err)

	s.Equal("json", cfg.Format)
	s.Equal("debug", cfg.LevelName())
}
