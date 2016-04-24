package overcurrent

import (
	"errors"

	. "gopkg.in/check.v1"
)

func (s *OvercurrentSuite) TestAnyError(c *C) {
	afi := NewAnyErrorFailureInterpreter()
	err := errors.New("Test error.")

	c.Assert(afi.ShouldTrip(nil), Equals, true)
	c.Assert(afi.ShouldTrip(err), Equals, true)
}
