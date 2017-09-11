package overcurrent

import (
	"context"
	"time"
)

type (
	// NoopBreaker is a circuit breaker that never trips.
	NoopBreaker struct{}

	// NoopCollector is a metric collector that does nothing.
	NoopCollector struct{}
)

var defaultCollector = NewNoopCollector()

// NewNoopBreaker creates a new non-tripping breaker.
func NewNoopBreaker() CircuitBreaker {
	return &NoopBreaker{}
}

func (b *NoopBreaker) Trip()                                {}
func (b *NoopBreaker) Reset()                               {}
func (b *NoopBreaker) ShouldTry() bool                      { return true }
func (b *NoopBreaker) MarkResult(err error) bool            { return true }
func (b *NoopBreaker) Call(f BreakerFunc) error             { return f(context.Background()) }
func (b *NoopBreaker) CallAsync(f BreakerFunc) <-chan error { return nil }

// NewNoopCollector creates a new do-nothing collector.
func NewNoopCollector() MetricCollector {
	return &NoopCollector{}
}

func (c *NoopCollector) ReportNew(BreakerConfig)                 {}
func (c *NoopCollector) ReportCount(EventType)                   {}
func (c *NoopCollector) ReportDuration(EventType, time.Duration) {}
func (c *NoopCollector) ReportState(CircuitState)                {}
