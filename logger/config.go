package logger

import "log/slog"

// Config holds configuration for the logger.
type Config struct {
	// Level is the minimum logging level.
	// Defaults to slog.LevelInfo.
	Level slog.Level

	// Format specifies the output format.
	// Values: "json" (default), "text".
	Format string

	// AddSource includes the source file and line number in the log.
	AddSource bool
}
