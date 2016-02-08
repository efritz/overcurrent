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

type Breaker struct {
	config BreakerConfig
	Events chan CircuitEvent

	hardTrip        bool
	lastFailureTime *time.Time
}

func NewBreaker(config BreakerConfig) *Breaker {
	return &Breaker{
		config: config,
		Events: make(chan CircuitEvent),
	}
}

func (b *Breaker) State() CircuitState {
	if b.config.TripCondition.ShouldTrip() {
		if time.Now().Sub(*b.lastFailureTime) > time.Duration(b.config.ResetTimeout) {
			return HalfClosedState
		} else {
			return OpenState
		}
	} else {
		return ClosedState
	}
}

func (b *Breaker) Call(f func() error) error {
	if !b.shouldTry() {
		return CircuitOpenError
	}

	if err := b.callWithTimeout(f); err != nil && b.config.FailureInterpreter.ShouldTrip(err) {
		b.recordError()
		return err
	}

	b.Reset()
	return nil
}

func (b *Breaker) shouldTry() bool {
	if b.hardTrip || b.State() == OpenState {
		return false
	}

	if b.State() == HalfClosedState {
		return rand.Float64() <= b.config.HalfClosedRetryProbability
	}

	return true
}

func (b *Breaker) callWithTimeout(f func() error) error {
	if b.config.InvocationTimeout == 0 {
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
	case <-time.After(time.Duration(b.config.InvocationTimeout)):
		return InvocationTimeoutError
	}
}

func (b *Breaker) recordError() {
	now := time.Now()
	b.lastFailureTime = &now

	b.config.TripCondition.Failure()
	b.sendEvent(TripEvent)
}

func (b *Breaker) sendEvent(event CircuitEvent) {
	select {
	case b.Events <- event:
	}
}

func (b *Breaker) Trip() {
	b.hardTrip = true
	b.recordError()
}

func (b *Breaker) Reset() {
	b.lastFailureTime = nil
	b.hardTrip = false

	b.config.TripCondition.Success()
	b.sendEvent(ResetEvent)
}
