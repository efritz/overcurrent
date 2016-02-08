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

	DefaultInvocationTimeout          = 0.01
	DefaultResetTimeout               = 0.1
	DefaultHalfClosedRetryProbability = 0.5
)

var (
	CircuitOpenError       = errors.New("Circuit is open.")
	InvocationTimeoutError = errors.New("Invocation has timed out.")
)

//
//

type BreakerConfig struct {
	InvocationTimeout          float64
	ResetTimeout               float64
	HalfClosedRetryProbability float64
	TripCondition              TripCondition
	FailureInterpreter         FailureInterpreter
}

func NewBreakerConfig() BreakerConfig {
	return BreakerConfig{
		InvocationTimeout:          DefaultInvocationTimeout,
		ResetTimeout:               DefaultResetTimeout,
		HalfClosedRetryProbability: DefaultHalfClosedRetryProbability,
		TripCondition:              NewConsecutiveFailureTripCondition(5),
		FailureInterpreter:         NewAnyErrorFailureInterpreter(),
	}
}

//
//

type Breaker struct {
	circuit func() error
	config  BreakerConfig
	Events  chan CircuitEvent

	hardTrip        bool
	lastFailureTime *time.Time
}

func NewBreaker(circuit func() error, config BreakerConfig) *Breaker {
	return &Breaker{
		circuit: circuit,
		config:  config,
		Events:  make(chan CircuitEvent),
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

func (b *Breaker) Call() error {
	if !b.shouldTry() {
		return CircuitOpenError
	}

	if err := b.callWithTimeout(); err != nil && b.config.FailureInterpreter.ShouldTrip(err) {
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

func (b *Breaker) callWithTimeout() error {
	if b.config.InvocationTimeout == 0 {
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
