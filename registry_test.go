package overcurrent

import (
	"context"
	"errors"
	"sync"
	"time"

	"github.com/efritz/glock"

	"github.com/aphistic/sweet"
	. "github.com/onsi/gomega"
)

type RegistrySuite struct{}

func (s *RegistrySuite) TestSuccessfulCall(t sweet.T) {
	var (
		r      = NewRegistry()
		called = false
	)

	r.Configure("test")

	err := r.Call("test", func(ctx context.Context) error {
		called = true
		return nil
	}, nil)

	Expect(err).To(BeNil())
	Expect(called).To(BeTrue())
}

func (s *RegistrySuite) TestErrorCall(t sweet.T) {
	var (
		r  = NewRegistry()
		ex = errors.New("utoh")
	)

	r.Configure("test")

	err := r.Call("test", func(ctx context.Context) error {
		return ex
	}, nil)

	Expect(err).To(Equal(ex))
}

func (s *RegistrySuite) TestErrorCallWithFallback(t sweet.T) {
	var (
		r      = NewRegistry()
		called = false
		ex     = errors.New("utoh")
	)

	r.Configure("test")

	err := r.Call("test", func(ctx context.Context) error {
		return ex
	}, func(err error) error {
		Expect(err).To(Equal(ex))
		called = true
		return nil
	})

	Expect(err).To(BeNil())
	Expect(called).To(BeTrue())
}

func (s *RegistrySuite) TestFallbackError(t sweet.T) {
	var (
		r    = NewRegistry()
		err1 = errors.New("utoh 1")
		err2 = errors.New("utoh 2")
	)

	r.Configure("test")

	err := r.Call("test", func(ctx context.Context) error {
		return err1
	}, func(err error) error {
		Expect(err).To(Equal(err1))
		return err2
	})

	Expect(err).To(Equal(err2))
}

func (s *RegistrySuite) TestBreaker(t sweet.T) {
	var (
		r         = NewRegistry()
		callCount = 0
	)

	r.Configure("test", testConfig())

	fallback := func(err error) error {
		callCount++
		return nil
	}

	for i := 0; i < 5; i++ {
		Expect(r.Call("test", errFunc, fallback)).To(BeNil())
	}

	Expect(r.Call("test", errFunc, nil)).To(Equal(ErrCircuitOpen))
	Expect(r.Call("test", errFunc, fallback)).To(BeNil())
	Expect(callCount).To(Equal(6))
}

func (s *RegistrySuite) TestConcurrency(t sweet.T) {
	var (
		r       = NewRegistry()
		started = make(chan struct{}) // Signals start of f
		block   = make(chan error, 5) // Blocks inside f
		wg      = sync.WaitGroup{}    // Signals end of f
	)

	defer close(started)

	f := func() BreakerFunc {
		return func(ctx context.Context) error {
			defer wg.Done()
			started <- struct{}{}
			return <-block
		}
	}

	r.Configure(
		"test",
		testConfig(),
		WithMaxConcurrency(5),
		WithMaxConcurrencyTimeout(0),
	)

	wg.Add(5)
	r.CallAsync("test", f(), nil)
	r.CallAsync("test", f(), nil)
	r.CallAsync("test", f(), nil)
	r.CallAsync("test", f(), nil)
	r.CallAsync("test", f(), nil)

	for i := 0; i < 5; i++ {
		<-started
	}

	Expect(r.Call("test", nilFunc, func(err error) error {
		Expect(err).To(Equal(ErrMaxConcurrency))
		return nil
	})).To(BeNil())

	close(block)
	wg.Wait()

	Expect(r.Call("test", nilFunc, nil)).To(BeNil())
}

func (s *RegistrySuite) TestConcurrencyUnblocked(t sweet.T) {
	var (
		r       = NewRegistry()
		started = make(chan struct{}) // Signals start of f
		block   = make(chan error, 5) // Blocks inside f
		wg      = sync.WaitGroup{}    // Signals end of f
		result  = make(chan error)
	)

	defer close(started)

	f := func() BreakerFunc {
		return func(ctx context.Context) error {
			defer wg.Done()
			started <- struct{}{}
			return <-block
		}
	}

	r.Configure(
		"test",
		testConfig(),
		WithMaxConcurrency(5),
		WithMaxConcurrencyTimeout(time.Minute),
	)

	wg.Add(5)
	r.CallAsync("test", f(), nil)
	r.CallAsync("test", f(), nil)
	r.CallAsync("test", f(), nil)
	r.CallAsync("test", f(), nil)
	r.CallAsync("test", f(), nil)

	for i := 0; i < 5; i++ {
		<-started
	}

	go func() {
		defer close(result)

		result <- r.Call("test", nilFunc, func(err error) error {
			Expect(err).To(Equal(ErrMaxConcurrency))
			return nil
		})
	}()

	Consistently(result).ShouldNot(Receive())
	close(block)
	Eventually(result).Should(Receive(BeNil()))
}

func (s *RegistrySuite) TestConcurrencyTimeout(t sweet.T) {
	var (
		clock   = glock.NewMockClock()
		r       = newRegistryWithClock(clock)
		started = make(chan struct{}) // Signals start of f
		block   = make(chan error, 5) // Blocks inside f
		wg      = sync.WaitGroup{}    // Signals end of f
		result  = make(chan error)
	)

	defer close(started)

	f := func() BreakerFunc {
		return func(ctx context.Context) error {
			defer wg.Done()
			started <- struct{}{}
			return <-block
		}
	}

	r.Configure(
		"test",
		testConfig(),
		WithMaxConcurrency(5),
		WithMaxConcurrencyTimeout(time.Minute),
		withClock(clock),
	)

	wg.Add(5)
	r.CallAsync("test", f(), nil)
	r.CallAsync("test", f(), nil)
	r.CallAsync("test", f(), nil)
	r.CallAsync("test", f(), nil)
	r.CallAsync("test", f(), nil)

	for i := 0; i < 5; i++ {
		<-started
	}

	go func() {
		defer close(result)

		result <- r.Call("test", nilFunc, func(err error) error {
			Expect(err).To(Equal(ErrMaxConcurrency))
			return nil
		})
	}()

	Consistently(result).ShouldNot(Receive())
	clock.Advance(time.Minute)
	Eventually(result).Should(Receive(BeNil()))
	close(block)
}

func (s *RegistrySuite) TestDoubleConfigure(t sweet.T) {
	r := NewRegistry()
	Expect(r.Configure("test")).To(BeNil())
	Expect(r.Configure("test")).To(Equal(ErrAlreadyConfigured))
}

func (s *RegistrySuite) TestCallUnconfigured(t sweet.T) {
	Expect(NewRegistry().Call("test", nilFunc, nil)).To(Equal(ErrBreakerUnconfigured))
}

func (s *RegistrySuite) TestCallAsyncUnconfigured(t sweet.T) {
	ch := NewRegistry().CallAsync("test", nilFunc, nil)
	Eventually(ch).Should(Receive(Equal(ErrBreakerUnconfigured)))
	Eventually(ch).Should(BeClosed())
}
