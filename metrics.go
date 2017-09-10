package overcurrent

import "time"

type (
	MetricCollector interface {
		ReportNew(BreakerConfig)
		ReportCount(EventType)
		ReportDuration(EventType, time.Duration)
		ReportState(CircuitState)
	}

	NamedMetricCollector interface {
		ReportNew(string, BreakerConfig)
		ReportCount(string, EventType)
		ReportDuration(string, EventType, time.Duration)
		ReportState(string, CircuitState)
	}

	namedCollector struct {
		name      string
		collector NamedMetricCollector
	}

	BreakerConfig struct {
		MaxConcurrency int
	}

	EventType     int
	noopCollector struct{}
)

const (
	EventTypeAttempt EventType = iota
	EventTypeSuccess
	EventTypeFailure
	EventTypeError
	EventTypeBadRequest
	EventTypeShortCircuit
	EventTypeTimeout
	EventTypeRejection
	EventTypeFallbackSuccess
	EventTypeFallbackFailure
	EventTypeRunDuration
	EventTypeTotalDuration
	EventTypeSemaphoreQueued
	EventTypeSemaphoreDequeued
	EventTypeSemaphoreAcquired
	EventTypeSemaphoreReleased
)

var defaultCollector = &noopCollector{}

func (c *noopCollector) ReportNew(BreakerConfig)                 {}
func (c *noopCollector) ReportCount(EventType)                   {}
func (c *noopCollector) ReportDuration(EventType, time.Duration) {}
func (c *noopCollector) ReportState(CircuitState)                {}

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
