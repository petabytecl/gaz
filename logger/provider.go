package logger

import (
	"log/slog"
	"os"

	"github.com/lmittmann/tint"
)

// NewLogger creates a new slog.Logger based on the configuration.
// It sets the default logger to the returned logger.
func NewLogger(cfg *Config) *slog.Logger {
	// Create LevelVar for dynamic level changing
	lvl := new(slog.LevelVar)
	lvl.Set(cfg.Level)

	var handler slog.Handler

	// Default to JSON if not text
	if cfg.Format == "text" {
		// Use tint for text output (nice colors for dev)
		handler = tint.NewHandler(os.Stdout, &tint.Options{
			Level:      lvl,
			AddSource:  cfg.AddSource,
			TimeFormat: "15:04:05.000",
		})
	} else {
		// Default to JSON
		handler = slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
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
