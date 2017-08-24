package overcurrent

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/efritz/glock"
	. "github.com/onsi/gomega"
)

type TimeoutSuite struct{}

func (s *TimeoutSuite) TestZeroTimeout(t *testing.T) {
	fn := func(ctx context.Context) error {
		return nil
	}

	Expect(callWithTimeout(fn, nil, 0)).To(BeNil())
}

func (s *TimeoutSuite) TestNoError(t *testing.T) {
	var (
		clock = glock.NewMockClock()
		fn    = func(ctx context.Context) error {
			return nil
		}
	)

	Expect(callWithTimeout(fn, clock, time.Minute)).To(BeNil())

	args := clock.GetAfterArgs()
	Expect(args).To(HaveLen(1))
	Expect(args[0]).To(Equal(time.Minute))
}

func (s *TimeoutSuite) TestError(t *testing.T) {
	var (
		clock = glock.NewMockClock()
		fn    = func(ctx context.Context) error {
			return errors.New("utoh")
		}
	)

	Expect(callWithTimeout(fn, clock, time.Minute)).To(MatchError("utoh"))

	args := clock.GetAfterArgs()
	Expect(args).To(HaveLen(1))
	Expect(args[0]).To(Equal(time.Minute))
}

func (s *TimeoutSuite) TestsTimeout(t *testing.T) {
	var (
		clock  = glock.NewMockClock()
		sync   = make(chan struct{})
		errors = make(chan error)
		fn     = func(ctx context.Context) error {
			defer close(sync)
			<-ctx.Done()
			return nil
		}
	)

	go func() {
		defer close(errors)
		errors <- callWithTimeout(fn, clock, time.Minute)
	}()

	Consistently(sync).ShouldNot(Receive())
	Consistently(errors).ShouldNot(Receive())
	clock.Advance(time.Minute * 2)
	Eventually(errors).Should(Receive(Equal(ErrInvocationTimeout)))
	Eventually(sync).Should(BeClosed())
}
