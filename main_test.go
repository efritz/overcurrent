package overcurrent

import (
	"testing"
	"time"

	"github.com/aphistic/sweet"
	. "github.com/onsi/gomega"
)

func Test(t *testing.T) {
	sweet.T(func(s *sweet.S) {
		RegisterFailHandler(sweet.GomegaFail)

		s.RunSuite(t, &TripSuite{})
		s.RunSuite(t, &FailureSuite{})
		s.RunSuite(t, &BreakerSuite{})
	})
}

//
//
//

type mockClock struct {
	ch        <-chan time.Time
	now       time.Time
	afterArgs []time.Duration
}

func newMockClock(ch <-chan time.Time) *mockClock {
	return &mockClock{
		ch:  ch,
		now: time.Now(),
	}
}

func (m *mockClock) Now() time.Time {
	return m.now
}

func (m *mockClock) After(duration time.Duration) <-chan time.Time {
	m.afterArgs = append(m.afterArgs, duration)
	return m.ch
}

func (m *mockClock) advance(d time.Duration) {
	m.now = m.now.Add(d)
}
