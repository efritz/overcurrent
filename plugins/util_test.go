package plugins

import (
	"time"

	"github.com/aphistic/sweet"
	. "github.com/onsi/gomega"
)

type UtilSuite struct{}

func (s *UtilSuite) TestSortDurations(t sweet.T) {
	values := []time.Duration{
		time.Second * 2,
		time.Second * 5,
		time.Second * 4,
		time.Second * 1,
		time.Second * 3,
	}

	sorted := sortDurations(values)
	Expect(sorted).To(BeEquivalentTo(values))

	Expect(sorted).To(Equal([]time.Duration{
		time.Second * 1,
		time.Second * 2,
		time.Second * 3,
		time.Second * 4,
		time.Second * 5,
	}))
}

func (s *UtilSuite) TestMean(t sweet.T) {
	values := []time.Duration{
		time.Second * 1,
		time.Second * 2,
		time.Second * 3,
		time.Second * 4,
		time.Second * 5,
	}

	Expect(mean(values)).To(Equal(time.Second * 3))
}

func (s *UtilSuite) TestMeanZero(t sweet.T) {
	Expect(mean(nil)).To(Equal(time.Duration(0)))
}

func (s *UtilSuite) TestPercentiles(t sweet.T) {
	values := []time.Duration{}
	for i := 0; i < 100; i++ {
		values = append(values, time.Second*time.Duration(i))
	}

	for i := 0; i < 100; i++ {
		Expect(percentile(values, float64(i)/100)).To(Equal(values[i]))
	}

	Expect(percentile(values, 0)).To(Equal(time.Duration(0)))
	Expect(percentile(values, 1)).To(Equal(time.Duration(time.Second * 99)))
	Expect(percentile(values, 2)).To(Equal(time.Duration(time.Second * 99)))
}

func (s *UtilSuite) TestPercentilesLarge(t sweet.T) {
	values := []time.Duration{}
	for i := 0; i < 10000; i++ {
		values = append(values, time.Second*time.Duration(i))
	}

	for i := 0; i < 10000; i++ {
		Expect(percentile(values, float64(i)/10000)).To(Equal(values[i]))
	}

	Expect(percentile(values, 0)).To(Equal(time.Duration(0)))
	Expect(percentile(values, 1)).To(Equal(time.Duration(time.Second * 9999)))
	Expect(percentile(values, 2)).To(Equal(time.Duration(time.Second * 9999)))
}

func (s *UtilSuite) TestPercentilesZero(t sweet.T) {
	Expect(percentile(nil, 0.5)).To(Equal(time.Duration(0)))
}

func (s *UtilSuite) TestPercentilesFirst(t sweet.T) {
	Expect(percentile([]time.Duration{time.Second}, 0)).To(Equal(time.Second))
}

func (s *UtilSuite) TestRound(t sweet.T) {
	Expect(round(0.363636, 0.05)).To(BeNumerically("~", 0.35, 0.001))
	Expect(round(3.232, 0.05)).To(BeNumerically("~", 3.25, 0.001))
	Expect(round(0.4888, 0.05)).To(BeNumerically("~", 0.5, 0.001))
	Expect(round(-0.363636, 0.05)).To(BeNumerically("~", -0.35, 0.001))
	Expect(round(-3.232, 0.05)).To(BeNumerically("~", -3.25, 0.001))
	Expect(round(-0.4888, 0.05)).To(BeNumerically("~", -0.5, 0.001))
}
