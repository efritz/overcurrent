package overcurrent

import "time"

type (
	MetricCollector interface {
		Report(EventType)
		ReportDuration(EventType, time.Duration)
	}

	NamedMetricCollector interface {
		Report(string, EventType)
		ReportDuration(string, EventType, time.Duration)
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
	EventTypeShortCircuit
	EventTypeTimeout
	EventTypeError
	EventTypeSuccess
	EventTypeFailure
	EventTypeRejection
	EventTypeFallbackSuccess
	EventTypeFallbackFailure
	EventTypeRunDuration
	EventTypeTotalDuration
)

var defaultCollector = &noopCollector{}

func (c *noopCollector) Report(eventType EventType)                                 {}
func (c *noopCollector) ReportDuration(eventType EventType, duration time.Duration) {}

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
