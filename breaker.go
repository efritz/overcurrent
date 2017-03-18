package overcurrent

import (
	"fmt"
	"math/rand"
	"time"
)

type (
	// CircuitBreaker protects the invocation of a function and monitors failures.
	// After a certain failure threshold is reached, future invocations will instead
	// return an ErrErrCircuitOpen instead of attempting to invoke the function again.
	CircuitBreaker struct {
		config          *CircuitBreakerConfig
		clock           clock
		hardTrip        bool
		lastState       circuitState
		lastFailureTime *time.Time
		resetTimeout    *time.Duration
	}

	circuitState int
)

const (
	openState       circuitState = iota // Failure state
	closedState                         // Success state
	halfClosedState                     // Cautious, probabilistic retry state
)

var (
	// ErrCircuitOpen occurs when the Call method fails immediatley.
	ErrCircuitOpen = fmt.Errorf("circuit is open")

	// ErrInvocationTimeout occurs when the method invoked by Call
	// takes too long to execute.
	ErrInvocationTimeout = fmt.Errorf("invocation has timed out")
)

// NewCircuitBreaker creates a CircuitBreaker with the given configuration.
func NewCircuitBreaker(config *CircuitBreakerConfig) *CircuitBreaker {
	return newCircuitBreakerWithClock(config, &realClock{})
}

func newCircuitBreakerWithClock(config *CircuitBreakerConfig, clock clock) *CircuitBreaker {
	return &CircuitBreaker{
		config:    config,
		lastState: closedState,
		clock:     clock,
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

	err := cb.callWithTimeout(f)

	if err != nil && cb.config.FailureInterpreter.ShouldTrip(err) {
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
		if cb.clock.Now().Sub(*cb.lastFailureTime) >= *cb.resetTimeout {
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

	ch := make(chan error)
	go func() {
		defer close(ch)
		ch <- f()
	}()

	select {
	case err := <-ch:
		return err

	case <-cb.clock.After(cb.config.InvocationTimeout):
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
	now := cb.clock.Now()
	cb.lastFailureTime = &now
	cb.config.TripCondition.Failure()
}

func (cb *CircuitBreaker) updateBackoff() {
	reset := cb.config.ResetBackoff.NextInterval()
	cb.resetTimeout = &reset
}
