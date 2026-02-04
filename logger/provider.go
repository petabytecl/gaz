package logger

import (
	"fmt"
	"io"
	"log/slog"
	"os"

	"github.com/petabytecl/gaz/logger/tint"
)

// NewLogger creates a new slog.Logger based on the configuration.
// It sets the default logger to the returned logger.
// Output is resolved from cfg.Output: "stdout", "stderr", or a file path.
func NewLogger(cfg *Config) *slog.Logger {
	w := resolveOutput(cfg)
	return NewLoggerWithWriter(cfg, w)
}

// NewLoggerWithWriter creates a new slog.Logger writing to the given writer.
// This is useful for testing or custom output destinations.
// It sets the default logger to the returned logger.
func NewLoggerWithWriter(cfg *Config, w io.Writer) *slog.Logger {
	// Create LevelVar for dynamic level changing
	lvl := new(slog.LevelVar)
	lvl.Set(cfg.Level)

	var handler slog.Handler

	// Default to JSON if not text
	if cfg.Format == "text" {
		// Use tint for text output (nice colors for dev)
		handler = tint.NewHandler(w, &tint.Options{
			Level:      lvl,
			AddSource:  cfg.AddSource,
			TimeFormat: "15:04:05.000",
		})
	} else {
		// Default to JSON
		handler = slog.NewJSONHandler(w, &slog.HandlerOptions{
			Level:     lvl,
			AddSource: cfg.AddSource,
		})
	}

	// Wrap with ContextHandler to propagate context values
	handler = NewContextHandler(handler)

	logger := slog.New(handler)
	slog.SetDefault(logger)

	return logger
}

// resolveOutput resolves the output destination from the config.
// Returns os.Stdout for "stdout" or empty, os.Stderr for "stderr",
// or opens a file for any other path. Falls back to stdout on file errors.
func resolveOutput(cfg *Config) io.Writer {
	switch cfg.Output {
	case "", "stdout":
		return os.Stdout
	case "stderr":
		return os.Stderr
	default:
		// File path - attempt to open
		//nolint:gosec // Log files need to be readable by log monitoring tools
		f, err := os.OpenFile(cfg.Output, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
		if err != nil {
			// Log warning to stderr and fall back to stdout
			fmt.Fprintf(os.Stderr, "logger: failed to open %s: %v, falling back to stdout\n",
				cfg.Output, err)
			return os.Stdout
		}
		return f
	}
}
