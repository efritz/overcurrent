package hystrix

import (
	"github.com/aphistic/sweet"
	. "github.com/onsi/gomega"
)

type CollectorSuite struct{}

func (s *CollectorSuite) TestX(t sweet.T) {
	Expect(true).To(BeTrue()) // TODO
}
