package overcurrent

import "time"

type (
	clock interface {
		Now() time.Time
		// After(duration time.Duration) <-chan time.Time
	}

	realClock struct{}
)

func (rc *realClock) Now() time.Time {
	return time.Now()
}

// func (rc *realClock) After(duration time.Duration) <-chan time.Time {
// 	return time.After(duration)
// }
