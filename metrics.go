package overcurrent

import "time"

type (
	MetricCollector interface {
		// ReportNew fires when a breaker is first initialized so the collector
		// can track the immutable config of circuit breakers.
		ReportNew(BreakerConfig)

		// ReportCount fires each time a non-latency event is emitted.
		ReportCount(EventType)

		// ReportDuration fires on latency events with the time spent inside a
		// code path of-interest (inside a breaker func or inside a Call func).
		ReportDuration(EventType, time.Duration)

		// ReportState fires when a breaker changes state.
		ReportState(CircuitState)
	}

	NamedMetricCollector interface {
		// ReportNew is MetricCollector.ReportNew with the name of the breaker
		// passed in as a first argument.
		ReportNew(string, BreakerConfig)

		// ReportCount is MetricCollector.ReportCount with the name of the breaker
		// passed in as a first argument.
		ReportCount(string, EventType)

		// ReportDuration is MetricCollector.ReportDuration with the name of the breaker
		// passed in as a first argument.
		ReportDuration(string, EventType, time.Duration)

		// ReportState is MetricCollector.ReportState with the name of the breaker
		// passed in as a first argument.
		ReportState(string, CircuitState)
	}

	// BreakerConfig is a struct that contains a copy of some of a breaker's
	// initialization values. This struct may grow as metric collectors track
	// additional breaker state.
	BreakerConfig struct {
		MaxConcurrency int
	}

	namedCollector struct {
		name      string
		collector NamedMetricCollector
	}

	// EventType distinguishes interesting occurrences
	EventType int
)

const (
	// EventTypeAttempt occurs when Call or CallAsync is called.
	EventTypeAttempt EventType = iota

	// EventTypeSuccess occurs when a breaker func returns a nil error
	EventTypeSuccess

	// EventTypeFailure occurs when a breaker func returns a non-nil error
	// or cannot be called due to breaker status or semaphore contention.
	EventTypeFailure

	// EventTypeError occurs when a breaker func returns a non-nil error
	// which is interpreted as an error against the breaker.
	EventTypeError

	// EventTypeBadRequest occurs when a breaker func returns a non-nil
	// error which is not interpreted as an error against the breaker.
	EventTypeBadRequest

	// EventTypeShortCircuit occurs when a circuit is open and no breaker
	// func is invoked
	EventTypeShortCircuit

	// EventTypeTimeout occurs when execution of a breaker func times out
	EventTypeTimeout

	// EventTypeRejection occurs when a breaker func cannot be invoked due
	// to semaphore contention.
	EventTypeRejection

	// EventTypeFallbackSuccess occurs when a fallback func returns a nil
	// error
	EventTypeFallbackSuccess

	// EventTypeFallbackFailure occurs when a fallback func returns a non-nil
	// error
	EventTypeFallbackFailure

	// EventTypeRunDuration marks the duration of a breaker func invocation.
	EventTypeRunDuration

	// EventTypeTotalDuration marks the duration of a Call or CallAsync method.
	EventTypeTotalDuration

	// EventTypeSemaphoreQueued occurs once a routine begins waiting for a
	// semaphore token. This event does not occur if a token is immediately
	// available.
	EventTypeSemaphoreQueued

	// EventTypeSemaphoreDequeued occurs once a routine stops waiting for a
	// semaphore token. This could be because the routine got a successful token,
	// or because the max timeout has elapsed.
	EventTypeSemaphoreDequeued

	// EventTypeSemaphoreAcquired occurs once a semaphore token is acquired and
	// the breaker func can be invoked.
	EventTypeSemaphoreAcquired

	// EventTypeSemaphoreReleased occurs after the breaker func is invoked.
	EventTypeSemaphoreReleased
)

// NamedCollector converts a named metric collector into a metric collector. The
// name given to this constructor will be sent as the first argument to all of the
// named metric collector methods.
func NamedCollector(name string, collector NamedMetricCollector) MetricCollector {
	return &namedCollector{
		name:      name,
		collector: collector,
	}
}

func (c *namedCollector) ReportNew(config BreakerConfig) {
	c.collector.ReportNew(c.name, config)
}

func (c *namedCollector) ReportCount(eventType EventType) {
	c.collector.ReportCount(c.name, eventType)
}

func (c *namedCollector) ReportDuration(eventType EventType, duration time.Duration) {
	c.collector.ReportDuration(c.name, eventType, duration)
}

func (c *namedCollector) ReportState(state CircuitState) {
	c.collector.ReportState(c.name, state)
}
