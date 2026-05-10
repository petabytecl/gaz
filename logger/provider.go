package logger

import (
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"syscall"

	"github.com/petabytecl/gaz/logger/tint"
)

// nopCloser is an io.Closer that does nothing.
// Used for stdout/stderr where closing is not desired.
type nopCloser struct{}

func (nopCloser) Close() error { return nil }

// NewLogger creates a new slog.Logger based on the configuration.
// It sets the default logger to the returned logger.
// Output is resolved from cfg.Output: "stdout", "stderr", or a file path.
func NewLogger(cfg *Config) *slog.Logger {
	w := resolveOutput(cfg)
	return NewLoggerWithWriter(cfg, w)
}

// NewLoggerWithCloser creates a new slog.Logger and returns an io.Closer
// that closes the underlying output handle. For stdout/stderr, the closer
// is a no-op. For file-based output, the closer closes the file.
// The caller is responsible for calling Close() when the logger is no longer needed.
func NewLoggerWithCloser(cfg *Config) (*slog.Logger, io.Closer) {
	w, closer := resolveOutputWithCloser(cfg)
	return NewLoggerWithWriter(cfg, w), closer
}

// NewLoggerWithWriter creates a new slog.Logger writing to the given writer.
// This is useful for testing or custom output destinations.
//
// Unlike previous versions, this function no longer calls slog.SetDefault.
// Use SetGlobal explicitly when the logger should become the process-wide default.
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

	return slog.New(handler)
}

// SetGlobal sets the given logger as the global default via slog.SetDefault.
// Called once during App.Build(), not on every logger creation.
func SetGlobal(l *slog.Logger) {
	slog.SetDefault(l)
}

// resolveOutputWithCloser resolves the output destination and returns both the writer
// and a closer. For file outputs, the closer closes the file handle. For stdout/stderr,
// the closer is a no-op.
func resolveOutputWithCloser(cfg *Config) (io.Writer, io.Closer) {
	switch cfg.Output {
	case "", "stdout":
		return os.Stdout, nopCloser{}
	case "stderr":
		return os.Stderr, nopCloser{}
	default:
		cleanPath := filepath.Clean(cfg.Output)
		if strings.Contains(cleanPath, "..") {
			_, _ = fmt.Fprintf(os.Stderr, "logger: path must not contain '..': %s\n", cfg.Output)
			return os.Stdout, nopCloser{}
		}
		//nolint:gosec // Log files readable by owner+group only (0o640)
		f, err := os.OpenFile(
			cleanPath,
			os.O_APPEND|os.O_CREATE|os.O_WRONLY|syscall.O_NOFOLLOW,
			0o640,
		)
		if err != nil {
			_, _ = fmt.Fprintf(os.Stderr, "logger: failed to open %s: %v, falling back to stdout\n",
				cfg.Output, err)
			return os.Stdout, nopCloser{}
		}
		return f, f
	}
}

// resolveOutput resolves the output destination from the config.
// Returns os.Stdout for "stdout" or empty, os.Stderr for "stderr",
// or opens a file for any other path. Falls back to stdout on file errors.
func resolveOutput(cfg *Config) io.Writer {
	w, _ := resolveOutputWithCloser(cfg)
	return w
}
