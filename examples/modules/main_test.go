package main

import "testing"

func TestRun(t *testing.T) {
	if err := run(); err != nil {
		t.Fatalf("run() failed: %v", err)
	}
}
