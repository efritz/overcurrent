package overcurrent

type TripCondition interface {
	Failure()
	Success()
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

func NewConsecutiveFailureTripCondition(threshold int) *ConsecutiveFailureTripCondition {
	return &ConsecutiveFailureTripCondition{
		count:     0,
		threshold: threshold,
	}
}
