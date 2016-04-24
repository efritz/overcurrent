package overcurrent

import (
	"time"

	. "gopkg.in/check.v1"
)

func (s *OvercurrentSuite) TestConsecutive(c *C) {
	tc := NewConsecutiveFailureTripCondition(25)

	c.Assert(tc.ShouldTrip(), Equals, false)

	for i := 0; i < 24; i++ {
		tc.Failure()
	}

	c.Assert(tc.ShouldTrip(), Equals, false)
	tc.Failure()
	c.Assert(tc.ShouldTrip(), Equals, true)
}

func (s *OvercurrentSuite) TestConsecutiveBrokenChain(c *C) {
	tc := NewConsecutiveFailureTripCondition(25)

	c.Assert(tc.ShouldTrip(), Equals, false)

	for i := 0; i < 30; i++ {
		if i == 15 {
			tc.Success()
		}

		tc.Failure()
	}

	c.Assert(tc.ShouldTrip(), Equals, false)
}

func (s *OvercurrentSuite) TestWindow(c *C) {
	tc := NewWindowFailureTripCondition(150*time.Millisecond, 10)

	c.Assert(tc.ShouldTrip(), Equals, false)

	tc.Failure()
	<-time.After(50 * time.Millisecond)

	for i := 0; i < 8; i++ {
		tc.Success()
		tc.Failure()
	}

	<-time.After(50 * time.Millisecond)
	c.Assert(tc.ShouldTrip(), Equals, false)
	tc.Failure()
	c.Assert(tc.ShouldTrip(), Equals, true)

	<-time.After(50 * time.Millisecond)
	c.Assert(tc.ShouldTrip(), Equals, false)
	tc.Failure()
	c.Assert(tc.ShouldTrip(), Equals, true)
}
