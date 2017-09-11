package overcurrent

import (
	"time"

	"github.com/efritz/glock"
)

type semaphore struct {
	clock glock.Clock
	ch    chan struct{}
}

func newSemaphore(clock glock.Clock, capacity int) *semaphore {
	s := &semaphore{
		clock: clock,
		ch:    make(chan struct{}, capacity),
	}

	for i := 0; i < capacity; i++ {
		s.signal()
	}

	return s
}

func (s *semaphore) wait(timeout time.Duration, collector MetricCollector) bool {
	select {
	case <-s.ch:
		return true
	default:
	}

	if timeout == 0 {
		return false
	}

	collector.ReportCount(EventTypeSemaphoreQueued)
	defer collector.ReportCount(EventTypeSemaphoreDequeued)

	select {
	case <-s.ch:
		return true

	case <-s.clock.After(timeout):
		return false
	}
}

func (s *semaphore) signal() {
	s.ch <- struct{}{}
}
