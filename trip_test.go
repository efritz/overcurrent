package overcurrent

import . "gopkg.in/check.v1"

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
