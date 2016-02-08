package overcurrent

type FailureInterpreter interface {
	ShouldTrip(error) bool
}

//
//

type AnyErrorFailureInterpreter struct{}

func (fi *AnyErrorFailureInterpreter) ShouldTrip(err error) bool {
	return true
}

func NewAnyErrorFailureInterpreter() *AnyErrorFailureInterpreter {
	return &AnyErrorFailureInterpreter{}
}
