package overcurrent

import (
	"fmt"
	"time"

	"github.com/efritz/glock"
)

var (
	// ErrInvocationTimeout occurs when the method takes too long to execute.
	ErrInvocationTimeout = fmt.Errorf("invocation has timed out")
)

func callWithTimeout(f func() error, clock glock.Clock, timeout time.Duration) error {
	if timeout == 0 {
		return f()
	}

	select {
	case err := <-callWithResultChan(f):
		return err

	case <-clock.After(timeout):
		return ErrInvocationTimeout
	}
}

func callWithResultChan(f func() error) <-chan error {
	ch := make(chan error)

	go func() {
		ch <- f()
		close(ch)
	}()

	return ch
}
