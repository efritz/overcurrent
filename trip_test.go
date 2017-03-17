package overcurrent

import (
	"testing"
	"time"

	. "github.com/onsi/gomega"
)

type TripSuite struct{}

func (s *TripSuite) TestConsecutive(t *testing.T) {
	tc := NewConsecutiveFailureTripCondition(25)
	Expect(tc.ShouldTrip()).To(BeFalse())

	for i := 0; i < 24; i++ {
		tc.Failure()
	}

	Expect(tc.ShouldTrip()).To(BeFalse())
	tc.Failure()
	Expect(tc.ShouldTrip()).To(BeTrue())
}

func (s *TripSuite) TestConsecutiveBrokenChain(t *testing.T) {
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

func (s *TripSuite) TestWindow(t *testing.T) {
	tc := NewWindowFailureTripCondition(150*time.Millisecond, 10)
	Expect(tc.ShouldTrip()).To(BeFalse())

	tc.Failure()
	<-time.After(50 * time.Millisecond)

	for i := 0; i < 8; i++ {
		tc.Success()
		tc.Failure()
	}

	<-time.After(50 * time.Millisecond)
	Expect(tc.ShouldTrip()).To(BeFalse())
	tc.Failure()
	Expect(tc.ShouldTrip()).To(BeTrue())

	<-time.After(50 * time.Millisecond)
	Expect(tc.ShouldTrip()).To(BeFalse())
	tc.Failure()
	Expect(tc.ShouldTrip()).To(BeTrue())
}
