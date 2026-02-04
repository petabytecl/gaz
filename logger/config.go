// Package logger provides structured logging with context support.
package logger

import (
	"fmt"
	"log/slog"

	"github.com/spf13/pflag"
)

// Config holds configuration for the logger.
type Config struct {
	// Level is the minimum logging level.
	// Defaults to slog.LevelInfo.
	Level slog.Level

	// Format specifies the output format.
	// Values: "text" (default), "json".
	Format string

	// AddSource includes the source file and line number in the log.
	AddSource bool

	// Output specifies where logs are written.
	// Values: "stdout" (default), "stderr", or a file path.
	Output string

	// levelName is used for flag binding (internal).
	levelName string
}

// DefaultConfig returns a Config with sensible defaults.
func DefaultConfig() Config {
	return Config{
		Level:     slog.LevelInfo,
		levelName: "info",
		Format:    "text",
		Output:    "stdout",
		AddSource: false,
	}
}

// Namespace returns the configuration namespace for config binding.
func (c *Config) Namespace() string {
	return "log"
}

// Flags registers CLI flags for the logger configuration.
func (c *Config) Flags(fs *pflag.FlagSet) {
	fs.StringVar(&c.levelName, "log-level", c.levelName,
		"Log level: debug, info, warn, error")
	fs.StringVar(&c.Format, "log-format", c.Format,
		"Log format: text, json")
	fs.StringVar(&c.Output, "log-output", c.Output,
		"Log output: stdout, stderr, or file path")
	fs.BoolVar(&c.AddSource, "log-add-source", c.AddSource,
		"Include source file:line in logs")
}

// Validate validates the configuration and converts levelName to Level.
func (c *Config) Validate() error {
	// Validate and convert levelName to Level
	level, err := parseLevel(c.levelName)
	if err != nil {
		return err
	}
	c.Level = level

	// Validate format
	if c.Format != "text" && c.Format != "json" {
		return fmt.Errorf("invalid log format %q: must be text or json", c.Format)
	}

	return nil
}

// SetDefaults applies default values to zero-value fields.
func (c *Config) SetDefaults() {
	if c.Format == "" {
		c.Format = "text"
	}
	if c.Output == "" {
		c.Output = "stdout"
	}
	if c.levelName == "" {
		c.levelName = "info"
		c.Level = slog.LevelInfo
	}
}

// LevelName returns the string representation of the log level.
func (c *Config) LevelName() string {
	return c.levelName
}

// parseLevel parses a log level name and returns the corresponding slog.Level.
func parseLevel(name string) (slog.Level, error) {
	switch name {
	case "debug":
		return slog.LevelDebug, nil
	case "info":
		return slog.LevelInfo, nil
	case "warn":
		return slog.LevelWarn, nil
	case "error":
		return slog.LevelError, nil
	default:
		return slog.LevelInfo, fmt.Errorf(
			"invalid log level %q: must be debug, info, warn, or error", name)
	}
}
