package overcurrent

import (
	"errors"

	"github.com/aphistic/sweet"
	. "github.com/onsi/gomega"
)

type UtilSuite struct{}

func (s *UtilSuite) TestToErrChan(t sweet.T) {
	var (
		ex = errors.New("utoh")
		ch = toErrChan(func() error {
			return ex
		})
	)

	Eventually(ch).Should(Receive(Equal(ex)))
	Eventually(ch).Should(BeClosed())
}
