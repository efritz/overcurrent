package plugins

import (
	"github.com/aphistic/sweet"
	. "github.com/onsi/gomega"
)

type UtilSuite struct{}

func (s *UtilSuite) TestX(t sweet.T) {
	Expect(true).To(BeTrue()) // TODO
}
