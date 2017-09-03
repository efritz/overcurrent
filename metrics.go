package overcurrent

type (
	MetricCollector interface {
		Report(EventType)
		ReportDuration(EventType, uint32)
	}

	NoopCollector struct{}

	EventType int
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

func (c *NoopCollector) Report(eventType EventType)                          {}
func (c *NoopCollector) ReportDuration(eventType EventType, duration uint32) {}
