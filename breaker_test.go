package overcurrent

import (
	"fmt"
	"testing"
	"time"

	"github.com/efritz/backoff"
	. "github.com/onsi/gomega"
)

type BreakerSuite struct{}

func (s *BreakerSuite) TestSuccess(t *testing.T) {
	cb := NewCircuitBreaker(DefaultCircuitBreakerConfig())
	fn := func() error {
		return nil
	}

	Expect(cb.Call(fn)).To(BeNil())
}

func (s *BreakerSuite) TestNaturalError(t *testing.T) {
	err := fmt.Errorf("test error")

	cb := NewCircuitBreaker(DefaultCircuitBreakerConfig())
	fn := func() error {
		return err
	}

	Expect(cb.Call(fn)).To(Equal(err))
}

func (s *BreakerSuite) TestNaturalErrorTrip(t *testing.T) {
	err := fmt.Errorf("test error")

	cb := NewCircuitBreaker(DefaultCircuitBreakerConfig())
	fn := func() error {
		return err
	}

	for i := 0; i < 5; i++ {
		Expect(cb.Call(fn)).To(Equal(err))
	}

	Expect(cb.Call(fn)).To(Equal(ErrCircuitOpen))
}

func (s *BreakerSuite) TestTimeout(t *testing.T) {
	cb := NewCircuitBreaker(DefaultCircuitBreakerConfig())
	fn := func() error {
		<-time.After(250 * time.Millisecond)
		return nil
	}

	Expect(cb.Call(fn)).To(Equal(ErrInvocationTimeout))
}

func (s *BreakerSuite) TestTimeoutTrip(t *testing.T) {
	cb := NewCircuitBreaker(DefaultCircuitBreakerConfig())
	fn := func() error {
		<-time.After(250 * time.Millisecond)
		return nil
	}

	for i := 0; i < 5; i++ {
		Expect(cb.Call(fn)).To(Equal(ErrInvocationTimeout))
	}

	Expect(cb.Call(fn)).To(Equal(ErrCircuitOpen))
}

func (s *BreakerSuite) TestTimeoutDisabled(t *testing.T) {
	cb := NewCircuitBreaker(&CircuitBreakerConfig{
		InvocationTimeout:          0,
		ResetBackoff:               backoff.NewConstantBackoff(250 * time.Millisecond),
		HalfClosedRetryProbability: 1,
		FailureInterpreter:         NewAnyErrorFailureInterpreter(),
		TripCondition:              NewConsecutiveFailureTripCondition(5),
	})

	fn := func() error {
		<-time.After(250 * time.Millisecond)
		return nil
	}

	Expect(cb.Call(fn)).To(BeNil())
}

func (s *BreakerSuite) TestHalfOpenFailure(t *testing.T) {
	cb := NewCircuitBreaker(&CircuitBreakerConfig{
		InvocationTimeout:          DefaultInvocationTimeout,
		ResetBackoff:               backoff.NewConstantBackoff(250 * time.Millisecond),
		HalfClosedRetryProbability: 1,
		FailureInterpreter:         NewAnyErrorFailureInterpreter(),
		TripCondition:              NewConsecutiveFailureTripCondition(5),
	})

	err := fmt.Errorf("test error")
	fn1 := func() error { return err }
	fn2 := func() error { return nil }

	for i := 0; i < 5; i++ {
		Expect(cb.Call(fn1)).To(Equal(err))
	}

	Expect(cb.Call(fn1)).To(Equal(ErrCircuitOpen))
	<-time.After(250 * time.Millisecond)
	Expect(cb.Call(fn1)).To(Equal(err))
	Expect(cb.Call(fn2)).To(Equal(ErrCircuitOpen))
}

func (s *BreakerSuite) TestHalfOpenReset(t *testing.T) {
	cb := NewCircuitBreaker(&CircuitBreakerConfig{
		InvocationTimeout:          DefaultInvocationTimeout,
		ResetBackoff:               backoff.NewConstantBackoff(250 * time.Millisecond),
		HalfClosedRetryProbability: 1,
		FailureInterpreter:         NewAnyErrorFailureInterpreter(),
		TripCondition:              NewConsecutiveFailureTripCondition(5),
	})

	err := fmt.Errorf("test error")
	fn1 := func() error { return err }
	fn2 := func() error { return nil }

	for i := 0; i < 5; i++ {
		Expect(cb.Call(fn1)).To(Equal(err))
	}

	Expect(cb.Call(fn1)).To(Equal(ErrCircuitOpen))
	<-time.After(250 * time.Millisecond)
	Expect(cb.Call(fn2)).To(BeNil())
}

func (s *BreakerSuite) TestHalfOpenProbability(t *testing.T) {
	runs := 5000
	prob := .25
	dist := .01

	success := 0
	failure := 0

	fn1 := func() error { return fmt.Errorf("test error") }
	fn2 := func() error {
		success++
		return nil
	}

	for i := 0; i < runs; i++ {
		cb := NewCircuitBreaker(&CircuitBreakerConfig{
			InvocationTimeout:          DefaultInvocationTimeout,
			ResetBackoff:               backoff.NewConstantBackoff(1 * time.Nanosecond),
			HalfClosedRetryProbability: prob,
			FailureInterpreter:         NewAnyErrorFailureInterpreter(),
			TripCondition:              NewConsecutiveFailureTripCondition(1),
		})

		cb.Call(fn1)
		<-time.After(1 * time.Nanosecond)

		if cb.Call(fn2) == ErrCircuitOpen {
			failure++
		}
	}

	lower := int(float64(runs)*prob - float64(runs)*dist)
	upper := int(float64(runs)*prob + float64(runs)*dist)

	Expect(success + failure).To(Equal(runs))
	Expect(success).To(BeNumerically(">=", lower))
	Expect(success).To(BeNumerically("<=", upper))
}

func (s *BreakerSuite) TestResetBackoff(t *testing.T) {
	cb := NewCircuitBreaker(&CircuitBreakerConfig{
		InvocationTimeout:          DefaultInvocationTimeout,
		ResetBackoff:               backoff.NewLinearBackoff(100*time.Millisecond, 50*time.Millisecond, time.Second),
		HalfClosedRetryProbability: 1,
		FailureInterpreter:         NewAnyErrorFailureInterpreter(),
		TripCondition:              NewConsecutiveFailureTripCondition(5),
	})

	err := fmt.Errorf("test error")
	fn1 := func() error { return err }
	fn2 := func() error { return nil }

	runTest := func() {
		for i := 0; i < 5; i++ {
			Expect(cb.Call(fn1)).To(Equal(err))
		}

		Expect(cb.Call(fn1)).To(Equal(ErrCircuitOpen))
		<-time.After(100 * time.Millisecond)
		Expect(cb.Call(fn1)).To(Equal(err))

		Expect(cb.Call(fn2)).To(Equal(ErrCircuitOpen))
		<-time.After(150 * time.Millisecond)
		Expect(cb.Call(fn1)).To(Equal(err))

		Expect(cb.Call(fn2)).To(Equal(ErrCircuitOpen))
		<-time.After(200 * time.Millisecond)
		Expect(cb.Call(fn1)).To(Equal(err))

		Expect(cb.Call(fn2)).To(Equal(ErrCircuitOpen))
		<-time.After(250 * time.Millisecond)
		Expect(cb.Call(fn2)).To(BeNil())
	}

	runTest()
	runTest()
}

func (s *BreakerSuite) TestHardTrip(t *testing.T) {
	cb := NewCircuitBreaker(&CircuitBreakerConfig{
		InvocationTimeout:          DefaultInvocationTimeout,
		ResetBackoff:               backoff.NewConstantBackoff(250 * time.Millisecond),
		HalfClosedRetryProbability: 1,
		FailureInterpreter:         NewAnyErrorFailureInterpreter(),
		TripCondition:              NewConsecutiveFailureTripCondition(1),
	})

	fn := func() error {
		return nil
	}

	Expect(cb.Call(fn)).To(BeNil())
	cb.Trip()

	Expect(cb.Call(fn)).To(Equal(ErrCircuitOpen))
	<-time.After(250 * time.Millisecond)
	Expect(cb.Call(fn)).To(Equal(ErrCircuitOpen))

	cb.Reset()
	Expect(cb.Call(fn)).To(BeNil())
}

func (s *BreakerSuite) TestHardReset(t *testing.T) {
	cb := NewCircuitBreaker(DefaultCircuitBreakerConfig())

	err := fmt.Errorf("test error")
	fn1 := func() error { return err }
	fn2 := func() error { return nil }

	for i := 0; i < 5; i++ {
		Expect(cb.Call(fn1)).To(Equal(err))
	}

	Expect(cb.Call(fn2)).To(Equal(ErrCircuitOpen))

	cb.Reset()
	Expect(cb.Call(fn2)).To(BeNil())
}
