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
		clockChan = make(chan time.Time)
		clock     = newMockClock(clockChan)
		breaker   = newCircuitBreakerWithClock(testConfig(), clock)
	)

	go func() {
		defer close(clockChan)
		clockChan <- clock.now
	}()

	Expect(breaker.Call(blockingFunc)).To(Equal(ErrInvocationTimeout))
	Expect(clock.afterArgs).To(HaveLen(1))
	Expect(clock.afterArgs[0]).To(Equal(time.Minute))
}

func (s *BreakerSuite) TestTimeoutTrip(t *testing.T) {
	var (
		clockChan = make(chan time.Time)
		clock     = newMockClock(clockChan)
		breaker   = newCircuitBreakerWithClock(testConfig(), clock)
	)

	go func() {
		defer close(clockChan)

		for i := 0; i < 5; i++ {
			clockChan <- clock.now
		}
	}()

	for i := 0; i < 5; i++ {
		Expect(breaker.Call(blockingFunc)).To(Equal(ErrInvocationTimeout))
	}

	Expect(breaker.Call(blockingFunc)).To(Equal(ErrCircuitOpen))
}

func (s *BreakerSuite) TestTimeoutDisabled(t *testing.T) {

	var (
		clockChan = make(chan time.Time)
		clock     = newMockClock(clockChan)
		config    = testConfig()
		breaker   = newCircuitBreakerWithClock(config, clock)
		sync      = make(chan time.Time)
	)

	fn := func() error {
		<-sync
		return nil
	}

	go func() {
		defer close(clockChan)
		clockChan <- clock.now
	}()

	Expect(breaker.Call(fn)).To(Equal(ErrInvocationTimeout))

	config.InvocationTimeout = 0
	close(sync)
	Expect(breaker.Call(fn)).To(BeNil())
}

func (s *BreakerSuite) TestHalfOpenFailure(t *testing.T) {
	var (
		config    = testConfig()
		clockChan = make(chan time.Time)
		clock     = newMockClock(clockChan)
		breaker   = newCircuitBreakerWithClock(config, clock)
	)

	defer close(clockChan)
	config.HalfClosedRetryProbability = 1

	for i := 0; i < 5; i++ {
		Expect(breaker.Call(errFunc)).To(Equal(errTest))
	}

	Expect(breaker.Call(errFunc)).To(Equal(ErrCircuitOpen))

	// Wait for retry backoff
	clock.advance(15 * time.Second)
	Expect(breaker.Call(errFunc)).To(Equal(errTest))
	Expect(breaker.Call(nilFunc)).To(Equal(ErrCircuitOpen))
}

func (s *BreakerSuite) TestHalfOpenReset(t *testing.T) {
	var (
		config    = testConfig()
		clockChan = make(chan time.Time)
		clock     = newMockClock(clockChan)
		breaker   = newCircuitBreakerWithClock(config, clock)
	)

	defer close(clockChan)
	config.HalfClosedRetryProbability = 1

	for i := 0; i < 5; i++ {
		Expect(breaker.Call(errFunc)).To(Equal(errTest))
	}

	Expect(breaker.Call(errFunc)).To(Equal(ErrCircuitOpen))

	// Wait for retry backoff
	clock.advance(15 * time.Second)
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
		config    = testConfig()
		clockChan = make(chan time.Time)
		clock     = newMockClock(clockChan)
		breaker   = newCircuitBreakerWithClock(config, clock)
	)

	defer close(clockChan)
	config.HalfClosedRetryProbability = probability
	config.TripCondition = NewConsecutiveFailureTripCondition(1)

	breaker.Call(errFunc)
	clock.advance(15 * time.Second)
	return breaker.Call(nilFunc) != ErrCircuitOpen
}

func (s *BreakerSuite) TestResetBackoff(t *testing.T) {
	var (
		config    = testConfig()
		clockChan = make(chan time.Time)
		clock     = newMockClock(clockChan)
		breaker   = newCircuitBreakerWithClock(config, clock)
	)

	config.HalfClosedRetryProbability = 1
	config.ResetBackoff = backoff.NewLinearBackoff(
		100*time.Millisecond,
		50*time.Millisecond,
		time.Second,
	)

	for i := 0; i < 2; i++ {
		for j := 0; j < 5; j++ {
			Expect(breaker.Call(errFunc)).To(Equal(errTest))
		}

		Expect(breaker.Call(errFunc)).To(Equal(ErrCircuitOpen))
		clock.advance(100 * time.Millisecond)
		Expect(breaker.Call(errFunc)).To(Equal(errTest))

		Expect(breaker.Call(nilFunc)).To(Equal(ErrCircuitOpen))
		clock.advance(150 * time.Millisecond)
		Expect(breaker.Call(errFunc)).To(Equal(errTest))

		Expect(breaker.Call(nilFunc)).To(Equal(ErrCircuitOpen))
		clock.advance(200 * time.Millisecond)
		Expect(breaker.Call(errFunc)).To(Equal(errTest))

		Expect(breaker.Call(nilFunc)).To(Equal(ErrCircuitOpen))
		clock.advance(250 * time.Millisecond)
		Expect(breaker.Call(nilFunc)).To(BeNil())
	}
}

func (s *BreakerSuite) TestHardTrip(t *testing.T) {
	var (
		clockChan = make(chan time.Time)
		clock     = newMockClock(clockChan)
		breaker   = newCircuitBreakerWithClock(testConfig(), clock)
	)

	defer close(clockChan)

	Expect(breaker.Call(nilFunc)).To(BeNil())
	breaker.Trip()

	Expect(breaker.Call(nilFunc)).To(Equal(ErrCircuitOpen))
	clock.advance(250 * time.Millisecond)
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
//
//

var (
	errTest = fmt.Errorf("test error")
)

func nilFunc() error {
	return nil
}

func errFunc() error {
	return errTest
}

func blockingFunc() error {
	ch := make(chan struct{})
	<-ch

	return nil
}

func testConfig() *CircuitBreakerConfig {
	return &CircuitBreakerConfig{
		InvocationTimeout:          time.Minute,
		HalfClosedRetryProbability: 0.75,
		ResetBackoff:               backoff.NewConstantBackoff(15 * time.Second),
		FailureInterpreter:         NewAnyErrorFailureInterpreter(),
		TripCondition:              NewConsecutiveFailureTripCondition(5),
	}
}
