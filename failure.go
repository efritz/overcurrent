package overcurrent

// FailureInterpreter is the interface that determines if an error should
// affect whether or not the circuit breaker will trip. This is useful if
// some errors are indictive of a system failure and others are not (e.g.
// HTTP 500 vs HTTP 400 responses).
type FailureInterpreter interface {
	ShouldTrip(error) bool
}

//
//

type AnyErrorFailureInterpreter struct{}

func (fi *AnyErrorFailureInterpreter) ShouldTrip(err error) bool {
	return true
}

// A failure interpreter that trips on every error.
func NewAnyErrorFailureInterpreter() *AnyErrorFailureInterpreter {
	return &AnyErrorFailureInterpreter{}
}
