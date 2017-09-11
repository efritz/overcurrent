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

	FailureInterpreterFunc func(error) bool
)

func (f FailureInterpreterFunc) ShouldTrip(err error) bool {
	return f(err)
}

// NewAnyErrorFailureInterpreter creates a failure interpreter that trips
// on every error.
func NewAnyErrorFailureInterpreter() FailureInterpreter {
	return FailureInterpreterFunc(func(error) bool {
		return true
	})
}
