package internal

import "testing"

func TestAvailabilityStatus_Constants(t *testing.T) {
	tests := []struct {
		status   AvailabilityStatus
		expected string
	}{
		{StatusUnknown, "unknown"},
		{StatusUp, "up"},
		{StatusDown, "down"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			if string(tt.status) != tt.expected {
				t.Errorf("expected %q, got %q", tt.expected, string(tt.status))
			}
		})
	}
}

func TestAvailabilityStatus_String(t *testing.T) {
	tests := []struct {
		status   AvailabilityStatus
		expected string
	}{
		{StatusUnknown, "unknown"},
		{StatusUp, "up"},
		{StatusDown, "down"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			if tt.status.String() != tt.expected {
				t.Errorf("String() = %q, expected %q", tt.status.String(), tt.expected)
			}
		})
	}
}

func TestAvailabilityStatus_IsComparable(t *testing.T) {
	// Verify status can be compared with == operator
	upStatus := StatusUp
	if upStatus != StatusUp {
		t.Error("StatusUp should equal itself")
	}
	if StatusUp == StatusDown {
		t.Error("StatusUp should not equal StatusDown")
	}
}
