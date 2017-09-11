package plugins

import (
	"github.com/aphistic/sweet"
	. "github.com/onsi/gomega"
)

type HystrixSuite struct{}

func (s *HystrixSuite) TestX(t sweet.T) {
	Expect(true).To(BeTrue()) // TODO
}
