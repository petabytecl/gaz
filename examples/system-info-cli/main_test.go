package main

import (
	"bytes"
	"strings"
	"testing"
)

func TestExecuteVersion(t *testing.T) {
	buf := new(bytes.Buffer)
	// We run the "version" command
	if err := execute([]string{"version"}, buf); err != nil {
		t.Fatalf("execute(version) failed: %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "sysinfo v1.0.0") {
		t.Errorf("expected version output, got: %q", output)
	}
}
