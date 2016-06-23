package overcurrent

// FailureInterpreter is the interface that determines if an error should
// affect whether or not the circuit breaker will trip. This is useful if
// some errors are indictive of a system failure and others are not (e.g.
// HTTP 500 vs HTTP 400 responses).
type FailureInterpreter interface {
	// ShouldTrip determines if this error should cause a CircuitBreaker
	// to transition from an open to a closed state.
	ShouldTrip(error) bool
}

//
//

// AnyErrorFailureInterpreter is a failure interpreter that trips on every
// error, regardless of type or properties.
type AnyErrorFailureInterpreter struct{}

// ShouldTrip returns true for all errors.
func (fi *AnyErrorFailureInterpreter) ShouldTrip(err error) bool {
	return true
}

// NewAnyErrorFailureInterpreter creates an AnyErrorFailureInterpreter.
func NewAnyErrorFailureInterpreter() *AnyErrorFailureInterpreter {
	return &AnyErrorFailureInterpreter{}
}
