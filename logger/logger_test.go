package logger

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// captureOutput captures output of a function that writes to os.Stdout.
func captureOutput(f func()) string {
	r, w, _ := os.Pipe()
	stdout := os.Stdout
	os.Stdout = w //nolint:reassign // mocking stdout for testing
	defer func() {
		os.Stdout = stdout //nolint:reassign // restoring stdout
	}()

	f()
	_ = w.Close()

	var buf bytes.Buffer
	_, _ = io.Copy(&buf, r)
	return buf.String()
}

func TestNewLogger_JSON(t *testing.T) {
	output := captureOutput(func() {
		cfg := &Config{
			Level:  slog.LevelInfo,
			Format: "json",
		}
		logger := NewLogger(cfg)
		logger.Info("test message", "key", "value")
	})

	require.NotEmpty(t, output)

	var logMap map[string]any
	err := json.Unmarshal([]byte(output), &logMap)
	require.NoError(t, err, "Output should be valid JSON")

	assert.Equal(t, "test message", logMap["msg"])
	assert.Equal(t, "value", logMap["key"])
	assert.Equal(t, "INFO", logMap["level"])
}

func TestNewLogger_Text(t *testing.T) {
	output := captureOutput(func() {
		cfg := &Config{
			Level:  slog.LevelInfo,
			Format: "text",
		}
		logger := NewLogger(cfg)
		logger.Info("test message", "key", "value")
	})

	require.NotEmpty(t, output)
	assert.Contains(t, output, "test message")
	// Tint adds colors, so exact match is hard. Check for key and value.
	assert.Contains(t, output, "key=")
	assert.Contains(t, output, "value")
}

func TestNewLoggerWithCloser_FileOutput(t *testing.T) {
	tmpFile := t.TempDir() + "/test.log"

	cfg := &Config{
		Level:  slog.LevelInfo,
		Format: "json",
		Output: tmpFile,
	}

	logger, closer := NewLoggerWithCloser(cfg)
	require.NotNil(t, logger)
	require.NotNil(t, closer)

	// Write something
	logger.Info("hello file")

	// Close should succeed
	err := closer.Close()
	require.NoError(t, err)

	// Verify file has content
	data, err := os.ReadFile(tmpFile)
	require.NoError(t, err)
	assert.Contains(t, string(data), "hello file")
}

func TestNewLoggerWithCloser_StdoutOutput(t *testing.T) {
	cfg := &Config{
		Level:  slog.LevelInfo,
		Format: "json",
		Output: "stdout",
	}

	logger, closer := NewLoggerWithCloser(cfg)
	require.NotNil(t, logger)
	require.NotNil(t, closer)

	// Closer should be a no-op (not nil)
	err := closer.Close()
	require.NoError(t, err)
}

func TestNewLoggerWithCloser_StderrOutput(t *testing.T) {
	cfg := &Config{
		Level:  slog.LevelInfo,
		Format: "json",
		Output: "stderr",
	}

	logger, closer := NewLoggerWithCloser(cfg)
	require.NotNil(t, logger)
	require.NotNil(t, closer)

	err := closer.Close()
	require.NoError(t, err)
}

func TestNewLoggerWithCloser_CloserClosesFile(t *testing.T) {
	tmpFile := t.TempDir() + "/test-close.log"

	cfg := &Config{
		Level:  slog.LevelInfo,
		Format: "json",
		Output: tmpFile,
	}

	_, closer := NewLoggerWithCloser(cfg)

	// Close the file
	err := closer.Close()
	require.NoError(t, err)

	// Second close should return an error (file already closed)
	err = closer.Close()
	assert.Error(t, err, "closing an already-closed file should return an error")
}

func TestNewLogger_ContextPropagation(t *testing.T) {
	output := captureOutput(func() {
		cfg := &Config{
			Level:  slog.LevelInfo,
			Format: "json",
		}
		logger := NewLogger(cfg)

		ctx := context.Background()
		ctx = WithTraceID(ctx, "trace-123")
		ctx = WithRequestID(ctx, "req-456")

		logger.InfoContext(ctx, "test context", "foo", "bar")
	})

	require.NotEmpty(t, output)

	var logMap map[string]any
	err := json.Unmarshal([]byte(output), &logMap)
	require.NoError(t, err, "Output should be valid JSON")

	assert.Equal(t, "trace-123", logMap[TraceIDKey])
	assert.Equal(t, "req-456", logMap[RequestIDKey])
	assert.Equal(t, "test context", logMap["msg"])
}
