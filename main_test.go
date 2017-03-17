package overcurrent

import (
	"testing"

	"github.com/aphistic/sweet"
	. "github.com/onsi/gomega"
)

func Test(t *testing.T) {
	sweet.T(func(s *sweet.S) {
		RegisterFailHandler(sweet.GomegaFail)

		s.RunSuite(t, &TripSuite{})
		s.RunSuite(t, &FailureSuite{})
		s.RunSuite(t, &BreakerSuite{})
	})
}
