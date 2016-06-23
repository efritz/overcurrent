package overcurrent

import (
	"errors"
	"math/rand"
	"time"

	"github.com/efritz/backoff"
)

// Backoff is the interface to a backoff interval generator. See the
// backoff dependency for details.
type Backoff backoff.Backoff

type circuitState int

const (
	openState       circuitState = iota // Failure state
	closedState                         // Success state
	halfClosedState                     // Cautious, probabilistic retry state
)

var (
	// ErrCircuitOpen occurs when the Call method fails immediatley.
	ErrCircuitOpen = errors.New("Circuit is open.")

	// ErrInvocationTimeout occurs when the method invoked by Call
	// takes too long to execute.
	ErrInvocationTimeout = errors.New("Invocation has timed out.")
)

//
//

// CircuitBreaker protects the invocation of a function and monitors failures.
// After a certain failure threshold is reached, future invocations will instead
// return an ErrErrCircuitOpen instead of attempting to invoke the function again.
type CircuitBreaker struct {
	config          *CircuitBreakerConfig
	hardTrip        bool
	lastState       circuitState
	lastFailureTime *time.Time
	resetTimeout    *time.Duration
}

// NewCircuitBreaker creates a CircuitBreaker with the given configuration.
func NewCircuitBreaker(config *CircuitBreakerConfig) *CircuitBreaker {
	return &CircuitBreaker{
		config:    config,
		lastState: closedState,
	}
}

// Call attempts to call the given function if the circuit breaker is closed, or if
// the circuit breaker is half-closed (with some probability). Otherwise, return an
// ErrCircuitOpen. If the function times out, the circuit breaker will fail with an
// ErrInvocationTimeout. If the function is invoked and yields an error value, that
// value is returned.
func (cb *CircuitBreaker) Call(f func() error) error {
	if !cb.shouldTry() {
		return ErrCircuitOpen
	}

	if err := cb.callWithTimeout(f); err != nil && cb.config.FailureInterpreter.ShouldTrip(err) {
		cb.recordFailure()
		return err
	}

	cb.Reset()
	return nil
}

// Trip manually trips the circuit breaker. The circuit breaker will remain open
// until it is manually reset.
func (cb *CircuitBreaker) Trip() {
	cb.hardTrip = true
	cb.recordFailure()
}

// Reset the circuit breaker.
func (cb *CircuitBreaker) Reset() {
	cb.recordSuccess()
}

//
//

func (cb *CircuitBreaker) shouldTry() bool {
	if cb.hardTrip || cb.state() == openState {
		return false
	}

	if cb.state() == halfClosedState {
		return rand.Float64() <= cb.config.HalfClosedRetryProbability
	}

	return true
}

func (cb *CircuitBreaker) state() circuitState {
	if !cb.config.TripCondition.ShouldTrip() {
		cb.lastState = closedState
		return cb.lastState
	}

	if cb.lastState == closedState {
		cb.config.ResetBackoff.Reset()
	}

	if cb.lastState != openState {
		cb.updateBackoff()
	}

	if cb.lastFailureTime != nil {
		if time.Now().Sub(*cb.lastFailureTime) >= *cb.resetTimeout {
			cb.lastState = halfClosedState
			return halfClosedState
		}
	}

	cb.lastState = openState
	return openState
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
	case <-time.After(cb.config.InvocationTimeout):
		return ErrInvocationTimeout
	}
}

func (cb *CircuitBreaker) recordSuccess() {
	cb.hardTrip = false
	cb.lastFailureTime = nil
	cb.resetTimeout = nil

	cb.config.ResetBackoff.Reset()
	cb.config.TripCondition.Success()
}

func (cb *CircuitBreaker) recordFailure() {
	now := time.Now()
	cb.lastFailureTime = &now
	cb.config.TripCondition.Failure()
}

func (cb *CircuitBreaker) updateBackoff() {
	reset := cb.config.ResetBackoff.NextInterval()
	cb.resetTimeout = &reset
}
