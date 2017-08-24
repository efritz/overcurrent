package overcurrent

import (
	"time"

	"github.com/aphistic/sweet"
	"github.com/efritz/glock"
	. "github.com/onsi/gomega"
)

type TripSuite struct{}

func (s *TripSuite) TestConsecutive(t sweet.T) {
	tc := NewConsecutiveFailureTripCondition(25)
	Expect(tc.ShouldTrip()).To(BeFalse())

	for i := 0; i < 24; i++ {
		tc.Failure()
	}

	Expect(tc.ShouldTrip()).To(BeFalse())
	tc.Failure()
	Expect(tc.ShouldTrip()).To(BeTrue())
}

func (s *TripSuite) TestConsecutiveBrokenChain(t sweet.T) {
	tc := NewConsecutiveFailureTripCondition(25)
	Expect(tc.ShouldTrip()).To(BeFalse())

	for i := 0; i < 30; i++ {
		if i == 15 {
			tc.Success()
		}

		tc.Failure()
	}

	Expect(tc.ShouldTrip()).To(BeFalse())
}

func (s *TripSuite) TestWindow(t sweet.T) {
	var (
		clock = glock.NewMockClock()
		tc    = newWindowFailureTripConditionWithClock(
			3*time.Second,
			10,
			clock,
		)
	)

	// failures -> 1 ... 2-9 ... 10 ... 10
	// window 1 -> [--------------]
	// window 2 ->       [---------------]

	// 1st
	Expect(tc.ShouldTrip()).To(BeFalse())
	tc.Failure()
	clock.Advance(time.Second)

	// 2nd .. 9th
	for i := 0; i < 8; i++ {
		tc.Success()
		tc.Failure()
	}

	// 1st 10th
	clock.Advance(time.Second)
	Expect(tc.ShouldTrip()).To(BeFalse())
	tc.Failure()
	Expect(tc.ShouldTrip()).To(BeTrue())

	// Expire one
	clock.Advance(time.Second)
	Expect(tc.ShouldTrip()).To(BeFalse())

	// 2nd 10th
	tc.Failure()
	Expect(tc.ShouldTrip()).To(BeTrue())
}

func (s *TripSuite) TestPercentage(t sweet.T) {
	tc := NewPercentageFailureTripCondition(100, .75)

	times(25, tc.Success)
	times(75, tc.Failure)
	Expect(tc.ShouldTrip()).To(BeTrue())
}

func (s *TripSuite) TestPercentageSmallWindow(t sweet.T) {
	tc := NewPercentageFailureTripCondition(100, .75)

	times(50, tc.Failure)
	Expect(tc.ShouldTrip()).To(BeFalse())
}

func (s *TripSuite) TestPercentageLogPushesOutSuccess(t sweet.T) {
	tc := NewPercentageFailureTripCondition(100, .75)

	times(25, tc.Success)
	times(75, tc.Failure)
	times(25, func() {
		tc.Success()
		Expect(tc.ShouldTrip()).To(BeTrue())
	})

	tc.Success()
	Expect(tc.ShouldTrip()).To(BeFalse())
}

func (s *TripSuite) TestPercentageLogPushesOutFailure(t sweet.T) {
	tc := NewPercentageFailureTripCondition(100, .75)

	times(75, tc.Failure)
	times(25, tc.Success)

	tc.Success()
	Expect(tc.ShouldTrip()).To(BeFalse())
}

func times(n int, f func()) {
	for i := 0; i < n; i++ {
		f()
	}
}
