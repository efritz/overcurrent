package overcurrent

const (
	DefaultInvocationTimeout          = 0.01
	DefaultResetTimeout               = 0.1
	DefaultHalfClosedRetryProbability = 0.5
)

//
//

type BreakerConfig struct {
	InvocationTimeout          float64
	ResetTimeout               float64
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
