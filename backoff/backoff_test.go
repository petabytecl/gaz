package backoff

import (
	"testing"
	"time"
)

func TestStop(t *testing.T) {
	if Stop != -1 {
		t.Errorf("Stop = %v, want -1", Stop)
	}
}

func TestZeroBackOff(t *testing.T) {
	t.Run("NextBackOff returns 0", func(t *testing.T) {
		b := &ZeroBackOff{}
		for i := 0; i < 5; i++ {
			got := b.NextBackOff()
			if got != 0 {
				t.Errorf("NextBackOff() = %v, want 0", got)
			}
		}
	})

	t.Run("Reset does not panic", func(t *testing.T) {
		b := &ZeroBackOff{}
		b.Reset() // Should not panic
	})
}

func TestStopBackOff(t *testing.T) {
	t.Run("NextBackOff returns Stop", func(t *testing.T) {
		b := &StopBackOff{}
		for i := 0; i < 5; i++ {
			got := b.NextBackOff()
			if got != Stop {
				t.Errorf("NextBackOff() = %v, want Stop (%v)", got, Stop)
			}
		}
	})

	t.Run("Reset does not panic", func(t *testing.T) {
		b := &StopBackOff{}
		b.Reset() // Should not panic
	})
}

func TestConstantBackOff(t *testing.T) {
	delay := 500 * time.Millisecond

	t.Run("NextBackOff returns configured delay", func(t *testing.T) {
		b := NewConstantBackOff(delay)
		for i := 0; i < 5; i++ {
			got := b.NextBackOff()
			if got != delay {
				t.Errorf("NextBackOff() = %v, want %v", got, delay)
			}
		}
	})

	t.Run("Reset does not panic", func(t *testing.T) {
		b := NewConstantBackOff(delay)
		b.Reset() // Should not panic
	})

	t.Run("Zero delay works", func(t *testing.T) {
		b := NewConstantBackOff(0)
		got := b.NextBackOff()
		if got != 0 {
			t.Errorf("NextBackOff() = %v, want 0", got)
		}
	})
}

func TestInterfaceCompliance(t *testing.T) {
	// This test verifies that all types implement the BackOff interface.
	// The compile-time checks in backoff.go should catch issues, but
	// this test provides runtime verification.
	var _ BackOff = (*ZeroBackOff)(nil)
	var _ BackOff = (*StopBackOff)(nil)
	var _ BackOff = (*ConstantBackOff)(nil)
}
