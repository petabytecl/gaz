package healthx

// AvailabilityStatus represents system/component availability.
type AvailabilityStatus string

const (
	// StatusUnknown means the status is not yet known.
	StatusUnknown AvailabilityStatus = "unknown"
	// StatusUp means the system/component is available.
	StatusUp AvailabilityStatus = "up"
	// StatusDown means the system/component is unavailable.
	StatusDown AvailabilityStatus = "down"
)

// String returns the string representation of the status.
func (s AvailabilityStatus) String() string {
	return string(s)
}
