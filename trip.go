package overcurrent

import "time"
import "github.com/efritz/glock"

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
		clock     glock.Clock
	}
)

// NewConsecutiveFailureTripCondition creates a new ConsecutiveFailureTripCondition.
func NewConsecutiveFailureTripCondition(threshold int) *ConsecutiveFailureTripCondition {
	return &ConsecutiveFailureTripCondition{
		count:     0,
		threshold: threshold,
	}
}

func (tc *ConsecutiveFailureTripCondition) Failure() {
	tc.count++
}

func (tc *ConsecutiveFailureTripCondition) Success() {
	tc.count = 0
}

func (tc *ConsecutiveFailureTripCondition) ShouldTrip() bool {
	return tc.count >= tc.threshold
}

// NewWindowFailureTripCondition creates a new WindowFailureTripCondition.
func NewWindowFailureTripCondition(window time.Duration, threshold int) *WindowFailureTripCondition {
	return newWindowFailureTripConditionWithClock(window, threshold, glock.NewRealClock())

}

func newWindowFailureTripConditionWithClock(window time.Duration, threshold int, clock glock.Clock) *WindowFailureTripCondition {
	return &WindowFailureTripCondition{
		log:       []time.Time{},
		window:    window,
		threshold: threshold,
		clock:     clock,
	}
}

func (tc *WindowFailureTripCondition) Failure() {
	tc.log = append(tc.log, tc.clock.Now())
}

func (tc *WindowFailureTripCondition) Success() {
}

func (tc *WindowFailureTripCondition) ShouldTrip() bool {
	for len(tc.log) != 0 && tc.clock.Now().Sub(tc.log[0]) >= tc.window {
		tc.log = tc.log[1:]
	}

	return len(tc.log) >= tc.threshold
}
