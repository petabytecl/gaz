package tint

import (
	"bytes"
	"context"
	"log/slog"
	"strings"
	"testing"
)

func TestNewHandler_DefaultOptions(t *testing.T) {
	var buf bytes.Buffer
	h := NewHandler(&buf, nil)

	// Non-file writers should default to NoColor
	if !h.opts.NoColor {
		t.Error("expected NoColor=true for non-file writer")
	}

	// Default time format
	if h.opts.TimeFormat != "15:04:05.000" {
		t.Errorf("expected default TimeFormat '15:04:05.000', got %q", h.opts.TimeFormat)
	}
}

func TestHandler_Enabled(t *testing.T) {
	var buf bytes.Buffer

	tests := []struct {
		name  string
		level slog.Leveler
		check slog.Level
		want  bool
	}{
		{"default enables INFO", nil, slog.LevelInfo, true},
		{"default enables WARN", nil, slog.LevelWarn, true},
		{"default disables DEBUG", nil, slog.LevelDebug, false},
		{"debug level enables DEBUG", slog.LevelDebug, slog.LevelDebug, true},
		{"warn level disables INFO", slog.LevelWarn, slog.LevelInfo, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := NewHandler(&buf, &Options{Level: tt.level})
			if got := h.Enabled(context.Background(), tt.check); got != tt.want {
				t.Errorf("Enabled(%v) = %v, want %v", tt.check, got, tt.want)
			}
		})
	}
}

func TestHandler_WithAttrs_ReturnsNewInstance(t *testing.T) {
	var buf bytes.Buffer
	h1 := NewHandler(&buf, &Options{NoColor: true})

	h2 := h1.WithAttrs([]slog.Attr{slog.String("key", "value")})

	// Must return new instance
	if h1 == h2 {
		t.Error("WithAttrs returned same instance")
	}

	// Original should not have attrs
	if h1.attrsPrefix != "" {
		t.Error("original handler modified by WithAttrs")
	}

	// New handler should have attrs
	h2Typed := h2.(*Handler)
	if h2Typed.attrsPrefix == "" {
		t.Error("new handler missing attrs")
	}
}

func TestHandler_WithGroup_ReturnsNewInstance(t *testing.T) {
	var buf bytes.Buffer
	h1 := NewHandler(&buf, &Options{NoColor: true})

	h2 := h1.WithGroup("mygroup")

	// Must return new instance
	if h1 == h2 {
		t.Error("WithGroup returned same instance")
	}

	// Original should not have group
	if h1.groupPrefix != "" {
		t.Error("original handler modified by WithGroup")
	}

	// New handler should have group prefix
	h2Typed := h2.(*Handler)
	if h2Typed.groupPrefix != "mygroup." {
		t.Errorf("expected groupPrefix 'mygroup.', got %q", h2Typed.groupPrefix)
	}
}

func TestHandler_LevelColors(t *testing.T) {
	tests := []struct {
		level slog.Level
		color string
		label string
	}{
		{slog.LevelDebug, ansiBrightBlue, "DBG"},
		{slog.LevelInfo, ansiBrightGreen, "INF"},
		{slog.LevelWarn, ansiBrightYellow, "WRN"},
		{slog.LevelError, ansiBrightRed, "ERR"},
	}

	for _, tt := range tests {
		t.Run(tt.label, func(t *testing.T) {
			var buf bytes.Buffer
			h := NewHandler(&buf, &Options{NoColor: false, Level: slog.LevelDebug})
			// Force colors on for test (normally auto-detected)
			h.opts.NoColor = false

			logger := slog.New(h)
			logger.Log(context.Background(), tt.level, "test message")

			output := buf.String()
			if !strings.Contains(output, tt.color) {
				t.Errorf("expected color %q in output, got: %s", tt.color, output)
			}
			if !strings.Contains(output, tt.label) {
				t.Errorf("expected label %q in output, got: %s", tt.label, output)
			}
		})
	}
}

func TestHandler_NoColor(t *testing.T) {
	var buf bytes.Buffer
	h := NewHandler(&buf, &Options{NoColor: true})
	logger := slog.New(h)

	logger.Info("test message")

	output := buf.String()
	if strings.Contains(output, "\x1b[") {
		t.Errorf("expected no ANSI codes with NoColor=true, got: %s", output)
	}
}

func TestHandler_GroupedAttrs(t *testing.T) {
	var buf bytes.Buffer
	h := NewHandler(&buf, &Options{NoColor: true})
	logger := slog.New(h)

	logger.WithGroup("request").With("method", "GET").Info("handled")

	output := buf.String()
	if !strings.Contains(output, "request.method=GET") {
		t.Errorf("expected grouped attr 'request.method=GET', got: %s", output)
	}
}

func TestHandler_AddSource(t *testing.T) {
	var buf bytes.Buffer
	h := NewHandler(&buf, &Options{NoColor: true, AddSource: true})
	logger := slog.New(h)

	logger.Info("test with source")

	output := buf.String()
	// Should contain file:line pattern
	if !strings.Contains(output, "handler_test.go:") {
		t.Errorf("expected source location, got: %s", output)
	}
}

func TestHandler_TimeFormat(t *testing.T) {
	var buf bytes.Buffer
	h := NewHandler(&buf, &Options{NoColor: true, TimeFormat: "2006-01-02"})
	logger := slog.New(h)

	logger.Info("test")

	output := buf.String()
	// Should contain date format (year-month-day)
	if !strings.Contains(output, "2026-") && !strings.Contains(output, "20") {
		t.Errorf("expected date in output, got: %s", output)
	}
	// Should NOT contain time component
	if strings.Contains(output, ":04:") || strings.Contains(output, ".000") {
		t.Errorf("expected custom time format without time component, got: %s", output)
	}
}

func TestHandler_ConcurrentWrites(t *testing.T) {
	var buf bytes.Buffer
	h := NewHandler(&buf, &Options{NoColor: true})
	logger := slog.New(h)

	done := make(chan bool)
	for i := range 10 {
		go func(n int) {
			for j := range 100 {
				logger.Info("concurrent", "goroutine", n, "iteration", j)
			}
			done <- true
		}(i)
	}

	for range 10 {
		<-done
	}

	// Count newlines to verify all messages written
	lines := strings.Count(buf.String(), "\n")
	if lines != 1000 {
		t.Errorf("expected 1000 log lines, got %d", lines)
	}
}

func TestHandler_AttrTypes(t *testing.T) {
	var buf bytes.Buffer
	h := NewHandler(&buf, &Options{NoColor: true})
	logger := slog.New(h)

	logger.Info("types",
		slog.String("str", "hello"),
		slog.Int("int", 42),
		slog.Bool("bool", true),
		slog.Float64("float", 3.14),
	)

	output := buf.String()
	if !strings.Contains(output, "str=hello") {
		t.Errorf("expected str=hello, got: %s", output)
	}
	if !strings.Contains(output, "int=42") {
		t.Errorf("expected int=42, got: %s", output)
	}
	if !strings.Contains(output, "bool=true") {
		t.Errorf("expected bool=true, got: %s", output)
	}
	if !strings.Contains(output, "float=3.14") {
		t.Errorf("expected float=3.14, got: %s", output)
	}
}

func TestHandler_NestedGroups(t *testing.T) {
	var buf bytes.Buffer
	h := NewHandler(&buf, &Options{NoColor: true})
	logger := slog.New(h)

	logger.WithGroup("outer").WithGroup("inner").Info("nested", "key", "value")

	output := buf.String()
	if !strings.Contains(output, "outer.inner.key=value") {
		t.Errorf("expected nested group prefix 'outer.inner.key=value', got: %s", output)
	}
}

func TestHandler_InlineGroup(t *testing.T) {
	var buf bytes.Buffer
	h := NewHandler(&buf, &Options{NoColor: true})
	logger := slog.New(h)

	logger.Info("inline group",
		slog.Group("user",
			slog.String("name", "Alice"),
			slog.Int("age", 30),
		),
	)

	output := buf.String()
	if !strings.Contains(output, "user.name=Alice") {
		t.Errorf("expected user.name=Alice, got: %s", output)
	}
	if !strings.Contains(output, "user.age=30") {
		t.Errorf("expected user.age=30, got: %s", output)
	}
}

func TestHandler_EmptyGroupIgnored(t *testing.T) {
	var buf bytes.Buffer
	h := NewHandler(&buf, &Options{NoColor: true})
	logger := slog.New(h)

	// Empty group should be ignored
	logger.Info("test message", slog.Group("emptygroup"))

	output := buf.String()
	// The empty group name should not appear as a key prefix
	if strings.Contains(output, "emptygroup.") || strings.Contains(output, "emptygroup=") {
		t.Errorf("expected empty group to be ignored, got: %s", output)
	}
}

func TestHandler_EmptyAttrIgnored(t *testing.T) {
	var buf bytes.Buffer
	h := NewHandler(&buf, &Options{NoColor: true})
	logger := slog.New(h)

	// Empty attr (zero value) should be ignored
	logger.LogAttrs(context.Background(), slog.LevelInfo, "empty attr", slog.Attr{})

	output := buf.String()
	// Should just have "empty attr" message with newline
	if strings.Contains(output, "=") {
		t.Errorf("expected empty attr to be ignored, got: %s", output)
	}
}

// LogValuer implementation for testing value resolution.
type secretValue struct {
	value string
}

func (s secretValue) LogValue() slog.Value {
	return slog.StringValue("[REDACTED]")
}

func TestHandler_LogValuerResolution(t *testing.T) {
	var buf bytes.Buffer
	h := NewHandler(&buf, &Options{NoColor: true})
	logger := slog.New(h)

	secret := secretValue{value: "my-secret-password"}
	logger.Info("with secret", slog.Any("password", secret))

	output := buf.String()
	if strings.Contains(output, "my-secret-password") {
		t.Errorf("expected secret to be redacted, got: %s", output)
	}
	if !strings.Contains(output, "[REDACTED]") {
		t.Errorf("expected [REDACTED] in output, got: %s", output)
	}
}

func TestHandler_MessageWithoutAttrs(t *testing.T) {
	var buf bytes.Buffer
	h := NewHandler(&buf, &Options{NoColor: true})
	logger := slog.New(h)

	logger.Info("simple message")

	output := buf.String()
	if !strings.Contains(output, "simple message") {
		t.Errorf("expected 'simple message', got: %s", output)
	}
	if !strings.Contains(output, "INF") {
		t.Errorf("expected 'INF' level, got: %s", output)
	}
}

func TestHandler_WithAttrsPreservesContext(t *testing.T) {
	var buf bytes.Buffer
	h := NewHandler(&buf, &Options{NoColor: true})
	logger := slog.New(h)

	// Add attrs at different levels
	logger1 := logger.With("a", 1)
	logger2 := logger1.With("b", 2)

	logger2.Info("test")

	output := buf.String()
	if !strings.Contains(output, "a=1") {
		t.Errorf("expected a=1, got: %s", output)
	}
	if !strings.Contains(output, "b=2") {
		t.Errorf("expected b=2, got: %s", output)
	}
}
