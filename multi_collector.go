package overcurrent

import "time"

// MultiCollector is a metric collector that wraps several other collector
// instances. Each event will be passed to every backend in the registered
// order.
type MultiCollector struct {
	collectors []MetricCollector
}

// NewMultiCollector creates a new MultiCollector.
func NewMultiCollector(collectors ...MetricCollector) MetricCollector {
	return &MultiCollector{
		collectors: collectors,
	}
}

func (c *MultiCollector) ReportNew(config BreakerConfig) {
	for _, collector := range c.collectors {
		collector.ReportNew(config)
	}
}

func (c *MultiCollector) ReportCount(eventType EventType) {
	for _, collector := range c.collectors {
		collector.ReportCount(eventType)
	}
}

func (c *MultiCollector) ReportDuration(eventType EventType, duration time.Duration) {
	for _, collector := range c.collectors {
		collector.ReportDuration(eventType, duration)
	}
}

func (c *MultiCollector) ReportState(state CircuitState) {
	for _, collector := range c.collectors {
		collector.ReportState(state)
	}
}
