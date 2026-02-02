package internal

import (
	"testing"
	"time"
)

func TestEvery(t *testing.T) {
	tests := []struct {
		name          string
		duration      time.Duration
		expectedDelay time.Duration
	}{
		{
			name:          "5 minutes",
			duration:      5 * time.Minute,
			expectedDelay: 5 * time.Minute,
		},
		{
			name:          "1 hour",
			duration:      time.Hour,
			expectedDelay: time.Hour,
		},
		{
			name:          "30 seconds",
			duration:      30 * time.Second,
			expectedDelay: 30 * time.Second,
		},
		{
			name:          "sub-second rounds to 1 second",
			duration:      500 * time.Millisecond,
			expectedDelay: time.Second,
		},
		{
			name:          "zero rounds to 1 second",
			duration:      0,
			expectedDelay: time.Second,
		},
		{
			name:          "nanoseconds rounds to 1 second",
			duration:      1 * time.Nanosecond,
			expectedDelay: time.Second,
		},
		{
			name:          "truncates sub-second part",
			duration:      5*time.Minute + 500*time.Millisecond,
			expectedDelay: 5 * time.Minute,
		},
		{
			name:          "truncates nanoseconds",
			duration:      15*time.Minute + 50*time.Nanosecond,
			expectedDelay: 15 * time.Minute,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			schedule := Every(tt.duration)
			if schedule.Delay != tt.expectedDelay {
				t.Errorf("Every(%v).Delay = %v, expected %v", tt.duration, schedule.Delay, tt.expectedDelay)
			}
		})
	}
}

func TestConstantDelayScheduleNext(t *testing.T) {
	tests := []struct {
		name     string
		delay    time.Duration
		from     time.Time
		expected time.Time
	}{
		{
			name:     "15 minutes",
			delay:    15 * time.Minute,
			from:     time.Date(2024, 1, 15, 10, 30, 0, 0, time.Local),
			expected: time.Date(2024, 1, 15, 10, 45, 0, 0, time.Local),
		},
		{
			name:     "1 hour",
			delay:    time.Hour,
			from:     time.Date(2024, 1, 15, 10, 30, 0, 0, time.Local),
			expected: time.Date(2024, 1, 15, 11, 30, 0, 0, time.Local),
		},
		{
			name:     "wrap around midnight",
			delay:    2 * time.Hour,
			from:     time.Date(2024, 1, 15, 23, 30, 0, 0, time.Local),
			expected: time.Date(2024, 1, 16, 1, 30, 0, 0, time.Local),
		},
		{
			name:     "strips nanoseconds from from time",
			delay:    15 * time.Minute,
			from:     time.Date(2024, 1, 15, 10, 30, 0, 500000000, time.Local), // 500ms
			expected: time.Date(2024, 1, 15, 10, 45, 0, 0, time.Local),
		},
		{
			name:     "1 second",
			delay:    time.Second,
			from:     time.Date(2024, 1, 15, 10, 30, 45, 0, time.Local),
			expected: time.Date(2024, 1, 15, 10, 30, 46, 0, time.Local),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			schedule := Every(tt.delay)
			actual := schedule.Next(tt.from)
			if !actual.Equal(tt.expected) {
				t.Errorf("Next() = %v, expected %v", actual, tt.expected)
			}
		})
	}
}

func TestConstantDelayScheduleInterface(t *testing.T) {
	// Ensure ConstantDelaySchedule can be used as Schedule interface
	var _ interface {
		Next(time.Time) time.Time
	} = ConstantDelaySchedule{}
}
