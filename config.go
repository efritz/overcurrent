package overcurrent

import (
	"time"
)

const (
	DefaultInvocationTimeout          = 100 * time.Millisecond
	DefaultResetTimeout               = 100 * time.Millisecond
	DefaultHalfClosedRetryProbability = .5
)

//
//

type BreakerConfig struct {
	InvocationTimeout          time.Duration
	ResetTimeout               time.Duration
	HalfClosedRetryProbability float64
	TripCondition              TripCondition
	FailureInterpreter         FailureInterpreter
}

func NewBreakerConfig() BreakerConfig {
	return BreakerConfig{
		InvocationTimeout:          DefaultInvocationTimeout,
		ResetTimeout:               DefaultResetTimeout,
		HalfClosedRetryProbability: DefaultHalfClosedRetryProbability,
		TripCondition:              NewConsecutiveFailureTripCondition(5),
		FailureInterpreter:         NewAnyErrorFailureInterpreter(),
	}
}
