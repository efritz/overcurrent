package overcurrent

import (
	"errors"

	"github.com/aphistic/sweet"
	. "github.com/onsi/gomega"
)

type FailureSuite struct{}

func (s *FailureSuite) TestAnyError(t sweet.T) {
	var (
		afi = NewAnyErrorFailureInterpreter()
		err = errors.New("test error")
	)

	Expect(afi.ShouldTrip(nil)).To(BeTrue())
	Expect(afi.ShouldTrip(err)).To(BeTrue())
}
