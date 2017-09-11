package plugins

import (
	"sync"
	"time"

	"github.com/efritz/overcurrent"
)

// TODO - these need to roll over a longer period

type (
	BreakerStats struct {
		config    overcurrent.BreakerConfig
		counters  map[overcurrent.EventType]int
		durations map[overcurrent.EventType][]time.Duration
		currents  map[overcurrent.EventType]int
		maximums  map[overcurrent.EventType]int
		state     overcurrent.CircuitState
		mutex     *sync.RWMutex
	}

	dualStatRelation struct {
		eventType overcurrent.EventType
		delta     int
	}
)

var pairs = map[overcurrent.EventType]dualStatRelation{
	overcurrent.EventTypeSemaphoreQueued:   dualStatRelation{overcurrent.EventTypeSemaphoreQueued, +1},
	overcurrent.EventTypeSemaphoreDequeued: dualStatRelation{overcurrent.EventTypeSemaphoreQueued, -1},
	overcurrent.EventTypeSemaphoreAcquired: dualStatRelation{overcurrent.EventTypeSemaphoreAcquired, +1},
	overcurrent.EventTypeSemaphoreReleased: dualStatRelation{overcurrent.EventTypeSemaphoreAcquired, -1},
}

func NewBreakerStats(config overcurrent.BreakerConfig) *BreakerStats {
	return &BreakerStats{
		config:    config,
		currents:  map[overcurrent.EventType]int{},
		durations: map[overcurrent.EventType][]time.Duration{},
		counters:  map[overcurrent.EventType]int{},
		maximums:  map[overcurrent.EventType]int{},
		mutex:     &sync.RWMutex{},
	}
}

func (s *BreakerStats) Increment(eventType overcurrent.EventType) {
	delta := 1
	if dual, ok := pairs[eventType]; ok {
		eventType = dual.eventType
		delta = dual.delta
	}

	s.mutex.Lock()
	defer s.mutex.Unlock()

	s.counters[eventType] = s.counters[eventType] + 1
	s.currents[eventType] = s.currents[eventType] + delta

	if s.currents[eventType] > s.maximums[eventType] {
		s.maximums[eventType] = s.currents[eventType]
	}
}

func (s *BreakerStats) AddDuration(eventType overcurrent.EventType, duration time.Duration) {
	s.mutex.Lock()
	s.durations[eventType] = append(s.durations[eventType], duration)
	s.mutex.Unlock()
}

func (s *BreakerStats) SetState(state overcurrent.CircuitState) {
	s.mutex.Lock()
	s.state = state
	s.mutex.Unlock()
}

func (s *BreakerStats) Freeze() *BreakerStats {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	clone := &BreakerStats{
		config:    s.config,
		counters:  s.counters,
		durations: s.durations,
		currents:  s.currents,
		maximums:  s.maximums,
	}

	s.counters = map[overcurrent.EventType]int{}
	s.durations = map[overcurrent.EventType][]time.Duration{}
	s.currents = cloneEventMap(s.currents)
	s.maximums = cloneEventMap(s.currents)
	return clone
}

func cloneEventMap(values map[overcurrent.EventType]int) map[overcurrent.EventType]int {
	cloned := map[overcurrent.EventType]int{}
	for k, v := range values {
		cloned[k] = v
	}

	return cloned
}
