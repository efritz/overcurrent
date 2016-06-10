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

type BreakerConfig struct {
	InvocationTimeout          time.Duration
	ResetBackOff               BackOff
	HalfClosedRetryProbability float64
	FailureInterpreter         FailureInterpreter
	TripCondition              TripCondition
}

func NewBreakerConfig() BreakerConfig {
	return BreakerConfig{
		InvocationTimeout:          DefaultInvocationTimeout,
		ResetBackOff:               DefaultResetBackOff,
		HalfClosedRetryProbability: DefaultHalfClosedRetryProbability,
		FailureInterpreter:         NewAnyErrorFailureInterpreter(),
		TripCondition:              NewConsecutiveFailureTripCondition(5),
	}
}
