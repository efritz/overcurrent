# Overcurrent

[![GoDoc](https://godoc.org/github.com/efritz/overcurrent?status.svg)](https://godoc.org/github.com/efritz/overcurrent)
[![Build Status](https://secure.travis-ci.org/efritz/overcurrent.png)](http://travis-ci.org/efritz/overcurrent)
[![codecov.io](http://codecov.io/github/efritz/overcurrent/coverage.svg?branch=master)](http://codecov.io/github/efritz/overcurrent?branch=master)

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

These faults typically correct themselves after a short period of time.

## Example

First, create an instance of a `Breaker` with the following customizable config
parameters. The `InvocationTimeout` specifies how long a protected function can
run for before returning an error (zero allows for unbounded runtime). The
`ResetTimeout` specifies how long the circuit breaker stays in the open state
until transitioning to the half-closed state. The `HalfClosedRetryProbability`
specifies the probability that a request in the half-closed state will attempt
to retry instead of immediately returning a `CircuitOpenError`.

```go
breaker := NewBreaker(BreakerConfig {
	return BreakerConfig{
		InvocationTimeout:          50 * time.Milliecond,
		ResetTimeout:               1 * time.Second,
		HalfClosedRetryProbability: 0.1,
		FailureInterpreter:         NewAnyErrorFailureInterpreter(),
		TripCondition:              NewConsecutiveFailureTripCondition(5),
	})
```

The `FailureInterpreter` determines which errors returned from the protected
function should count as a failure with respect to the circuit breaker. As an
example, the following failure interpreter will only count HTTP 500 errors as
a circuit error.

```go
type ServerErrorFailureInterpreter struct{}

func (fi *ServerErrorFailureInterpreter) ShouldTrip(err error) bool {
	return err >= http.StatusInternalServerError
}
```

The `TripCondition` determines, based on recent failure history, when the
breaker should trip. This interface can be customized to trip after a number
of failures in a row, number of failures in a given time span, fail rate, etc.

To use the breaker, simply pass the function that attempts to access a resource
to the `Call` method of the breaker. This method may attempt to call the passed
function. The breaker will return a custom error if the circuit is closed or if
the function is invoked but takes too long to complete. If the function is called
and completes (successfully or unsuccessfully), the method returns the error that
the function returns.

```go
err := breaker.Call(func() error {
	resp, err := http.Get("http://example.com")
	if err != nil {
		return err
	}

	// Process HTTP response
	return nil
})

if err == nil {
	// Success
} else if err == InvocationTimeoutError {
	// Took too long
} else if err == CircuitOpenError {
	// Not attempted, in failure mode
} else {
	// Unsuccessful, error is HTTP error
}
```

*Design Choice:* The protected function is given to the breaker as a parameter
to each invocation of `Call`, as opposed to begin registered with the circuit
breaker at initialization. This is to increase the flexibility of the API so
the input to the function can easily change on each invocation. It is **not**
advised that several disparate functions are passed to the same breaker - failures
from one function will influence the other in ways that are not intuitive.

The breaker can also be explicitly tripped and reset. If a breaker is manually
tripped, then it will remain in open state until it is manually reset (it will
never transition to the half-closed state).

```go
breaker.Trip()
```

```go
breaker.Reset()
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
