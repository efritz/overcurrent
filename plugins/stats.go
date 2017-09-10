package plugins

import (
	"sync"
	"time"

	"github.com/efritz/overcurrent"
)

// TODO - these need to roll over a longer period

type BreakerStats struct {
	config           overcurrent.BreakerConfig
	counters         map[overcurrent.EventType]int
	durations        map[overcurrent.EventType][]time.Duration
	semaphoreQueue   int
	semaphoreCurrent int
	semaphoreMax     int
	state            overcurrent.CircuitState
	mutex            *sync.RWMutex
}

func NewBreakerStats(config overcurrent.BreakerConfig) *BreakerStats {
	return &BreakerStats{
		config:    config,
		counters:  map[overcurrent.EventType]int{},
		durations: map[overcurrent.EventType][]time.Duration{},
		mutex:     &sync.RWMutex{},
	}
}

// TODO - make this less ad-hoc

func (s *BreakerStats) Increment(eventType overcurrent.EventType) {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	if eventType == overcurrent.EventTypeSemaphoreQueued {
		s.semaphoreQueue++
	} else if eventType == overcurrent.EventTypeSemaphoreDequeued {
		s.semaphoreQueue--
	} else if eventType == overcurrent.EventTypeSemaphoreAcquired {
		s.semaphoreCurrent++
		s.counters[eventType] = s.counters[eventType] + 1

		if s.semaphoreCurrent > s.semaphoreMax {
			s.semaphoreMax = s.semaphoreCurrent
		}
	} else if eventType == overcurrent.EventTypeSemaphoreReleased {
		s.semaphoreCurrent--
	} else {
		// TODO - update these things without a lock if possible
		s.counters[eventType] = s.counters[eventType] + 1
	}
}

func (s *BreakerStats) AddDuration(eventType overcurrent.EventType, duration time.Duration) {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	s.durations[eventType] = append(s.durations[eventType], duration)
}

func (s *BreakerStats) SetState(state overcurrent.CircuitState) {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	s.state = state
}

// TODO - don't return entire state decomposed, be sane about this

func (s *BreakerStats) GetAndReset() (overcurrent.CircuitState, int, int, int, map[overcurrent.EventType]int, map[overcurrent.EventType][]time.Duration) {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	var (
		counters     = s.counters
		durations    = s.durations
		semaphoreMax = s.semaphoreMax
	)

	s.counters = map[overcurrent.EventType]int{}
	s.durations = map[overcurrent.EventType][]time.Duration{}
	s.semaphoreMax = s.semaphoreCurrent

	return s.state, s.semaphoreQueue, s.semaphoreCurrent, semaphoreMax, counters, durations
}
