package tintx

import "log/slog"

// ANSI color escape sequences for log levels.
const (
	ansiBrightRed    = "\x1b[91m" // ERROR
	ansiBrightYellow = "\x1b[93m" // WARN
	ansiBrightGreen  = "\x1b[92m" // INFO
	ansiBrightBlue   = "\x1b[94m" // DEBUG
	ansiReset        = "\x1b[0m"
	ansiFaint        = "\x1b[2m"
)

// Options configure the Handler behavior.
type Options struct {
	// Level is the minimum level to log. Uses slog.Leveler interface.
	// Default: slog.LevelInfo
	Level slog.Leveler

	// AddSource includes file:line in output when true.
	AddSource bool

	// TimeFormat is the time.Layout format string for timestamps.
	// Default: "15:04:05.000" (matches current logger usage)
	TimeFormat string

	// NoColor disables ANSI color output.
	// Auto-detected based on TTY when not explicitly set.
	NoColor bool
}
