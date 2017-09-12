package overcurrent

import "time"

type (
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

	namedCollector struct {
		name      string
		collector NamedMetricCollector
	}
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
