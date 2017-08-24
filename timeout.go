package overcurrent

import (
	"context"
	"fmt"
	"time"

	"github.com/efritz/glock"
)

var (
	// ErrInvocationTimeout occurs when the method takes too long to execute.
	ErrInvocationTimeout = fmt.Errorf("invocation has timed out")
)

func callWithTimeout(f BreakerFunc, clock glock.Clock, timeout time.Duration) error {
	if timeout == 0 {
		return f(context.Background())
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	select {
	case err := <-callWithResultChan(f, ctx):
		return err

	case <-clock.After(timeout):
		return ErrInvocationTimeout
	}
}

func callWithResultChan(f BreakerFunc, ctx context.Context) <-chan error {
	ch := make(chan error, 1)

	go func() {
		defer close(ch)
		ch <- f(ctx)
	}()

	return ch
}
