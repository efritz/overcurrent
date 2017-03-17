package overcurrent

import "time"

type (
	// TripCondition is the interface that controls the open/closed state of the
	// circuit breaker based on failure history.
	TripCondition interface {
		// Invoked when the circuit breaker was closed or half-closed and failure
		// occurs, or when the breaker is hard-tripped. Stats should be collected
		// at this point.
		Failure()

		// Invoked when the a call to a service completes successfully (note - all
		// successful calls, not just ones immediately after a failure period), or
		// when the breaker is hard reset. Stats should be reset at this point.
		Success()

		// Determines the state of the breaker - returns true or false for open and
		// closed state, respectively.
		ShouldTrip() bool
	}

	// ConsecutiveFailureTripCondition is a trip condition that trips the circuit
	// breaker after a configurable number of failures occur in a row. A single
	// successful call will break the failure chain.
	ConsecutiveFailureTripCondition struct {
		count     int
		threshold int
	}

	// WindowFailureTripCondition is a trip condition that trips the circuit
	// breaker after a configurable number of failures occur within a configurable
	// rolling window. The circuit breaker will remain in the open state until
	// e time has elapsed for a failure to fall out of recent memory.
	WindowFailureTripCondition struct {
		log       []time.Time
		window    time.Duration
		threshold int
	}
)

// Failure increases the failure count.
func (tc *ConsecutiveFailureTripCondition) Failure() {
	tc.count++
}

// Success resets the failure count.
func (tc *ConsecutiveFailureTripCondition) Success() {
	tc.count = 0
}

// ShouldTrip returns true if the failure count meets or exceeds the failure threshold.
func (tc *ConsecutiveFailureTripCondition) ShouldTrip() bool {
	return tc.count >= tc.threshold
}

// NewConsecutiveFailureTripCondition creates a new ConsecutiveFailureTripCondition.
func NewConsecutiveFailureTripCondition(threshold int) *ConsecutiveFailureTripCondition {
	return &ConsecutiveFailureTripCondition{
		count:     0,
		threshold: threshold,
	}
}

// Failure logs the time of this failure.
func (tc *WindowFailureTripCondition) Failure() {
	tc.log = append(tc.log, time.Now())
}

// Success is a no-op.
func (tc *WindowFailureTripCondition) Success() {
}

// ShouldTrip returns true if the number of logged failures within the window
// meets or exceeds the failure threshold.
func (tc *WindowFailureTripCondition) ShouldTrip() bool {
	i := 0
	for i < len(tc.log) && time.Now().Sub(tc.log[i]) > tc.window {
		i++
	}

	if i > 0 {
		tc.log = tc.log[i:]
	}

	return len(tc.log) >= tc.threshold
}

// NewWindowFailureTripCondition creates a new WindowFailureTripCondition.
func NewWindowFailureTripCondition(window time.Duration, threshold int) *WindowFailureTripCondition {
	return &WindowFailureTripCondition{
		log:       []time.Time{},
		window:    window,
		threshold: threshold,
	}
}
