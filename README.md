# Overcurrent

[![GoDoc](https://godoc.org/github.com/efritz/overcurrent?status.svg)](https://godoc.org/github.com/efritz/overcurrent)
[![Build Status](https://secure.travis-ci.org/efritz/overcurrent.png)](http://travis-ci.org/efritz/overcurrent)
[![codecov.io](http://codecov.io/github/efritz/overcurrent/coverage.svg?branch=master)](http://codecov.io/github/efritz/overcurrent?branch=master)
[![Go Report Card](https://goreportcard.com/badge/github.com/efritz/overcurrent)](https://goreportcard.com/report/github.com/efritz/overcurrent)

Go library for protecting function calls via circuit breaker pattern.

The circuit breaker pattern can prevent an application from repeatedly trying to
execute an operation that is likely to fail. If the problem appears to have been
rectified, the application can attempt to invoke the operation. This is useful
in a distributed environment where an application accesses remote resources and
services. It is possible (and likely at scale) for these operations to fail due
to transient faults such as:

- timeouts
- slow network connections
- resource or service being overcommitted
- resource or service being temporarily unavailable

These faults typically correct themselves after a short period of time. You can
read more about the pattern
[here](https://msdn.microsoft.com/en-us/library/dn589784.aspx) and
[here](http://martinfowler.com/bliki/CircuitBreaker.html).

This library depends on the [backoff](https://github.com/efritz/backoff) library,
which defines structures for creating backoff interval generator.

## Example

First, create a circuit breaker instance with any of the following configuration
functions supplied. A circuit breaker can be created with all default parameters.

The `InvocationTimeout` specifies how long a protected function can
run for before returning an error (zero allows for unbounded runtime).

The `ResetBackoff` specifies how long the circuit breaker stays in the open state
until transitioning to the half-closed state. If a failure occurs while in the
half-closed state, the circuit breaker will transition back to the open state, and
the time spent before transitioning to the half-closed state may increase, depending
on the implementation of the backoff interface.

The `HalfClosedRetryProbability` specifies the probability that a request in the
half-closed state will attempt to retry instead of immediately returning a
`CircuitOpenError`.

```go
breaker := NewCircuitBreaker(
	WithInvocationTimeout(50 * time.Millisecond),
	WithResetBackoff(backoff.NewConstantBackoff(1 * time.Second)),
	WithHalfClosedRetryProbability(0.1),
	WithFailureInterpreter(NewAnyErrorFailureInterpreter()),
	WithTripCondition(NewConsecutiveFailureTripCondition(5)),
)
```

The `FailureInterpreter` determines which errors returned from the protected
function should count as a failure with respect to the circuit breaker. As an
example, the following failure interpreter will only count HTTP 500 errors as
a circuit error (given an omitted definition of the type ProtocolError).

```go
type ServerErrorFailureInterpreter struct{}

func (fi *ServerErrorFailureInterpreter) ShouldTrip(err error) bool {
	if perr, ok := err.(ProtocolError); ok {
		return perr.StatusCode >= 500
	}

	return false
}
```

The `TripCondition` determines, based on recent failure history, when the
breaker should trip. This interface can be customized to trip after a number
of failures in a row, number of failures in a given time span, fail rate, etc.

The breaker can be explicitly tripped and reset via the `Trip` and `Reset` methods.
If a breaker is manually tripped, then it will remain in open state until it is
manually reset (it will never transition to the half-closed state).

### Function API

To use the breaker, simply pass the function that attempts to access a resource
to the `Call` method of the breaker. This method may attempt to call the passed
function. The breaker will return a custom error if the circuit is closed or if
the function is invoked but takes too long to complete. If the function is called
and completes (successfully or unsuccessfully), the method returns the error that
the function returns.

```go
err := breaker.Call(func(ctx context.Context) error {
	req, err := http.NewRequest("GET", "http://example.com", nil)
	if err != nil {
		return err
	}

	// Wrap in context passed to function so that the request
	// is canceled if the function exceeds the invocation max.
	resp, err := http.DefaultClient.Do(r.WithContext(ctx))
	if err != nil {
		return err
	}

	// Process HTTP response
	return nil
})

if err == nil {
	// Success
} else if err == ErrInvocationTimeout {
	// Took too long
} else if err == ErrCircuitOpen {
	// Not attempted, in failure mode
} else {
	// Unsuccessful, error is HTTP error
}
```

*Design Choice:* The protected function is given to the breaker as a parameter
to each invocation of `Call`, as opposed to begin registered with the circuit
breaker at initialization. This is to increase the flexibility of the API so
the input to the function can easily change on each invocation. It is **not**
advised that several disparate functions are passed to the same breaker -
failures from one function will influence the other in ways that are not
intuitive.

A breaker also has a `CallAsync` function that does the work of `Call` in a
separate goroutine and returns a channnel that receives an error value then
immediately closes.

### Registry

A *registry* is a collection of breakers which can be invoked by a unique name.
In order to use a breaker in the registry, it must first be configured. Any of
the options used to configure a breaker can be used along with two new options:

- max concurrency: the maximum number of goroutines which may execute the
breaker function concurrently
- max concurrency timeout: how long you are willing to wait while the breaker
function is at max concurrency

```go
registry := NewRegistry()

registry.Configure(
	"redis-cache",
	WithFailureInterpreter(NewAnyErrorFailureInterpreter()),
	WithTripCondition(NewConsecutiveFailureTripCondition(5)),
	// ...
	WithMaxConcurrency(50),
	WithMaxConcurrencyTimeout(time.Second),
)
```

To use a breaker, invoke the `Call` method of the registry with the name of
the breaker to use. You can also pass a second nillable *fallback function*
which is invoked when the breaker function fails (or fails to be called due
to the circuit state or the concurrency state of the breaker).

```go
registry.Call("redis-cache", func(ctx context.Context) error {
	// get value from redis
}, func(err error) error {
	// do some canned action
	return nil
})
```

Symmetrically to the breaker, a `CallAsync` method is also available with the
same semantics as `Call`.

### Non-Function API

Sometimes a chunk of code which should be protected by the circuit breaker is
not easily contained in a single function. In these cases, the methods used by
the `Call` method are exposed for manual use. First, the `ShouldTry` method can
be queried to get the current state of the circuit breaker.

```go
if breaker.ShouldTry() {
	// Try to execute the protected section
}
```

Then, in the location where an attempt has been made, the result can be applied
to the circuit breaker manually. Special effort must be made to ensure that *all*
code paths mark a result with the breaker; otherwise, the trip condition may set
the circuit state erroneously.

```go
if success {
	// A nil error is successful
	breaker.MarkResult(nil)
} else {
	// Mark whatever error is accepted by the registered failure interpreter
	breaker.MarkResult(ErrBadAttempt)
}
```

## Metric Collectors

### Hystrix

Overcurrent comes with a metric collector compatible with the Netflix Hystrix
[Dashboard](https://github.com/Netflix/Hystrix/wiki/Dashboard). For testing purposes,
a stand-alone dashboard can be used via Docker (see `docker-compose.yml` for details).
Below is a minimal example setting up the Hystrix collector.

```go
import (
	"github.com/efritz/overcurrent"
	"github.com/efritz/overcurrent/hystrix"
)

func main() {
	hystrixCollector := hystrix.NewCollector()
	hystrixCollector.Start()
	defer hystrixCollector.Stop()

	registry := overcurrent.NewRegistry()
	registry.Configure(
		"name",
		overcurrent.WithCollector(overcurrent.NamedCollector("name", hystrixCollector)),
	)

	// ...

	http.ListenAndServe("0.0.0.0:8080", hystrixCollector.Handler())
}
```

## License

Copyright (c) 2016 Eric Fritz

Permission is hereby granted, free of charge, to any person obtaining a copy
of this software and associated documentation files (the "Software"), to deal
in the Software without restriction, including without limitation the rights
to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
copies of the Software, and to permit persons to whom the Software is
furnished to do so, subject to the following conditions:

The above copyright notice and this permission notice shall be included in
all copies or substantial portions of the Software.

THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN
THE SOFTWARE.
