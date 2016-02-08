package overcurrent

import (
	"errors"
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

type Breaker struct {
	Events chan CircuitEvent

	circuit func() error

	invocationTimeout     float64
	failureThreshold      int
	resetTimeout          float64
	halfClosedProbability float64

	failureCount    int
	hardTrip        bool
	lastFailureTime *time.Time
}

func NewBreaker(circuit func() error) *Breaker {
	return &Breaker{
		Events:                make(chan CircuitEvent),
		circuit:               circuit,
		invocationTimeout:     0.01,
		failureThreshold:      5,
		resetTimeout:          0.1,
		halfClosedProbability: 0.5,
	}
}

func (b *Breaker) State() CircuitState {
	if b.failureCount >= b.failureThreshold {
		if time.Now().Sub(*b.lastFailureTime) > time.Duration(b.resetTimeout) {
			return HalfClosedState
		} else {
			return OpenState
		}
	} else {
		return ClosedState
	}
}

func (b *Breaker) Call() error {
	if b.shouldTry() {
		return CircuitOpenError
	}

	if err := b.callWithTimeout(); err != nil {
		b.recordError()
		return err
	}

	b.Reset()
	return nil
}

func (b *Breaker) shouldTry() bool {
	return b.hardTrip || b.State() == OpenState || (b.State() == HalfClosedState && rand.Float32() <= b.halfClosedProbability)
}

func (b *Breaker) callWithTimeout() error {
	if b.invocationTimeout == 0 {
		return b.circuit()
	}

	c := make(chan error)
	go func() {
		c <- b.circuit()
		close(c)
	}()

	select {
	case err := <-c:
		return err
	case <-time.After(time.Duration(b.invocationTimeout)):
		return InvocationTimeoutError
	}
}

func (b *Breaker) recordError() {
	now := time.Now()

	b.failureCount++
	b.lastFailureTime = &now
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
	b.failureCount = 0
	b.lastFailureTime = nil
	b.hardTrip = false
	b.sendEvent(ResetEvent)
}
