package plugins

import (
	"time"

	"github.com/aphistic/sweet"
	"github.com/efritz/glock"
	"github.com/efritz/overcurrent"
	. "github.com/onsi/gomega"
)

type StatsSuite struct{}

var testConfig = overcurrent.BreakerConfig{
	MaxConcurrency: 50,
}

func (s *StatsSuite) TestConfig(t sweet.T) {
	stats := NewBreakerStats(testConfig)
	Expect(stats.config.MaxConcurrency).To(Equal(50))
}

func (s *StatsSuite) TestState(t sweet.T) {
	stats := NewBreakerStats(testConfig)
	stats.SetState(overcurrent.StateHalfClosed)
	Expect(stats.state).To(Equal(overcurrent.StateHalfClosed))
}

func (s *StatsSuite) TestIncrement(t sweet.T) {
	clock := glock.NewMockClock()
	stats := newBreakerStatsWithClock(testConfig, clock)

	for j := 0; j < 30; j++ {
		clock.Advance(time.Second)

		for i := 0; i < 20; i++ {
			stats.Increment(overcurrent.EventTypeSuccess)
		}
	}

	// Should be 200, not 600 due to pruning the first 20 (of 30) seconds
	Expect(stats.Freeze().counters[overcurrent.EventTypeSuccess]).To(Equal(200))
}

func (s *StatsSuite) TestIncrementDual(t sweet.T) {
	clock := glock.NewMockClock()
	stats := newBreakerStatsWithClock(testConfig, clock)

	for _, pair := range [][]int{[]int{10, 5}, []int{50, 20}, []int{10, 30}} {
		for i := 0; i < pair[0]; i++ {
			stats.Increment(overcurrent.EventTypeSemaphoreAcquired)
		}

		for i := 0; i < pair[1]; i++ {
			stats.Increment(overcurrent.EventTypeSemaphoreReleased)
		}

		clock.Advance(time.Second)
	}

	frozen1 := stats.Freeze()
	Expect(frozen1.currents[overcurrent.EventTypeSemaphoreAcquired]).To(Equal(15))
	Expect(frozen1.maximums[overcurrent.EventTypeSemaphoreAcquired]).To(Equal(55))
	Expect(frozen1.counters[overcurrent.EventTypeSemaphoreAcquired]).To(Equal(70))
	Expect(frozen1.counters[overcurrent.EventTypeSemaphoreReleased]).To(Equal(55))

	//
	// Test behavior of expiring buckets
	//

	clock.Advance(time.Second * 7)
	frozen2 := stats.Freeze()
	Expect(frozen2.currents[overcurrent.EventTypeSemaphoreAcquired]).To(Equal(15))
	Expect(frozen2.maximums[overcurrent.EventTypeSemaphoreAcquired]).To(Equal(55))
	Expect(frozen2.counters[overcurrent.EventTypeSemaphoreAcquired]).To(Equal(60))
	Expect(frozen2.counters[overcurrent.EventTypeSemaphoreReleased]).To(Equal(50))

	clock.Advance(time.Second * 1)
	frozen3 := stats.Freeze()
	Expect(frozen3.currents[overcurrent.EventTypeSemaphoreAcquired]).To(Equal(15))
	Expect(frozen3.maximums[overcurrent.EventTypeSemaphoreAcquired]).To(Equal(45))
	Expect(frozen3.counters[overcurrent.EventTypeSemaphoreAcquired]).To(Equal(10))
	Expect(frozen3.counters[overcurrent.EventTypeSemaphoreReleased]).To(Equal(30))

	clock.Advance(time.Second * 1)
	frozen4 := stats.Freeze()
	Expect(frozen4.currents[overcurrent.EventTypeSemaphoreAcquired]).To(Equal(15))
	Expect(frozen4.maximums[overcurrent.EventTypeSemaphoreAcquired]).To(Equal(15))
	Expect(frozen4.counters[overcurrent.EventTypeSemaphoreAcquired]).To(Equal(0))
	Expect(frozen4.counters[overcurrent.EventTypeSemaphoreReleased]).To(Equal(0))
}

func (s *StatsSuite) TestAddDuration(t sweet.T) {
	// TODO
}
