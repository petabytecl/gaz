package health

import (
	"context"
	"errors"
	"testing"

	"github.com/petabytecl/gaz/healthx"
)

func TestManager_LivenessChecker(t *testing.T) {
	m := NewManager()

	called := false
	m.AddLivenessCheck("test-check", func(_ context.Context) error {
		called = true
		return nil
	})

	checker := m.LivenessChecker()
	res := checker.Check(context.Background())

	if res.Status != healthx.StatusUp {
		t.Errorf("expected up status, got %s", res.Status)
	}
	if !called {
		t.Error("expected check to be called")
	}
}

func TestManager_ReadinessChecker(t *testing.T) {
	m := NewManager()

	m.AddReadinessCheck("fail-check", func(_ context.Context) error {
		return errors.New("oops")
	})

	checker := m.ReadinessChecker()
	res := checker.Check(context.Background())

	if res.Status == healthx.StatusUp {
		t.Error("expected failure status, got up")
	}
}

func TestManager_StartupChecker(t *testing.T) {
	m := NewManager()

	m.AddStartupCheck("startup-check", func(_ context.Context) error {
		return nil
	})

	checker := m.StartupChecker()
	res := checker.Check(context.Background())

	if res.Status != healthx.StatusUp {
		t.Errorf("expected up status, got %s", res.Status)
	}
}
