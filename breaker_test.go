package overcurrent

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/efritz/backoff"
	"github.com/efritz/glock"
	. "github.com/onsi/gomega"
)

type BreakerSuite struct{}

func (s *BreakerSuite) TestSuccess(t *testing.T) {
	breaker := NewCircuitBreaker(testConfig())
	Expect(breaker.Call(nilFunc)).To(BeNil())
}

func (s *BreakerSuite) TestNaturalError(t *testing.T) {
	breaker := NewCircuitBreaker(testConfig())
	Expect(breaker.Call(errFunc)).To(Equal(errTest))
}

func (s *BreakerSuite) TestNaturalErrorTrip(t *testing.T) {
	breaker := NewCircuitBreaker(testConfig())

	for i := 0; i < 5; i++ {
		Expect(breaker.Call(errFunc)).To(Equal(errTest))
	}

	Expect(breaker.Call(errFunc)).To(Equal(ErrCircuitOpen))
}

func (s *BreakerSuite) TestTimeout(t *testing.T) {
	var (
		clock   = glock.NewMockClock()
		breaker = NewCircuitBreaker(testConfig(), withClock(clock))
	)

	go func() {
		// Ensure after was called before advancing
		for len(clock.GetAfterArgs()) == 0 {
			<-time.After(time.Millisecond)
		}

		clock.Advance(time.Minute)
	}()

	Expect(breaker.Call(blockingFunc)).To(Equal(ErrInvocationTimeout))
}

func (s *BreakerSuite) TestTimeoutTrip(t *testing.T) {
	var (
		clock   = glock.NewMockClock()
		breaker = NewCircuitBreaker(testConfig(), withClock(clock))
	)

	go func() {
		for i := 0; i < 5; i++ {
			// Ensure after was called before advancing
			for len(clock.GetAfterArgs()) == 0 {
				<-time.After(time.Millisecond)
			}

			clock.Advance(time.Minute)
		}
	}()

	for i := 0; i < 5; i++ {
		Expect(breaker.Call(blockingFunc)).To(Equal(ErrInvocationTimeout))
	}

	Expect(breaker.Call(blockingFunc)).To(Equal(ErrCircuitOpen))
}

func (s *BreakerSuite) TestTimeoutDisabled(t *testing.T) {
	var (
		clock   = glock.NewMockClock()
		sync    = make(chan time.Time)
		breaker = NewCircuitBreaker(
			testConfig(),
			withClock(clock),
			WithInvocationTimeout(0),
		)
	)

	fn := func(ctx context.Context) error {
		<-sync
		return nil
	}

	close(sync)
	Expect(breaker.Call(fn)).To(BeNil())
	Expect(clock.GetAfterArgs()).To(HaveLen(0))
}

func (s *BreakerSuite) TestHalfOpenFailure(t *testing.T) {
	var (
		clock   = glock.NewMockClock()
		breaker = NewCircuitBreaker(
			testConfig(),
			withClock(clock),
			WithHalfClosedRetryProbability(1),
		)
	)

	for i := 0; i < 5; i++ {
		Expect(breaker.Call(errFunc)).To(Equal(errTest))
	}

	Expect(breaker.Call(errFunc)).To(Equal(ErrCircuitOpen))

	// Wait for retry backoff
	clock.Advance(15 * time.Second)
	Expect(breaker.Call(errFunc)).To(Equal(errTest))
	Expect(breaker.Call(nilFunc)).To(Equal(ErrCircuitOpen))
}

func (s *BreakerSuite) TestHalfOpenReset(t *testing.T) {
	var (
		clock   = glock.NewMockClock()
		breaker = NewCircuitBreaker(
			testConfig(),
			withClock(clock),
			WithHalfClosedRetryProbability(1),
		)
	)

	for i := 0; i < 5; i++ {
		Expect(breaker.Call(errFunc)).To(Equal(errTest))
	}

	Expect(breaker.Call(errFunc)).To(Equal(ErrCircuitOpen))

	// Wait for retry backoff
	clock.Advance(15 * time.Second)
	Expect(breaker.Call(nilFunc)).To(BeNil())
}

var (
	TRIALS      = 50000
	PROBAIBLITY = 0.25
)

func (s *BreakerSuite) TestHalfOpenProbability(t *testing.T) {
	success := 0
	failure := 0

	for i := 0; i < TRIALS; i++ {
		if runHalfOpenProbabilityTrial(PROBAIBLITY) {
			success++
		} else {
			failure++
		}
	}

	Expect(success + failure).To(Equal(TRIALS))
	Expect(success).To(BeNumerically("~", success, float64(TRIALS)*PROBAIBLITY))
}

func runHalfOpenProbabilityTrial(probability float64) (called bool) {
	var (
		clock   = glock.NewMockClock()
		breaker = NewCircuitBreaker(
			testConfig(),
			withClock(clock),
			WithHalfClosedRetryProbability(probability),
			WithTripCondition(NewConsecutiveFailureTripCondition(1)),
		)
	)

	breaker.Call(errFunc)
	clock.Advance(15 * time.Second)
	return breaker.Call(nilFunc) != ErrCircuitOpen
}

func (s *BreakerSuite) TestResetBackoff(t *testing.T) {
	var (
		clock   = glock.NewMockClock()
		breaker = NewCircuitBreaker(
			testConfig(),
			withClock(clock),
			WithHalfClosedRetryProbability(1),
			WithResetBackoff(backoff.NewLinearBackoff(
				100*time.Millisecond,
				50*time.Millisecond,
				time.Second,
			)),
		)
	)

	for i := 0; i < 2; i++ {
		for j := 0; j < 5; j++ {
			Expect(breaker.Call(errFunc)).To(Equal(errTest))
		}

		Expect(breaker.Call(errFunc)).To(Equal(ErrCircuitOpen))
		clock.Advance(100 * time.Millisecond)
		Expect(breaker.Call(errFunc)).To(Equal(errTest))

		Expect(breaker.Call(nilFunc)).To(Equal(ErrCircuitOpen))
		clock.Advance(150 * time.Millisecond)
		Expect(breaker.Call(errFunc)).To(Equal(errTest))

		Expect(breaker.Call(nilFunc)).To(Equal(ErrCircuitOpen))
		clock.Advance(200 * time.Millisecond)
		Expect(breaker.Call(errFunc)).To(Equal(errTest))

		Expect(breaker.Call(nilFunc)).To(Equal(ErrCircuitOpen))
		clock.Advance(250 * time.Millisecond)
		Expect(breaker.Call(nilFunc)).To(BeNil())
	}
}

func (s *BreakerSuite) TestHardTrip(t *testing.T) {
	var (
		clock   = glock.NewMockClock()
		breaker = NewCircuitBreaker(testConfig(), withClock(clock))
	)

	Expect(breaker.Call(nilFunc)).To(BeNil())
	breaker.Trip()

	Expect(breaker.Call(nilFunc)).To(Equal(ErrCircuitOpen))
	clock.Advance(250 * time.Millisecond)
	Expect(breaker.Call(nilFunc)).To(Equal(ErrCircuitOpen))

	breaker.Reset()
	Expect(breaker.Call(nilFunc)).To(BeNil())
}

func (s *BreakerSuite) TestHardReset(t *testing.T) {
	breaker := NewCircuitBreaker(testConfig())

	for i := 0; i < 5; i++ {
		Expect(breaker.Call(errFunc)).To(Equal(errTest))
	}

	Expect(breaker.Call(nilFunc)).To(Equal(ErrCircuitOpen))
	breaker.Reset()
	Expect(breaker.Call(nilFunc)).To(BeNil())
}

//
// Detailed in Issue #3
//

func (s *BreakerSuite) TestTripAfterSuccess(t *testing.T) {
	var (
		clock   = glock.NewMockClock()
		breaker = NewCircuitBreaker(
			testConfig(),
			WithHalfClosedRetryProbability(1),
			WithResetBackoff(backoff.NewConstantBackoff(time.Second)),
			WithFailureInterpreter(NewAnyErrorFailureInterpreter()),
			WithTripCondition(NewPercentageFailureTripCondition(100, 0.5)),
			withClock(clock),
		)
	)

	for i := 0; i < 40; i++ {
		breaker.MarkResult(nil)
	}

	for i := 0; i < 60; i++ {
		breaker.MarkResult(fmt.Errorf("utoh"))
	}

	Expect(breaker.ShouldTry()).To(BeFalse())
	clock.Advance(time.Minute)
	Expect(breaker.ShouldTry()).To(BeTrue())

	// Before the bug was patched, it was possible to get into a situation
	// where a nil result can cause the breaker to trip again, and the state
	// of the breaker made it so that the condition which determined when the
	// half-closed state was entered was indefinitely false.

	for i := 0; i < 50; i++ {
		breaker.MarkResult(nil)
		Expect(breaker.ShouldTry()).To(BeFalse())
		clock.Advance(time.Minute)
		Expect(breaker.ShouldTry()).To(BeTrue())
	}

	breaker.MarkResult(nil)
	Expect(breaker.ShouldTry()).To(BeTrue())
	Expect(breaker.ShouldTry()).To(BeTrue())
	Expect(breaker.ShouldTry()).To(BeTrue())
}

//
//
//

var errTest = fmt.Errorf("test error")

func nilFunc(ctx context.Context) error {
	return nil
}

func errFunc(ctx context.Context) error {
	return errTest
}

func blockingFunc(ctx context.Context) error {
	ch := make(chan struct{})
	<-ch

	return nil
}

func testConfig() BreakerConfig {
	return func(cb *circuitBreaker) {
		cb.invocationTimeout = time.Minute
		cb.halfClosedRetryProbability = 0.75
		cb.resetBackoff = backoff.NewConstantBackoff(15 * time.Second)
		cb.failureInterpreter = NewAnyErrorFailureInterpreter()
		cb.tripCondition = NewConsecutiveFailureTripCondition(5)
	}
}
