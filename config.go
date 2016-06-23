package overcurrent

import (
	"time"

	"github.com/efritz/backoff"
)

const (
	// DefaultInvocationTimeout is the invocation timeout used by
	// DefaultCircuitBreakerConfig.
	DefaultInvocationTimeout = 100 * time.Millisecond

	// DefaultHalfClosedRetryProbability is the probability that
	// the function shoudl be called while the CircuitBreaker is
	// in half-closed state used by DefaultCircuitBreakerConfig.
	DefaultHalfClosedRetryProbability = .5
)

//
//

// CircuitBreakerConfig is a configuration struct which describes the
// behavior of a CircuitBreaker.
type CircuitBreakerConfig struct {
	InvocationTimeout          time.Duration
	HalfClosedRetryProbability float64
	ResetBackoff               Backoff
	FailureInterpreter         FailureInterpreter
	TripCondition              TripCondition
}

// DefaultCircuitBreakerConfig creates a circuit breaker config usign the
// default values for timeouts and half-closed retry probability, a constant
// retry backoff of 1000ms, an any-error failure interpreter, and a trip
// condition which fires only after five ocnsecutive failures.
func DefaultCircuitBreakerConfig() *CircuitBreakerConfig {
	return &CircuitBreakerConfig{
		InvocationTimeout:          DefaultInvocationTimeout,
		HalfClosedRetryProbability: DefaultHalfClosedRetryProbability,
		ResetBackoff:               backoff.NewConstantBackoff(1000 * time.Millisecond),
		FailureInterpreter:         NewAnyErrorFailureInterpreter(),
		TripCondition:              NewConsecutiveFailureTripCondition(5),
	}
}
