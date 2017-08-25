package overcurrent

import (
	"context"
	"errors"

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
	}, func() error {
		called = true
		return nil
	})

	Expect(err).To(BeNil())
	Expect(called).To(BeTrue())
}

func (s *RegistrySuite) TestFallbackError(t sweet.T) {
	var (
		r   = NewRegistry()
		ex1 = errors.New("utoh 1")
		ex2 = errors.New("utoh 2")
	)

	r.Configure("test")

	err := r.Call("test", func(ctx context.Context) error {
		return ex1
	}, func() error {
		return ex2
	})

	Expect(err).To(Equal(ex2))
}

func (s *RegistrySuite) TestBreaker(t sweet.T) {
	var (
		r         = NewRegistry()
		callCount = 0
	)

	r.Configure("test", testConfig())

	fallback := func() error {
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
