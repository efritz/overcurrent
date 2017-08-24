package overcurrent

import (
	"context"
	"fmt"
	"math/rand"
	"time"

	"github.com/efritz/backoff"
	"github.com/efritz/glock"
)

type (
	// CircuitBreaker protects the invocation of a function and monitors failures.
	// After a certain failure threshold is reached, future invocations will instead
	// return an ErrErrCircuitOpen instead of attempting to invoke the function again.
	CircuitBreaker interface {
		// Trip manually trips the circuit breaker. The circuit breaker will remain open
		// until it is manually reset.
		Trip()

		// Reset the circuit breaker.
		Reset()

		// ShouldTry returns true if the circuit breaker is closed or half-closed with
		// some probability. Successive calls to this method may yield different results
		// depending on the registered trip condition.
		ShouldTry() bool

		// MarkResult takes the result of the protected section and marks it as a success if
		// the error is nil or if the failure interpreter decides not to trip on this error.
		MarkResult(err error) bool

		// Call attempts to call the given function if the circuit breaker is closed, or if
		// the circuit breaker is half-closed (with some probability). Otherwise, return an
		// ErrCircuitOpen. If the function times out, the circuit breaker will fail with an
		// ErrInvocationTimeout. If the function is invoked and yields a value before the
		// timeout elapses, that value is returned.
		Call(f BreakerFunc) error

		// CallAsync invokes the given function in a goroutine, returning a channel which
		// may receive one non-nil error value and then close. The channel will close without
		// writing a value on success.
		CallAsync(f BreakerFunc) <-chan error
	}

	BreakerConfig func(*circuitBreaker)
	BreakerFunc   func(ctx context.Context) error

	circuitBreaker struct {
		invocationTimeout          time.Duration
		halfClosedRetryProbability float64
		resetBackoff               backoff.Backoff
		failureInterpreter         FailureInterpreter
		tripCondition              TripCondition
		clock                      glock.Clock
		state                      circuitState
		hardTrip                   bool
		lastFailureTime            *time.Time
		resetTimeout               *time.Duration
	}

	circuitState int
)

const (
	openState       circuitState = iota // Failure state
	closedState                         // Success state
	halfClosedState                     // Cautious, probabilistic retry state
)

// ErrCircuitOpen occurs when the Call method fails immediately.
var ErrCircuitOpen = fmt.Errorf("circuit is open")

// NewCircuitBreaker creates a new CircuitBreaker.
func NewCircuitBreaker(configs ...BreakerConfig) CircuitBreaker {
	breaker := &circuitBreaker{
		invocationTimeout:          100 * time.Millisecond,
		halfClosedRetryProbability: 0.5,
		resetBackoff:               backoff.NewConstantBackoff(1000 * time.Millisecond),
		failureInterpreter:         NewAnyErrorFailureInterpreter(),
		tripCondition:              NewConsecutiveFailureTripCondition(5),
		clock:                      glock.NewRealClock(),
		state:                      closedState,
	}

	for _, config := range configs {
		config(breaker)
	}

	return breaker
}

func WithInvocationTimeout(timeout time.Duration) BreakerConfig {
	return func(cb *circuitBreaker) { cb.invocationTimeout = timeout }
}

func WithHalfClosedRetryProbability(probability float64) BreakerConfig {
	return func(cb *circuitBreaker) { cb.halfClosedRetryProbability = probability }
}

func WithResetBackoff(resetBackoff backoff.Backoff) BreakerConfig {
	return func(cb *circuitBreaker) { cb.resetBackoff = resetBackoff }
}

func WithFailureInterpreter(failureInterpreter FailureInterpreter) BreakerConfig {
	return func(cb *circuitBreaker) { cb.failureInterpreter = failureInterpreter }
}

func WithTripCondition(tripCondition TripCondition) BreakerConfig {
	return func(cb *circuitBreaker) { cb.tripCondition = tripCondition }
}

func withClock(clock glock.Clock) BreakerConfig {
	return func(cb *circuitBreaker) { cb.clock = clock }
}

//
// Breaker Implementation

func (cb *circuitBreaker) Trip() {
	cb.hardTrip = true
}

func (cb *circuitBreaker) Reset() {
	cb.hardTrip = false
	cb.resetTimeout = nil

	cb.resetBackoff.Reset()
	cb.tripCondition.Success()
}

func (cb *circuitBreaker) ShouldTry() bool {
	if cb.hardTrip {
		return false
	}

	if !cb.tripCondition.ShouldTrip() {
		cb.state = closedState
		return true
	}

	if cb.state == closedState {
		cb.resetBackoff.Reset()
	}

	if cb.state != openState {
		reset := cb.resetBackoff.NextInterval()
		cb.resetTimeout = &reset
	}

	if cb.resetTimeoutElapsed() {
		cb.state = halfClosedState
		return rand.Float64() < cb.halfClosedRetryProbability
	}

	cb.state = openState
	return false
}

func (cb *circuitBreaker) resetTimeoutElapsed() bool {
	if cb.state != openState {
		return false
	}

	if cb.lastFailureTime == nil || cb.resetTimeout == nil {
		return false
	}

	return cb.clock.Now().Sub(*cb.lastFailureTime) >= *cb.resetTimeout
}

func (cb *circuitBreaker) MarkResult(err error) bool {
	if err != nil && cb.failureInterpreter.ShouldTrip(err) {
		now := cb.clock.Now()
		cb.lastFailureTime = &now
		cb.tripCondition.Failure()
		return false
	}

	cb.Reset()
	return true
}

func (cb *circuitBreaker) Call(f BreakerFunc) error {
	if !cb.ShouldTry() {
		return ErrCircuitOpen
	}

	if err := callWithTimeout(f, cb.clock, cb.invocationTimeout); !cb.MarkResult(err) {
		return err
	}

	return nil
}

func (cb *circuitBreaker) CallAsync(f BreakerFunc) <-chan error {
	ch := make(chan error, 1)

	go func() {
		defer close(ch)
		ch <- cb.Call(f)
	}()

	return ch
}
