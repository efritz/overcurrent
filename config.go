package overcurrent

import (
	"time"

	"github.com/efritz/backoff"
)

const (
	DefaultInvocationTimeout          = 100 * time.Millisecond
	DefaultHalfClosedRetryProbability = .5
)

//
//

type CircuitBreakerConfig struct {
	InvocationTimeout          time.Duration
	HalfClosedRetryProbability float64
	ResetBackOff               BackOff
	FailureInterpreter         FailureInterpreter
	TripCondition              TripCondition
}

// Creates a circuit breaker config usign the default values for timeouts
// and half-closed retry probability, a constnat retry backoff of 1000ms,
// an any-error failure interpreter, and atrip condition which fires only
// after five ocnsecutive failures.
func DefaultCircuitBreakerConfig() *CircuitBreakerConfig {
	return &CircuitBreakerConfig{
		InvocationTimeout:          DefaultInvocationTimeout,
		HalfClosedRetryProbability: DefaultHalfClosedRetryProbability,
		ResetBackOff:               backoff.NewConstantBackOff(1000 * time.Millisecond),
		FailureInterpreter:         NewAnyErrorFailureInterpreter(),
		TripCondition:              NewConsecutiveFailureTripCondition(5),
	}
}
