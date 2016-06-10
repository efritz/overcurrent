package overcurrent

import (
	"errors"
	"time"

	"github.com/efritz/backoff"
	. "gopkg.in/check.v1"
)

func (s *OvercurrentSuite) TestSuccess(c *C) {
	cb := NewCircuitBreaker(DefaultCircuitBreakerConfig())
	fn := func() error {
		return nil
	}

	c.Assert(cb.Call(fn), Equals, nil)
}

func (s *OvercurrentSuite) TestNaturalError(c *C) {
	err := errors.New("Test error.")

	cb := NewCircuitBreaker(DefaultCircuitBreakerConfig())
	fn := func() error {
		return err
	}

	c.Assert(cb.Call(fn), Equals, err)
}

func (s *OvercurrentSuite) TestNaturalErrorTrip(c *C) {
	err := errors.New("Test error.")

	cb := NewCircuitBreaker(DefaultCircuitBreakerConfig())
	fn := func() error {
		return err
	}

	for i := 0; i < 5; i++ {
		c.Assert(cb.Call(fn), Equals, err)
	}

	c.Assert(cb.Call(fn), Equals, CircuitOpenError)
}

func (s *OvercurrentSuite) TestTimeout(c *C) {
	cb := NewCircuitBreaker(DefaultCircuitBreakerConfig())
	fn := func() error {
		<-time.After(250 * time.Millisecond)
		return nil
	}

	c.Assert(cb.Call(fn), Equals, InvocationTimeoutError)
}

func (s *OvercurrentSuite) TestTimeoutTrip(c *C) {
	cb := NewCircuitBreaker(DefaultCircuitBreakerConfig())
	fn := func() error {
		<-time.After(250 * time.Millisecond)
		return nil
	}

	for i := 0; i < 5; i++ {
		c.Assert(cb.Call(fn), Equals, InvocationTimeoutError)
	}

	c.Assert(cb.Call(fn), Equals, CircuitOpenError)
}

func (s *OvercurrentSuite) TestTimeoutDisabled(c *C) {
	cb := NewCircuitBreaker(&CircuitBreakerConfig{
		InvocationTimeout:          0,
		ResetBackOff:               backoff.NewConstantBackOff(250 * time.Millisecond),
		HalfClosedRetryProbability: 1,
		FailureInterpreter:         NewAnyErrorFailureInterpreter(),
		TripCondition:              NewConsecutiveFailureTripCondition(5),
	})

	fn := func() error {
		<-time.After(250 * time.Millisecond)
		return nil
	}

	c.Assert(cb.Call(fn), Equals, nil)
}

func (s *OvercurrentSuite) TestHalfOpenFailure(c *C) {
	cb := NewCircuitBreaker(&CircuitBreakerConfig{
		InvocationTimeout:          DefaultInvocationTimeout,
		ResetBackOff:               backoff.NewConstantBackOff(250 * time.Millisecond),
		HalfClosedRetryProbability: 1,
		FailureInterpreter:         NewAnyErrorFailureInterpreter(),
		TripCondition:              NewConsecutiveFailureTripCondition(5),
	})

	err := errors.New("Test error.")
	fn1 := func() error { return err }
	fn2 := func() error { return nil }

	for i := 0; i < 5; i++ {
		c.Assert(cb.Call(fn1), Equals, err)
	}

	c.Assert(cb.Call(fn1), Equals, CircuitOpenError)
	<-time.After(250 * time.Millisecond)
	c.Assert(cb.Call(fn1), Equals, err)
	c.Assert(cb.Call(fn2), Equals, CircuitOpenError)
}

func (s *OvercurrentSuite) TestHalfOpenReset(c *C) {
	cb := NewCircuitBreaker(&CircuitBreakerConfig{
		InvocationTimeout:          DefaultInvocationTimeout,
		ResetBackOff:               backoff.NewConstantBackOff(250 * time.Millisecond),
		HalfClosedRetryProbability: 1,
		FailureInterpreter:         NewAnyErrorFailureInterpreter(),
		TripCondition:              NewConsecutiveFailureTripCondition(5),
	})

	err := errors.New("Test error.")
	fn1 := func() error { return err }
	fn2 := func() error { return nil }

	for i := 0; i < 5; i++ {
		c.Assert(cb.Call(fn1), Equals, err)
	}

	c.Assert(cb.Call(fn1), Equals, CircuitOpenError)
	<-time.After(250 * time.Millisecond)
	c.Assert(cb.Call(fn2), Equals, nil)
}

func (s *OvercurrentSuite) TestHalfOpenProbability(c *C) {
	runs := 5000
	prob := .25
	dist := .01

	success := 0
	failure := 0

	fn1 := func() error { return errors.New("Test error.") }
	fn2 := func() error {
		success++
		return nil
	}

	for i := 0; i < runs; i++ {
		cb := NewCircuitBreaker(&CircuitBreakerConfig{
			InvocationTimeout:          DefaultInvocationTimeout,
			ResetBackOff:               backoff.NewConstantBackOff(1 * time.Nanosecond),
			HalfClosedRetryProbability: prob,
			FailureInterpreter:         NewAnyErrorFailureInterpreter(),
			TripCondition:              NewConsecutiveFailureTripCondition(1),
		})

		cb.Call(fn1)
		<-time.After(1 * time.Nanosecond)

		if cb.Call(fn2) == CircuitOpenError {
			failure++
		}
	}

	lower := int(float64(runs)*prob - float64(runs)*dist)
	upper := int(float64(runs)*prob + float64(runs)*dist)

	c.Assert(success+failure, Equals, runs)
	c.Assert(lower <= success && success <= upper, Equals, true)
}

func (s *OvercurrentSuite) TestResetBackOff(c *C) {
	cb := NewCircuitBreaker(&CircuitBreakerConfig{
		InvocationTimeout:          DefaultInvocationTimeout,
		ResetBackOff:               backoff.NewLinearBackOff(100*time.Millisecond, 50*time.Millisecond, time.Second),
		HalfClosedRetryProbability: 1,
		FailureInterpreter:         NewAnyErrorFailureInterpreter(),
		TripCondition:              NewConsecutiveFailureTripCondition(5),
	})

	err := errors.New("Test error.")
	fn1 := func() error { return err }
	fn2 := func() error { return nil }

	runTest := func() {
		for i := 0; i < 5; i++ {
			c.Assert(cb.Call(fn1), Equals, err)
		}

		c.Assert(cb.Call(fn1), Equals, CircuitOpenError)
		<-time.After(100 * time.Millisecond)
		c.Assert(cb.Call(fn1), Equals, err)
		c.Assert(cb.Call(fn2), Equals, CircuitOpenError)
		<-time.After(150 * time.Millisecond)
		c.Assert(cb.Call(fn1), Equals, err)
		c.Assert(cb.Call(fn2), Equals, CircuitOpenError)
		<-time.After(200 * time.Millisecond)
		c.Assert(cb.Call(fn1), Equals, err)
		c.Assert(cb.Call(fn2), Equals, CircuitOpenError)
		<-time.After(250 * time.Millisecond)
		c.Assert(cb.Call(fn2), Equals, nil)
	}

	runTest()
	runTest()
}

func (s *OvercurrentSuite) TestHardTrip(c *C) {
	cb := NewCircuitBreaker(&CircuitBreakerConfig{
		InvocationTimeout:          DefaultInvocationTimeout,
		ResetBackOff:               backoff.NewConstantBackOff(250 * time.Millisecond),
		HalfClosedRetryProbability: 1,
		FailureInterpreter:         NewAnyErrorFailureInterpreter(),
		TripCondition:              NewConsecutiveFailureTripCondition(1),
	})

	fn := func() error {
		return nil
	}

	c.Assert(cb.Call(fn), Equals, nil)
	cb.Trip()

	c.Assert(cb.Call(fn), Equals, CircuitOpenError)
	<-time.After(250 * time.Millisecond)
	c.Assert(cb.Call(fn), Equals, CircuitOpenError)

	cb.Reset()
	c.Assert(cb.Call(fn), Equals, nil)
}

func (s *OvercurrentSuite) TestHardReset(c *C) {
	cb := NewCircuitBreaker(DefaultCircuitBreakerConfig())

	err := errors.New("Test error.")
	fn1 := func() error { return err }
	fn2 := func() error { return nil }

	for i := 0; i < 5; i++ {
		c.Assert(cb.Call(fn1), Equals, err)
	}

	c.Assert(cb.Call(fn2), Equals, CircuitOpenError)

	cb.Reset()
	c.Assert(cb.Call(fn2), Equals, nil)
}
