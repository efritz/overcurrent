package overcurrent

import (
	"fmt"
	"math/rand"
	"time"

	"github.com/efritz/glock"
)

type (
	// CircuitBreaker protects the invocation of a function and monitors failures.
	// After a certain failure threshold is reached, future invocations will instead
	// return an ErrErrCircuitOpen instead of attempting to invoke the function again.
	CircuitBreaker struct {
		config          *CircuitBreakerConfig
		clock           glock.Clock
		state           circuitState
		hardTrip        bool
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
	// ErrCircuitOpen occurs when the Call method fails immediately.
	ErrCircuitOpen = fmt.Errorf("circuit is open")
)

// NewCircuitBreaker creates a CircuitBreaker with the given configuration.
func NewCircuitBreaker(config *CircuitBreakerConfig) *CircuitBreaker {
	return newCircuitBreakerWithClock(config, glock.NewRealClock())
}

func newCircuitBreakerWithClock(config *CircuitBreakerConfig, clock glock.Clock) *CircuitBreaker {
	return &CircuitBreaker{
		config: config,
		clock:  clock,
		state:  closedState,
	}
}

// Trip manually trips the circuit breaker. The circuit breaker will remain open
// until it is manually reset.
func (cb *CircuitBreaker) Trip() {
	cb.hardTrip = true
}

// Reset the circuit breaker.
func (cb *CircuitBreaker) Reset() {
	cb.hardTrip = false
	cb.lastFailureTime = nil
	cb.resetTimeout = nil

	cb.config.ResetBackoff.Reset()
	cb.config.TripCondition.Success()
}

// ShouldTry returns true if the circuit breaker is closed or half-closed with
// some probability. Successive calls to this method may yield different results
// depending on the registered trip condition.
func (cb *CircuitBreaker) ShouldTry() bool {
	if cb.hardTrip {
		return false
	}

	if !cb.config.TripCondition.ShouldTrip() {
		cb.state = closedState
		return true
	}

	if cb.state == closedState {
		cb.config.ResetBackoff.Reset()
	}

	if cb.state != openState {
		reset := cb.config.ResetBackoff.NextInterval()
		cb.resetTimeout = &reset
	}

	if cb.lastFailureTime != nil {
		if cb.clock.Now().Sub(*cb.lastFailureTime) >= *cb.resetTimeout {
			cb.state = halfClosedState
			return rand.Float64() < cb.config.HalfClosedRetryProbability
		}
	}

	cb.state = openState
	return false
}

// MarkResult takes the result of the protected section and marks it as a success if
// the error is nil or if the failure interpreter decides not to trip on this error.
func (cb *CircuitBreaker) MarkResult(err error) bool {
	if err != nil && cb.config.FailureInterpreter.ShouldTrip(err) {
		now := cb.clock.Now()
		cb.lastFailureTime = &now
		cb.config.TripCondition.Failure()
		return false
	}

	cb.Reset()
	return true
}

// Call attempts to call the given function if the circuit breaker is closed, or if
// the circuit breaker is half-closed (with some probability). Otherwise, return an
// ErrCircuitOpen. If the function times out, the circuit breaker will fail with an
// ErrInvocationTimeout. If the function is invoked and yields a value before the
// timeout elapses, that value is returned.
func (cb *CircuitBreaker) Call(f func() error) error {
	if !cb.ShouldTry() {
		return ErrCircuitOpen
	}

	if err := callWithTimeout(f, cb.clock, cb.config.InvocationTimeout); !cb.MarkResult(err) {
		return err
	}

	return nil
}
