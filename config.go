package overcurrent

import (
	"time"

	"github.com/efritz/backoff"
)

const (
	DefaultInvocationTimeout          = 100 * time.Millisecond
	DefaultHalfClosedRetryProbability = .5
)

var (
	DefaultResetBackOff = backoff.NewConstantBackOff(1000 * time.Millisecond)
)

//
//

type CircuitBreakerConfig struct {
	InvocationTimeout          time.Duration
	ResetBackOff               BackOff
	HalfClosedRetryProbability float64
	FailureInterpreter         FailureInterpreter
	TripCondition              TripCondition
}

// Creates a circuit breaker config usign the default values for timeouts
// and half-closed retry probability, an any-error failure interpreter, and
// a consecutive-failure trip condition (with a value of five).
func DefaultCircuitBreakerConfig() *CircuitBreakerConfig {
	return &CircuitBreakerConfig{
		InvocationTimeout:          DefaultInvocationTimeout,
		ResetBackOff:               DefaultResetBackOff,
		HalfClosedRetryProbability: DefaultHalfClosedRetryProbability,
		FailureInterpreter:         NewAnyErrorFailureInterpreter(),
		TripCondition:              NewConsecutiveFailureTripCondition(5),
	}
}
