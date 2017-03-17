package overcurrent

import (
	"errors"
	"testing"

	. "github.com/onsi/gomega"
)

type FailureSuite struct{}

func (s *FailureSuite) TestAnyError(t *testing.T) {
	var (
		afi = NewAnyErrorFailureInterpreter()
		err = errors.New("test error")
	)

	Expect(afi.ShouldTrip(nil)).To(BeTrue())
	Expect(afi.ShouldTrip(err)).To(BeTrue())
}
