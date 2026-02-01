package backoff

import (
	"testing"
	"time"
)

// testClock is a mock clock for testing elapsed time behavior.
type testClock struct {
	now time.Time
}

func (c *testClock) Now() time.Time          { return c.now }
func (c *testClock) Advance(d time.Duration) { c.now = c.now.Add(d) }

func TestNewExponentialBackOff_Defaults(t *testing.T) {
	b := NewExponentialBackOff()

	if b.InitialInterval != DefaultInitialInterval {
		t.Errorf("InitialInterval = %v, want %v", b.InitialInterval, DefaultInitialInterval)
	}
	if b.MaxInterval != DefaultMaxInterval {
		t.Errorf("MaxInterval = %v, want %v", b.MaxInterval, DefaultMaxInterval)
	}
	if b.Multiplier != DefaultMultiplier {
		t.Errorf("Multiplier = %v, want %v", b.Multiplier, DefaultMultiplier)
	}
	if b.RandomizationFactor != DefaultRandomizationFactor {
		t.Errorf("RandomizationFactor = %v, want %v", b.RandomizationFactor, DefaultRandomizationFactor)
	}
	if b.MaxElapsedTime != DefaultMaxElapsedTime {
		t.Errorf("MaxElapsedTime = %v, want %v", b.MaxElapsedTime, DefaultMaxElapsedTime)
	}
	if b.Stop != Stop {
		t.Errorf("Stop = %v, want %v", b.Stop, Stop)
	}
	if b.Clock != SystemClock {
		t.Errorf("Clock = %v, want SystemClock", b.Clock)
	}
}

func TestExponentialBackOff_Options(t *testing.T) {
	clock := &testClock{now: time.Now()}

	b := NewExponentialBackOff(
		WithInitialInterval(500*time.Millisecond),
		WithMaxInterval(10*time.Second),
		WithMultiplier(1.5),
		WithRandomizationFactor(0.3),
		WithMaxElapsedTime(30*time.Second),
		WithClock(clock),
	)

	if b.InitialInterval != 500*time.Millisecond {
		t.Errorf("InitialInterval = %v, want 500ms", b.InitialInterval)
	}
	if b.MaxInterval != 10*time.Second {
		t.Errorf("MaxInterval = %v, want 10s", b.MaxInterval)
	}
	if b.Multiplier != 1.5 {
		t.Errorf("Multiplier = %v, want 1.5", b.Multiplier)
	}
	if b.RandomizationFactor != 0.3 {
		t.Errorf("RandomizationFactor = %v, want 0.3", b.RandomizationFactor)
	}
	if b.MaxElapsedTime != 30*time.Second {
		t.Errorf("MaxElapsedTime = %v, want 30s", b.MaxElapsedTime)
	}
	if b.Clock != clock {
		t.Errorf("Clock not set correctly")
	}
}

func TestExponentialBackOff_InvalidOptions(t *testing.T) {
	// Invalid options should be ignored, keeping defaults
	b := NewExponentialBackOff(
		WithInitialInterval(-1),          // invalid
		WithMaxInterval(0),               // invalid
		WithMultiplier(-1),               // invalid
		WithRandomizationFactor(-0.5),    // invalid
		WithRandomizationFactor(1.5),     // invalid
		WithMaxElapsedTime(-time.Second), // invalid
		WithClock(nil),                   // invalid
	)

	if b.InitialInterval != DefaultInitialInterval {
		t.Errorf("InitialInterval = %v, want default %v", b.InitialInterval, DefaultInitialInterval)
	}
	if b.MaxInterval != DefaultMaxInterval {
		t.Errorf("MaxInterval = %v, want default %v", b.MaxInterval, DefaultMaxInterval)
	}
	if b.Multiplier != DefaultMultiplier {
		t.Errorf("Multiplier = %v, want default %v", b.Multiplier, DefaultMultiplier)
	}
	if b.RandomizationFactor != DefaultRandomizationFactor {
		t.Errorf("RandomizationFactor = %v, want default %v", b.RandomizationFactor, DefaultRandomizationFactor)
	}
	if b.Clock != SystemClock {
		t.Errorf("Clock should remain SystemClock when nil is passed")
	}
}

func TestExponentialBackOff_NextBackOff_Increases(t *testing.T) {
	b := NewExponentialBackOff(
		WithInitialInterval(100*time.Millisecond),
		WithMaxInterval(10*time.Second),
		WithRandomizationFactor(0), // no jitter for predictable testing
	)

	// First call should return InitialInterval
	got := b.NextBackOff()
	if got != 100*time.Millisecond {
		t.Errorf("First NextBackOff() = %v, want 100ms", got)
	}

	// Second call should return InitialInterval * Multiplier
	got = b.NextBackOff()
	expected := time.Duration(float64(100*time.Millisecond) * DefaultMultiplier)
	if got != expected {
		t.Errorf("Second NextBackOff() = %v, want %v", got, expected)
	}

	// Third call should return InitialInterval * Multiplier^2
	got = b.NextBackOff()
	expected = time.Duration(float64(100*time.Millisecond) * DefaultMultiplier * DefaultMultiplier)
	if got != expected {
		t.Errorf("Third NextBackOff() = %v, want %v", got, expected)
	}
}

func TestExponentialBackOff_Reset(t *testing.T) {
	b := NewExponentialBackOff(
		WithInitialInterval(100*time.Millisecond),
		WithRandomizationFactor(0),
	)

	// Advance a few iterations
	b.NextBackOff() // 100ms
	b.NextBackOff() // 200ms
	b.NextBackOff() // 400ms

	// Reset should restore initial interval
	b.Reset()

	got := b.NextBackOff()
	if got != 100*time.Millisecond {
		t.Errorf("After Reset(), NextBackOff() = %v, want 100ms", got)
	}
}

func TestExponentialBackOff_OverflowProtection(t *testing.T) {
	b := NewExponentialBackOff(
		WithInitialInterval(1*time.Minute),
		WithMaxInterval(5*time.Minute),
		WithMultiplier(10),               // Large multiplier to trigger overflow protection quickly
		WithRandomizationFactor(0),       // no jitter
		WithMaxElapsedTime(time.Hour*24), // Large enough to not trigger Stop
	)

	// After first call: currentInterval becomes 10 * 1min = 10min, but capped at 5min
	first := b.NextBackOff()
	if first != 1*time.Minute {
		t.Errorf("First NextBackOff() = %v, want 1min", first)
	}

	// Second call should return MaxInterval (capped)
	second := b.NextBackOff()
	if second != 5*time.Minute {
		t.Errorf("Second NextBackOff() = %v, want MaxInterval (5min)", second)
	}

	// Third call should also return MaxInterval
	third := b.NextBackOff()
	if third != 5*time.Minute {
		t.Errorf("Third NextBackOff() = %v, want MaxInterval (5min)", third)
	}

	// Verify currentInterval never exceeds MaxInterval (no negative durations)
	if b.currentInterval > b.MaxInterval {
		t.Errorf("currentInterval = %v, exceeds MaxInterval = %v", b.currentInterval, b.MaxInterval)
	}
	if b.currentInterval < 0 {
		t.Errorf("currentInterval = %v, is negative (overflow)", b.currentInterval)
	}
}

func TestExponentialBackOff_MaxElapsedTime(t *testing.T) {
	clock := &testClock{now: time.Now()}

	b := NewExponentialBackOff(
		WithInitialInterval(1*time.Second),
		WithMaxElapsedTime(5*time.Second),
		WithRandomizationFactor(0),
		WithClock(clock),
	)

	// First few calls should succeed
	got := b.NextBackOff()
	if got == Stop {
		t.Error("First NextBackOff() returned Stop, expected a duration")
	}

	// Advance time past MaxElapsedTime
	clock.Advance(6 * time.Second)

	// Next call should return Stop
	got = b.NextBackOff()
	if got != Stop {
		t.Errorf("After MaxElapsedTime exceeded, NextBackOff() = %v, want Stop", got)
	}
}

func TestExponentialBackOff_ZeroRandomizationFactor(t *testing.T) {
	b := NewExponentialBackOff(
		WithInitialInterval(100*time.Millisecond),
		WithRandomizationFactor(0),
	)

	// With zero randomization, result should be exactly currentInterval
	got := b.NextBackOff()
	if got != 100*time.Millisecond {
		t.Errorf("NextBackOff() with zero randomization = %v, want exactly 100ms", got)
	}
}

func TestExponentialBackOff_RandomizationInRange(t *testing.T) {
	b := NewExponentialBackOff(
		WithInitialInterval(100*time.Millisecond),
		WithRandomizationFactor(0.5),
	)

	// Run multiple times to check jitter is within expected range
	minExpected := 50 * time.Millisecond  // 100ms - 50%
	maxExpected := 151 * time.Millisecond // 100ms + 50% + 1 (due to formula)

	for i := 0; i < 100; i++ {
		// Reset to always get first interval
		b.Reset()
		got := b.NextBackOff()
		if got < minExpected || got > maxExpected {
			t.Errorf("NextBackOff() = %v, expected in range [%v, %v]", got, minExpected, maxExpected)
		}
	}
}

func TestExponentialBackOff_GetElapsedTime(t *testing.T) {
	clock := &testClock{now: time.Now()}
	b := NewExponentialBackOff(WithClock(clock))

	// Initial elapsed time should be 0
	if elapsed := b.GetElapsedTime(); elapsed != 0 {
		t.Errorf("Initial GetElapsedTime() = %v, want 0", elapsed)
	}

	// Advance clock
	clock.Advance(5 * time.Second)

	if elapsed := b.GetElapsedTime(); elapsed != 5*time.Second {
		t.Errorf("GetElapsedTime() = %v, want 5s", elapsed)
	}

	// Reset should restart the clock
	b.Reset()
	if elapsed := b.GetElapsedTime(); elapsed != 0 {
		t.Errorf("After Reset(), GetElapsedTime() = %v, want 0", elapsed)
	}
}

func TestExponentialBackOff_InterfaceCompliance(t *testing.T) {
	// Verify ExponentialBackOff implements BackOff interface
	var _ BackOff = (*ExponentialBackOff)(nil)
}

func TestGetRandomValueFromInterval(t *testing.T) {
	tests := []struct {
		name                string
		randomizationFactor float64
		random              float64 // 0.0 to 1.0
		currentInterval     time.Duration
		wantMin             time.Duration
		wantMax             time.Duration
	}{
		{
			name:                "zero factor returns exact interval",
			randomizationFactor: 0,
			random:              0.5,
			currentInterval:     time.Second,
			wantMin:             time.Second,
			wantMax:             time.Second,
		},
		{
			name:                "0.5 factor with random=0 returns min",
			randomizationFactor: 0.5,
			random:              0,
			currentInterval:     time.Second,
			wantMin:             500 * time.Millisecond,
			wantMax:             501 * time.Millisecond, // allow 1ns tolerance
		},
		{
			name:                "0.5 factor with random=1 returns max",
			randomizationFactor: 0.5,
			random:              1,
			currentInterval:     time.Second,
			wantMin:             1500 * time.Millisecond,
			wantMax:             1502 * time.Millisecond, // allow tolerance for +1 in formula
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := getRandomValueFromInterval(tt.randomizationFactor, tt.random, tt.currentInterval)
			if got < tt.wantMin || got > tt.wantMax {
				t.Errorf("getRandomValueFromInterval(%v, %v, %v) = %v, want in [%v, %v]",
					tt.randomizationFactor, tt.random, tt.currentInterval,
					got, tt.wantMin, tt.wantMax)
			}
		})
	}
}
