package cron

import (
	"log/slog"

	"github.com/robfig/cron/v3"
)

// slogAdapter adapts slog.Logger to implement cron.Logger interface.
//
// This enables robfig/cron to log using gaz's slog infrastructure,
// providing consistent structured logging across the application.
type slogAdapter struct {
	logger *slog.Logger
}

// NewSlogAdapter creates a cron.Logger that logs to the given slog.Logger.
//
// The adapter adds a "component" attribute with value "cron" to all log
// entries for easy filtering and correlation.
func NewSlogAdapter(logger *slog.Logger) cron.Logger {
	return &slogAdapter{logger: logger.With("component", "cron")}
}

// Info logs an informational message with key-value pairs.
//
// This is called by robfig/cron for routine operations like job scheduling.
func (a *slogAdapter) Info(msg string, keysAndValues ...any) {
	a.logger.Info(msg, keysAndValuesToSlog(keysAndValues)...)
}

// Error logs an error message with the error and key-value pairs.
//
// This is called by robfig/cron when operations fail.
func (a *slogAdapter) Error(err error, msg string, keysAndValues ...any) {
	attrs := keysAndValuesToSlog(keysAndValues)
	attrs = append(attrs, slog.Any("error", err))
	a.logger.Error(msg, attrs...)
}

// keysAndValuesToSlog converts key-value pairs to slog attributes.
//
// Keys must be strings; non-string keys are skipped. Values are wrapped
// with slog.Any to preserve their type.
func keysAndValuesToSlog(kvs []any) []any {
	var attrs []any
	for i := 0; i < len(kvs)-1; i += 2 {
		key, ok := kvs[i].(string)
		if !ok {
			continue
		}
		attrs = append(attrs, slog.Any(key, kvs[i+1]))
	}
	return attrs
}
