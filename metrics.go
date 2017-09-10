package overcurrent

import "time"

type (
	MetricCollector interface {
		Report(EventType)
		ReportDuration(EventType, time.Duration)
		ReportState(CircuitState)
	}

	NamedMetricCollector interface {
		Report(string, EventType)
		ReportDuration(string, EventType, time.Duration)
		ReportState(string, CircuitState)
	}

	namedCollector struct {
		name      string
		collector NamedMetricCollector
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

func (c *noopCollector) Report(eventType EventType)                                 {}
func (c *noopCollector) ReportDuration(eventType EventType, duration time.Duration) {}
func (c *noopCollector) ReportState(state CircuitState)                             {}

func NamedCollector(name string, collector NamedMetricCollector) MetricCollector {
	return &namedCollector{
		name:      name,
		collector: collector,
	}
}

func (c *namedCollector) Report(eventType EventType) {
	c.collector.Report(c.name, eventType)
}

func (c *namedCollector) ReportDuration(eventType EventType, duration time.Duration) {
	c.collector.ReportDuration(c.name, eventType, duration)
}

func (c *namedCollector) ReportState(state CircuitState) {
	c.collector.ReportState(c.name, state)
}
