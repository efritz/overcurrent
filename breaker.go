package overcurrent

import (
	"errors"
	"math/rand"
	"time"
)

type CircuitState int
type CircuitEvent int

const (
	OpenState CircuitState = iota
	ClosedState
	HalfClosedState

	TripEvent CircuitEvent = iota
	ResetEvent
)

var (
	CircuitOpenError       = errors.New("Circuit is open.")
	InvocationTimeoutError = errors.New("Invocation has timed out.")
)

//
//

// A CircuitBreaker protects the invocation of a function and monitors failures.
// After a certain failure threshold is reached, future invocations will instead
// return a CircuitOpenError instead of attempting to invoke the function again.
type CircuitBreaker struct {
	config BreakerConfig

	// A channel on which trip and reset events are sent. These can be monitored
	// for logging purposes, or safely ignored.
	Events chan CircuitEvent

	hardTrip        bool
	lastFailureTime *time.Time
}

func NewBreaker(config BreakerConfig) *CircuitBreaker {
	return &CircuitBreaker{
		config: config,
		Events: make(chan CircuitEvent),
	}
}

func (cb *CircuitBreaker) State() CircuitState {
	if cb.config.TripCondition.ShouldTrip() {
		if time.Now().Sub(*cb.lastFailureTime) > time.Duration(cb.config.ResetTimeout) {
			return HalfClosedState
		} else {
			return OpenState
		}
	} else {
		return ClosedState
	}
}

// Attempt to call the given function if the circuit breaker is closed, or if
// the circuit breaker is half-closed (with some probability). Otherwise, return
// a CircuitOpenError. If the function times out, the circuit breaker will fail
// with a InvocationTimeoutError. If the function is invoked and yields an error
// value, that value is returned.
func (cb *CircuitBreaker) Call(f func() error) error {
	if !cb.shouldTry() {
		return CircuitOpenError
	}

	if err := cb.callWithTimeout(f); err != nil && cb.config.FailureInterpreter.ShouldTrip(err) {
		cb.recordError()
		return err
	}

	cb.Reset()
	return nil
}

func (cb *CircuitBreaker) shouldTry() bool {
	if cb.hardTrip || cb.State() == OpenState {
		return false
	}

	if cb.State() == HalfClosedState {
		return rand.Float64() <= cb.config.HalfClosedRetryProbability
	}

	return true
}

func (cb *CircuitBreaker) callWithTimeout(f func() error) error {
	if cb.config.InvocationTimeout == 0 {
		return f()
	}

	c := make(chan error)
	go func() {
		c <- f()
		close(c)
	}()

	select {
	case err := <-c:
		return err
	case <-time.After(time.Duration(cb.config.InvocationTimeout)):
		return InvocationTimeoutError
	}
}

func (cb *CircuitBreaker) recordError() {
	now := time.Now()
	cb.lastFailureTime = &now

	cb.config.TripCondition.Failure()
	cb.sendEvent(TripEvent)
}

func (cb *CircuitBreaker) sendEvent(event CircuitEvent) {
	select {
	case cb.Events <- event:
	}
}

func (cb *CircuitBreaker) Trip() {
	cb.hardTrip = true
	cb.recordError()
}

func (cb *CircuitBreaker) Reset() {
	cb.lastFailureTime = nil
	cb.hardTrip = false

	cb.config.TripCondition.Success()
	cb.sendEvent(ResetEvent)
}
