package main

import (
	"bytes"
	"strings"
	"testing"
)

func TestExecuteVersion(t *testing.T) {
	buf := new(bytes.Buffer)
	if err := execute([]string{"version"}, buf); err != nil {
		t.Fatalf("execute(version) failed: %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "myapp v1.0.0") {
		t.Errorf("expected version output, got: %q", output)
	}
}

// Note: TestExecuteServe is tricky because runServe calls gaz.New().Run(cmd.Context())
// which blocks until signal. We can't easily test it without mocking app.Run or signal.
// But version command test verifies execute() logic.
