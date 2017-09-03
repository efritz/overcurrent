package overcurrent

import (
	"errors"
	"sync"

	"github.com/efritz/glock"
)

type (
	Registry interface {
		// Configure will register a new breaker instance under the given name using
		// the given configuration. A breaker config may not be changed after being
		// initialized. It is an error to register the same breaker twice, or try to
		// invoke Call or CallAsync with an unregistered breaker.
		Configure(name string, configs ...BreakerConfig) error

		// Call will invoke `Call` on the breaker configured with the given name. If
		// the breaker returns a non-nil error, the fallback function is invoked with
		// the error as the value. It may be the case that the fallback function is
		// invoked without the breaker function failing (e.g. circuit open).
		Call(name string, f BreakerFunc, fallback FallbackFunc) error

		// CallAsync will create a channel that receives the error value from an similar
		// invocation of Call. See the Breaker docs for more details.
		CallAsync(name string, f BreakerFunc, fallback FallbackFunc) <-chan error
	}

	registry struct {
		breakers map[string]*wrappedBreaker
		mutex    *sync.RWMutex
		clock    glock.Clock
	}

	wrappedBreaker struct {
		breaker   *circuitBreaker
		semaphore *semaphore
	}

	FallbackFunc func(error) error
)

var (
	ErrAlreadyConfigured   = errors.New("breaker is already configured")
	ErrBreakerUnconfigured = errors.New("breaker not configured")
	ErrMaxConcurrency      = errors.New("breaker is at max concurrency")
)

func NewRegistry() Registry {
	return newRegistryWithClock(glock.NewMockClock())
}

func newRegistryWithClock(clock glock.Clock) Registry {
	return &registry{
		breakers: map[string]*wrappedBreaker{},
		mutex:    &sync.RWMutex{},
		clock:    clock,
	}
}

func (r *registry) Configure(name string, configs ...BreakerConfig) error {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	if _, ok := r.breakers[name]; ok {
		return ErrAlreadyConfigured
	}

	breaker := newCircuitBreaker(configs...)

	r.breakers[name] = &wrappedBreaker{
		breaker:   breaker,
		semaphore: newSemaphore(r.clock, breaker.maxConcurrency),
	}

	return nil
}

func (r *registry) Call(name string, f BreakerFunc, fallback FallbackFunc) error {
	wrapped, err := r.getWrappedBreaker(name)
	if err != nil {
		return err
	}

	if wrapped.semaphore.wait(wrapped.breaker.maxConcurrencyTimeout) {
		err = func() error {
			// Ensure we don't eat capacity from the semaphore
			// if f panics (but we also don't want to hold the
			// semaphore while we execute the fallback func).
			defer wrapped.semaphore.signal()

			return wrapped.breaker.Call(f)
		}()
	} else {
		err = ErrMaxConcurrency
	}

	return tryFallback(err, fallback)
}

func (r *registry) CallAsync(name string, f BreakerFunc, fallback FallbackFunc) <-chan error {
	return toErrChan(func() error { return r.Call(name, f, fallback) })
}

func (r *registry) getWrappedBreaker(name string) (*wrappedBreaker, error) {
	r.mutex.RLock()
	defer r.mutex.RUnlock()

	breaker, ok := r.breakers[name]
	if !ok {
		return nil, ErrBreakerUnconfigured
	}

	return breaker, nil
}

func tryFallback(err error, fallback FallbackFunc) error {
	if err != nil && fallback != nil {
		return fallback(err)
	}

	return err
}
