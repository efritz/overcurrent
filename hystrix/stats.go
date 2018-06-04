package hystrix

import (
	"sort"
	"sync"
	"time"

	"github.com/efritz/glock"
	"github.com/efritz/overcurrent"
)

type (
	BreakerStats struct {
		config  overcurrent.BreakerConfig
		state   overcurrent.CircuitState
		buckets map[int64]*bucket
		mutex   sync.RWMutex
		clock   glock.Clock
	}

	bucket struct {
		counters  map[overcurrent.EventType]int
		durations map[overcurrent.EventType][]time.Duration
		currents  map[overcurrent.EventType]int
		maximums  map[overcurrent.EventType]int
	}

	FrozenBreakerStats struct {
		config    overcurrent.BreakerConfig
		state     overcurrent.CircuitState
		counters  map[overcurrent.EventType]int
		durations map[overcurrent.EventType][]time.Duration
		currents  map[overcurrent.EventType]int
		maximums  map[overcurrent.EventType]int
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
	return newBreakerStatsWithClock(config, glock.NewRealClock())
}

func newBreakerStatsWithClock(config overcurrent.BreakerConfig, clock glock.Clock) *BreakerStats {
	return &BreakerStats{
		config:  config,
		buckets: map[int64]*bucket{},
		clock:   clock,
	}
}

func (s *BreakerStats) SetState(state overcurrent.CircuitState) {
	s.mutex.Lock()
	s.state = state
	s.mutex.Unlock()
}

func (s *BreakerStats) Increment(eventType overcurrent.EventType) {
	var (
		dualType = eventType
		delta    = 1
	)

	if dual, ok := pairs[eventType]; ok {
		dualType = dual.eventType
		delta = dual.delta
	}

	s.mutex.Lock()
	s.getCurrentBucket().increment(eventType, dualType, delta)
	s.mutex.Unlock()
}

func (s *BreakerStats) AddDuration(eventType overcurrent.EventType, duration time.Duration) {
	s.mutex.Lock()
	s.getCurrentBucket().addDuration(eventType, duration)
	s.mutex.Unlock()
}

func (s *BreakerStats) getCurrentBucket() *bucket {
	now := s.clock.Now().Unix()

	if bucket, ok := s.buckets[now]; ok {
		return bucket
	}

	var (
		ts       = s.getBucketTimestamps()
		currents = map[overcurrent.EventType]int{}
		maximums = map[overcurrent.EventType]int{}
	)

	// If we need to create a new bucket, transfer the current
	// counts from the last active bucket to this one. The max
	// of this bucket will be the _current_ count, as that's
	// the only number we've seen.

	if len(ts) > 0 {
		previous := s.buckets[ts[len(ts)-1]]
		currents = cloneMap(previous.currents)
		maximums = cloneMap(previous.currents)
	}

	bucket := &bucket{
		counters:  map[overcurrent.EventType]int{},
		durations: map[overcurrent.EventType][]time.Duration{},
		currents:  currents,
		maximums:  maximums,
	}

	s.buckets[now] = bucket
	return bucket
}

func (s *BreakerStats) Freeze() *FrozenBreakerStats {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	var (
		counters  = map[overcurrent.EventType]int{}
		durations = map[overcurrent.EventType][]time.Duration{}
		currents  = map[overcurrent.EventType]int{}
		maximums  = map[overcurrent.EventType]int{}
	)

	// Ensure that at least one valid bucket exists at the
	// time we emit stats. That will ensure that all of the
	// semaphore queue/acquire current counts did not just
	// fall away.
	s.getCurrentBucket()

	for _, ts := range s.getBucketTimestamps() {
		bucket := s.buckets[ts]

		for k, v := range bucket.counters {
			counters[k] = counters[k] + v
		}

		for k, v := range bucket.durations {
			durations[k] = append(durations[k], v...)
		}

		for k, v := range bucket.currents {
			currents[k] = v
		}

		for k, v := range bucket.maximums {
			maximums[k] = max(maximums[k], v)
		}
	}

	return &FrozenBreakerStats{
		config:    s.config,
		state:     s.state,
		counters:  counters,
		durations: sortDurationMap(durations),
		currents:  currents,
		maximums:  maximums,
	}
}

func (s *BreakerStats) getBucketTimestamps() []int64 {
	var (
		order  = []int64{}
		expiry = s.clock.Now().Unix() - 10
	)

	for ts := range s.buckets {
		order = append(order, ts)
	}

	sort.Slice(order, func(a, b int) bool {
		return order[a] < order[b]
	})

	for len(order) > 1 && order[0] <= expiry {
		delete(s.buckets, order[0])
		order = order[1:]
	}

	for len(order) > 0 && order[0] <= expiry {
		order = order[1:]
	}

	return order
}

func (b *bucket) increment(typeA, typeB overcurrent.EventType, delta int) {
	b.counters[typeA] = b.counters[typeA] + 1
	b.currents[typeB] = b.currents[typeB] + delta
	b.maximums[typeB] = max(b.maximums[typeB], b.currents[typeB])
}

func (b *bucket) addDuration(eventType overcurrent.EventType, duration time.Duration) {
	b.durations[eventType] = append(b.durations[eventType], duration)
}
