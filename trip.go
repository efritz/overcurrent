package overcurrent

import (
	"time"

	"github.com/efritz/glock"
)

type (
	// TripCondition is the interface that controls the open/closed state of the
	// circuit breaker based on failure history.
	TripCondition interface {
		// Invoked when the a call to a service completes successfully (note - all
		// successful calls, not just ones immediately after a failure period), or
		// when the breaker is hard reset. Stats should be reset at this point.
		Success()

		// Invoked when the circuit breaker was closed or half-closed and failure
		// occurs, or when the breaker is hard-tripped. Stats should be collected
		// at this point.
		Failure()

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
	// the time has elapsed for a failure to fall out of recent memory.
	WindowFailureTripCondition struct {
		log       []time.Time
		window    time.Duration
		threshold int
		clock     glock.Clock
	}

	// PercentageFailureTripCondition is a trip condition that trips the circuit
	// breaker after the number of failures for the last fixed number of attempts
	// exceeds a percentage threshold.
	PercentageFailureTripCondition struct {
		log       []bool
		window    int
		threshold float64
		failures  int
	}
)

// NewConsecutiveFailureTripCondition creates a ConsecutiveFailureTripCondition.
func NewConsecutiveFailureTripCondition(threshold int) TripCondition {
	return &ConsecutiveFailureTripCondition{
		count:     0,
		threshold: threshold,
	}
}

func (tc *ConsecutiveFailureTripCondition) Success() {
	tc.count = 0
}

func (tc *ConsecutiveFailureTripCondition) Failure() {
	tc.count++
}

func (tc *ConsecutiveFailureTripCondition) ShouldTrip() bool {
	return tc.count >= tc.threshold
}

// NewWindowFailureTripCondition creates a WindowFailureTripCondition.
func NewWindowFailureTripCondition(window time.Duration, threshold int) TripCondition {
	return newWindowFailureTripConditionWithClock(window, threshold, glock.NewRealClock())
}

func newWindowFailureTripConditionWithClock(window time.Duration, threshold int, clock glock.Clock) TripCondition {
	return &WindowFailureTripCondition{
		log:       []time.Time{},
		window:    window,
		threshold: threshold,
		clock:     clock,
	}
}

func (tc *WindowFailureTripCondition) Success() {
}

func (tc *WindowFailureTripCondition) Failure() {
	tc.log = append(tc.log, tc.clock.Now())
}

func (tc *WindowFailureTripCondition) ShouldTrip() bool {
	for len(tc.log) != 0 && tc.clock.Now().Sub(tc.log[0]) >= tc.window {
		tc.log = tc.log[1:]
	}

	return len(tc.log) >= tc.threshold
}

// NewPercentageFailureFailureTripCondition creates a PercentageFailureTripCondition.
func NewPercentageFailureTripCondition(window int, threshold float64) TripCondition {
	return &PercentageFailureTripCondition{
		log:       []bool{},
		window:    window,
		threshold: threshold,
		failures:  0,
	}
}

func (tc *PercentageFailureTripCondition) Success() {
	tc.addToLog(true)
}

func (tc *PercentageFailureTripCondition) Failure() {
	tc.addToLog(false)
}

func (tc *PercentageFailureTripCondition) ShouldTrip() bool {
	if len(tc.log) < tc.window {
		return false
	}
	return float64(tc.failures)/float64(len(tc.log)) >= tc.threshold
}

func (tc *PercentageFailureTripCondition) addToLog(value bool) {
	tc.log = append(tc.log, value)

	if !value {
		tc.failures++
	}

	for len(tc.log) > tc.window {
		if !tc.log[0] {
			tc.failures--
		}

		tc.log = tc.log[1:]
	}
}
