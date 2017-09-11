package overcurrent

import (
	"errors"
	"sync"
	"time"

	"github.com/efritz/glock"
)

type (
	Registry interface {
		// Configure will register a new breaker instance under the given name using
		// the given configuration. A breaker's configuration may not be changed after
		// being initialized. It is an error to register the same breaker twice, or try
		// to invoke Call or CallAsync with an unregistered breaker.
		Configure(name string, configs ...BreakerConfigFunc) error

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
	return newRegistryWithClock(glock.NewRealClock())
}

func newRegistryWithClock(clock glock.Clock) Registry {
	return &registry{
		breakers: map[string]*wrappedBreaker{},
		mutex:    &sync.RWMutex{},
		clock:    clock,
	}
}

func (r *registry) Configure(name string, configs ...BreakerConfigFunc) error {
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
	wrapped, collector, err := r.getWrappedBreaker(name)
	if err != nil {
		return err
	}

	start := time.Now()
	err = r.call(wrapped, collector, f, fallback)
	elapsed := time.Now().Sub(start)

	collector.ReportDuration(EventTypeTotalDuration, elapsed)
	return err
}

func (r *registry) CallAsync(name string, f BreakerFunc, fallback FallbackFunc) <-chan error {
	return toErrChan(func() error {
		return r.Call(name, f, fallback)
	})
}

func (r *registry) getWrappedBreaker(name string) (*wrappedBreaker, MetricCollector, error) {
	r.mutex.RLock()
	defer r.mutex.RUnlock()

	wrapped, ok := r.breakers[name]
	if !ok {
		return nil, nil, ErrBreakerUnconfigured
	}

	return wrapped, wrapped.breaker.collector, nil
}

func (r *registry) call(wrapped *wrappedBreaker, collector MetricCollector, f BreakerFunc, fallback FallbackFunc) error {
	collector.ReportCount(EventTypeAttempt)

	err := r.callWithSemaphore(wrapped.breaker, wrapped.semaphore, f)
	if err == nil {
		collector.ReportCount(EventTypeSuccess)
		return nil
	}

	collector.ReportCount(EventTypeFailure)

	if err == ErrMaxConcurrency {
		collector.ReportCount(EventTypeRejection)
	}

	if fallback == nil {
		return err
	}

	if err := fallback(err); err != nil {
		collector.ReportCount(EventTypeFallbackFailure)
		return err
	}

	collector.ReportCount(EventTypeFallbackSuccess)
	return nil
}

func (r *registry) callWithSemaphore(breaker *circuitBreaker, semaphore *semaphore, f BreakerFunc) error {
	if !semaphore.wait(breaker.maxConcurrencyTimeout, breaker.collector) {
		return ErrMaxConcurrency
	}

	defer func() {
		breaker.collector.ReportCount(EventTypeSemaphoreReleased)
		semaphore.signal()
	}()

	breaker.collector.ReportCount(EventTypeSemaphoreAcquired)
	return breaker.Call(f)
}
