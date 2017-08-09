package overcurrent

// NoopBreaker is a circuit breaker that never trips.
type NoopBreaker struct{}

// NewNoopBreaker creates a new non-tripping breaker.
func NewNoopBreaker() CircuitBreaker {
	return &NoopBreaker{}
}

func (b *NoopBreaker) Trip()                     {}
func (b *NoopBreaker) Reset()                    {}
func (b *NoopBreaker) ShouldTry() bool           { return true }
func (b *NoopBreaker) MarkResult(err error) bool { return true }
func (b *NoopBreaker) Call(f func() error) error { return f() }
