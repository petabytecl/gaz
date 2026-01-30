package cron

import (
	"bytes"
	"errors"
	"log/slog"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewSlogAdapter(t *testing.T) {
	var buf bytes.Buffer
	logger := slog.New(slog.NewTextHandler(&buf, nil))

	adapter := NewSlogAdapter(logger)

	require.NotNil(t, adapter)

	// Verify adapter implements cron.Logger by calling Info
	adapter.Info("test message", "key", "value")

	output := buf.String()
	assert.Contains(t, output, "test message")
	assert.Contains(t, output, "component=cron")
	assert.Contains(t, output, "key=value")
}

func TestSlogAdapter_Info(t *testing.T) {
	var buf bytes.Buffer
	logger := slog.New(slog.NewTextHandler(&buf, nil))
	adapter := NewSlogAdapter(logger)

	adapter.Info("job scheduled", "job", "test-job", "schedule", "@hourly")

	output := buf.String()
	assert.Contains(t, output, "level=INFO")
	assert.Contains(t, output, "job scheduled")
	assert.Contains(t, output, "component=cron")
	assert.Contains(t, output, "job=test-job")
	assert.Contains(t, output, "schedule=@hourly")
}

func TestSlogAdapter_Info_EmptyKeyValues(t *testing.T) {
	var buf bytes.Buffer
	logger := slog.New(slog.NewTextHandler(&buf, nil))
	adapter := NewSlogAdapter(logger)

	adapter.Info("simple message")

	output := buf.String()
	assert.Contains(t, output, "level=INFO")
	assert.Contains(t, output, "simple message")
	assert.Contains(t, output, "component=cron")
}

func TestSlogAdapter_Error(t *testing.T) {
	var buf bytes.Buffer
	logger := slog.New(slog.NewTextHandler(&buf, nil))
	adapter := NewSlogAdapter(logger)

	testErr := errors.New("connection failed")
	adapter.Error(testErr, "job failed", "job", "test-job", "attempt", 3)

	output := buf.String()
	assert.Contains(t, output, "level=ERROR")
	assert.Contains(t, output, "job failed")
	assert.Contains(t, output, "component=cron")
	assert.Contains(t, output, "job=test-job")
	assert.Contains(t, output, "attempt=3")
	assert.Contains(t, output, "connection failed")
}

func TestSlogAdapter_Error_NilError(t *testing.T) {
	var buf bytes.Buffer
	logger := slog.New(slog.NewTextHandler(&buf, nil))
	adapter := NewSlogAdapter(logger)

	adapter.Error(nil, "error with nil", "key", "value")

	output := buf.String()
	assert.Contains(t, output, "level=ERROR")
	assert.Contains(t, output, "error with nil")
}

func TestKeysAndValuesToSlog_ValidPairs(t *testing.T) {
	kvs := []any{"key1", "value1", "key2", 42, "key3", true}

	attrs := keysAndValuesToSlog(kvs)

	assert.Len(t, attrs, 3)
}

func TestKeysAndValuesToSlog_NonStringKey(t *testing.T) {
	// Non-string keys should be skipped
	kvs := []any{123, "value1", "validKey", "value2"}

	attrs := keysAndValuesToSlog(kvs)

	// Only validKey should be included (123 is not a string key)
	assert.Len(t, attrs, 1)
}

func TestKeysAndValuesToSlog_OddLength(t *testing.T) {
	// Odd number of elements - last one should be ignored
	kvs := []any{"key1", "value1", "orphan"}

	attrs := keysAndValuesToSlog(kvs)

	// Only key1 should be included
	assert.Len(t, attrs, 1)
}

func TestKeysAndValuesToSlog_Empty(t *testing.T) {
	attrs := keysAndValuesToSlog(nil)
	assert.Nil(t, attrs)

	attrs = keysAndValuesToSlog([]any{})
	assert.Nil(t, attrs)
}

func TestSlogAdapter_MultipleMessages(t *testing.T) {
	var buf bytes.Buffer
	logger := slog.New(slog.NewTextHandler(&buf, nil))
	adapter := NewSlogAdapter(logger)

	adapter.Info("first message")
	adapter.Info("second message")
	adapter.Error(errors.New("test"), "third message")

	output := buf.String()
	lines := strings.Split(strings.TrimSpace(output), "\n")
	assert.Len(t, lines, 3)
}
