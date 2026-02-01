package cronx

import (
	"testing"
	"time"
)

func TestSpecScheduleNext(t *testing.T) {
	tests := []struct {
		name     string
		schedule *SpecSchedule
		from     time.Time
		expected time.Time
	}{
		{
			name: "every minute",
			schedule: &SpecSchedule{
				Second:   1 << 0, // second 0
				Minute:   ^uint64(0),
				Hour:     ^uint64(0),
				Dom:      ^uint64(0) | starBit,
				Month:    ^uint64(0) | starBit,
				Dow:      ^uint64(0) | starBit,
				Location: time.Local,
			},
			from:     time.Date(2024, 1, 15, 10, 30, 45, 0, time.Local),
			expected: time.Date(2024, 1, 15, 10, 31, 0, 0, time.Local),
		},
		{
			name: "specific hour",
			schedule: &SpecSchedule{
				Second:   1 << 0,
				Minute:   1 << 0,
				Hour:     1 << 9, // 9 AM
				Dom:      ^uint64(0) | starBit,
				Month:    ^uint64(0) | starBit,
				Dow:      ^uint64(0) | starBit,
				Location: time.Local,
			},
			from:     time.Date(2024, 1, 15, 10, 30, 0, 0, time.Local),
			expected: time.Date(2024, 1, 16, 9, 0, 0, 0, time.Local),
		},
		{
			name: "day wrap",
			schedule: &SpecSchedule{
				Second:   1 << 0,
				Minute:   1 << 0,
				Hour:     1 << 0, // midnight
				Dom:      ^uint64(0) | starBit,
				Month:    ^uint64(0) | starBit,
				Dow:      ^uint64(0) | starBit,
				Location: time.Local,
			},
			from:     time.Date(2024, 1, 15, 23, 59, 59, 0, time.Local),
			expected: time.Date(2024, 1, 16, 0, 0, 0, 0, time.Local),
		},
		{
			name: "month wrap",
			schedule: &SpecSchedule{
				Second:   1 << 0,
				Minute:   1 << 0,
				Hour:     1 << 0,
				Dom:      1 << 1, // 1st of month
				Month:    ^uint64(0) | starBit,
				Dow:      ^uint64(0) | starBit,
				Location: time.Local,
			},
			from:     time.Date(2024, 1, 15, 0, 0, 0, 0, time.Local),
			expected: time.Date(2024, 2, 1, 0, 0, 0, 0, time.Local),
		},
		{
			name: "year wrap",
			schedule: &SpecSchedule{
				Second:   1 << 0,
				Minute:   1 << 0,
				Hour:     1 << 0,
				Dom:      1 << 1,
				Month:    1 << 1, // January
				Dow:      ^uint64(0) | starBit,
				Location: time.Local,
			},
			from:     time.Date(2024, 2, 15, 0, 0, 0, 0, time.Local),
			expected: time.Date(2025, 1, 1, 0, 0, 0, 0, time.Local),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			actual := tt.schedule.Next(tt.from)
			if !actual.Equal(tt.expected) {
				t.Errorf("Next() = %v, expected %v", actual, tt.expected)
			}
		})
	}
}

func TestSpecScheduleNextWithTimezone(t *testing.T) {
	ny, err := time.LoadLocation("America/New_York")
	if err != nil {
		t.Skip("America/New_York timezone not available")
	}

	schedule := &SpecSchedule{
		Second:   1 << 0,
		Minute:   1 << 0,
		Hour:     1 << 9, // 9 AM
		Dom:      ^uint64(0) | starBit,
		Month:    ^uint64(0) | starBit,
		Dow:      ^uint64(0) | starBit,
		Location: ny,
	}

	// From UTC, should get next 9 AM New York time
	fromUTC := time.Date(2024, 1, 15, 15, 0, 0, 0, time.UTC) // 10 AM NY
	actual := schedule.Next(fromUTC)

	// Should be next day 9 AM NY, which is 14:00 UTC
	expected := time.Date(2024, 1, 16, 14, 0, 0, 0, time.UTC)
	if !actual.Equal(expected) {
		t.Errorf("Next() = %v, expected %v", actual, expected)
	}
}

func TestDayMatches(t *testing.T) {
	tests := []struct {
		name     string
		schedule *SpecSchedule
		t        time.Time
		expected bool
	}{
		{
			name: "dom and dow both star",
			schedule: &SpecSchedule{
				Dom: ^uint64(0) | starBit,
				Dow: ^uint64(0) | starBit,
			},
			t:        time.Date(2024, 1, 15, 0, 0, 0, 0, time.Local), // Monday
			expected: true,
		},
		{
			name: "dom star, dow specific match",
			schedule: &SpecSchedule{
				Dom: ^uint64(0) | starBit,
				Dow: 1 << 1, // Monday
			},
			t:        time.Date(2024, 1, 15, 0, 0, 0, 0, time.Local), // Monday
			expected: true,
		},
		{
			name: "dom star, dow specific no match",
			schedule: &SpecSchedule{
				Dom: ^uint64(0) | starBit,
				Dow: 1 << 0, // Sunday
			},
			t:        time.Date(2024, 1, 15, 0, 0, 0, 0, time.Local), // Monday
			expected: false,
		},
		{
			name: "both specific, dom matches",
			schedule: &SpecSchedule{
				Dom: 1 << 15, // 15th
				Dow: 1 << 0,  // Sunday
			},
			t:        time.Date(2024, 1, 15, 0, 0, 0, 0, time.Local), // Monday 15th
			expected: true,
		},
		{
			name: "both specific, dow matches",
			schedule: &SpecSchedule{
				Dom: 1 << 16, // 16th
				Dow: 1 << 1,  // Monday
			},
			t:        time.Date(2024, 1, 15, 0, 0, 0, 0, time.Local), // Monday 15th
			expected: true,
		},
		{
			name: "both specific, neither matches",
			schedule: &SpecSchedule{
				Dom: 1 << 16, // 16th
				Dow: 1 << 0,  // Sunday
			},
			t:        time.Date(2024, 1, 15, 0, 0, 0, 0, time.Local), // Monday 15th
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			actual := dayMatches(tt.schedule, tt.t)
			if actual != tt.expected {
				t.Errorf("dayMatches() = %v, expected %v", actual, tt.expected)
			}
		})
	}
}

func TestSpecScheduleNoMatch(t *testing.T) {
	// Schedule that can never be satisfied (Feb 30th)
	schedule := &SpecSchedule{
		Second:   1 << 0,
		Minute:   1 << 0,
		Hour:     1 << 0,
		Dom:      1 << 30, // 30th
		Month:    1 << 2,  // February
		Dow:      ^uint64(0) | starBit,
		Location: time.Local,
	}

	from := time.Date(2024, 1, 1, 0, 0, 0, 0, time.Local)
	actual := schedule.Next(from)

	if !actual.IsZero() {
		t.Errorf("Expected zero time for unsatisfiable schedule, got %v", actual)
	}
}
