package overcurrent

func toErrChan(f func() error) <-chan error {
	ch := make(chan error, 1)

	go func() {
		ch <- f()
		close(ch)
	}()

	return ch
}
