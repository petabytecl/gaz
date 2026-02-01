package backoff

import "time"

// Timer is an interface for a timer that can be used to wait for backoff durations.
// This abstraction enables testing without real time delays.
type Timer interface {
	// Start begins the timer to fire after the given duration.
	Start(duration time.Duration)
	// Stop cancels the timer. It should be called when the timer is no longer needed.
	Stop()
	// C returns the channel that receives the current time when the timer fires.
	C() <-chan time.Time
}

// defaultTimer implements Timer interface using time.Timer.
type defaultTimer struct {
	timer *time.Timer
}

// C returns the timer's channel which receives the current time when the timer fires.
func (t *defaultTimer) C() <-chan time.Time {
	return t.timer.C
}

// Start starts the timer to fire after the given duration.
func (t *defaultTimer) Start(duration time.Duration) {
	if t.timer == nil {
		t.timer = time.NewTimer(duration)
	} else {
		t.timer.Reset(duration)
	}
}

// Stop is called when the timer is not used anymore and resources may be freed.
func (t *defaultTimer) Stop() {
	if t.timer != nil {
		t.timer.Stop()
	}
}
