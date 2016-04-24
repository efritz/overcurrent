package overcurrent

import (
	"time"
)

const (
	DefaultInvocationTimeout          = 100 * time.Millisecond
	DefaultResetTimeout               = 1000 * time.Millisecond
	DefaultHalfClosedRetryProbability = .5
)

//
//

type BreakerConfig struct {
	InvocationTimeout          time.Duration
	ResetTimeout               time.Duration
	HalfClosedRetryProbability float64
	FailureInterpreter         FailureInterpreter
	TripCondition              TripCondition
}

func NewBreakerConfig() BreakerConfig {
	return BreakerConfig{
		InvocationTimeout:          DefaultInvocationTimeout,
		ResetTimeout:               DefaultResetTimeout,
		HalfClosedRetryProbability: DefaultHalfClosedRetryProbability,
		FailureInterpreter:         NewAnyErrorFailureInterpreter(),
		TripCondition:              NewConsecutiveFailureTripCondition(5),
	}
}
