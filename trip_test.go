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
	var (
		clock = newMockClock()
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
	clock.advance(time.Second)

	// 2nd .. 9th
	for i := 0; i < 8; i++ {
		tc.Success()
		tc.Failure()
	}

	// 1st 10th
	clock.advance(time.Second)
	Expect(tc.ShouldTrip()).To(BeFalse())
	tc.Failure()
	Expect(tc.ShouldTrip()).To(BeTrue())

	// Expire one
	clock.advance(time.Second)
	Expect(tc.ShouldTrip()).To(BeFalse())

	// 2nd 10th
	tc.Failure()
	Expect(tc.ShouldTrip()).To(BeTrue())
}

//
//
//

type mockClock struct {
	now time.Time
}

func newMockClock() *mockClock {
	return &mockClock{
		now: time.Now(),
	}
}

func (m *mockClock) Now() time.Time {
	return m.now
}

func (m *mockClock) advance(d time.Duration) {
	m.now = m.now.Add(d)
}
