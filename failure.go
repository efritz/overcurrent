package overcurrent

type (
	// FailureInterpreter is the interface that determines if an error should
	// affect whether or not the circuit breaker will trip. This is useful if
	// some errors are indictive of a system failure and others are not (e.g.
	// HTTP 500 vs HTTP 400 responses).
	FailureInterpreter interface {
		// ShouldTrip determines if this error should cause a CircuitBreaker
		// to transition from an open to a closed state.
		ShouldTrip(error) bool
	}

	// AnyErrorFailureInterpreter trips on every error.
	AnyErrorFailureInterpreter struct{}
)

// NewAnyErrorFailureInterpreter creates an AnyErrorFailureInterpreter.
func NewAnyErrorFailureInterpreter() FailureInterpreter {
	return &AnyErrorFailureInterpreter{}
}

func (fi *AnyErrorFailureInterpreter) ShouldTrip(err error) bool {
	return true
}
