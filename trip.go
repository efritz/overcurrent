package overcurrent

// TripCondition is the interface that controls the open/closed state of the
// circuit breaker based on failure history.
type TripCondition interface {
	// Invoked when the circuit breaker was closed or half-closed and failure
	// occurs, or when the breaker is hard-tripped. Stats should be collected
	// at this point.
	Failure()

	// Invoked when the a call to a service completes successfully (note - all
	// successful calls, not just ones immediately after a failure period), or
	// when the breaker is hard reset. Stats should be reset at this point.
	Success()

	// Determines the state of the breaker - returns true or false for open and
	// closed state, respectively. This method should be idempotent and use the
	// state altered by the failure and success methods.
	ShouldTrip() bool
}

//
//

type ConsecutiveFailureTripCondition struct {
	count     int
	threshold int
}

func (tc *ConsecutiveFailureTripCondition) Failure()         { tc.count++ }
func (tc *ConsecutiveFailureTripCondition) Success()         { tc.count = 0 }
func (tc *ConsecutiveFailureTripCondition) ShouldTrip() bool { return tc.count >= tc.threshold }

// A trip condition that trips the circuit breaker after a configurable number
// of failures occur in a row. A single successful call will break the failure
// chain.
func NewConsecutiveFailureTripCondition(threshold int) *ConsecutiveFailureTripCondition {
	return &ConsecutiveFailureTripCondition{
		count:     0,
		threshold: threshold,
	}
}
