package overcurrent

import (
	"time"

	"github.com/aphistic/sweet"
	"github.com/efritz/glock"
	. "github.com/onsi/gomega"
)

type SemaphoreSuite struct{}

func (s *SemaphoreSuite) TestWaitSignal(t sweet.T) {
	var (
		clock     = glock.NewMockClock()
		semaphore = newSemaphore(clock, 10)
		sync      = make(chan struct{})
	)

	for i := 0; i < 10; i++ {
		Expect(semaphore.wait(time.Second)).To(BeTrue())
	}

	go func() {
		defer close(sync)
		semaphore.wait(time.Second)
	}()

	Consistently(sync).ShouldNot(Receive())
	semaphore.signal()
	Eventually(sync).Should(BeClosed())

	for i := 0; i < 10; i++ {
		semaphore.signal()
	}

	for i := 0; i < 10; i++ {
		Expect(semaphore.wait(time.Second)).To(BeTrue())
	}
}

func (s *SemaphoreSuite) TestWaitTimeout(t sweet.T) {
	var (
		clock     = glock.NewMockClock()
		semaphore = newSemaphore(clock, 10)
		value     = make(chan bool)
	)

	for i := 0; i < 10; i++ {
		Expect(semaphore.wait(time.Second)).To(BeTrue())
	}

	go func() {
		defer close(value)
		value <- semaphore.wait(time.Minute)
	}()

	Consistently(value).ShouldNot(Receive())
	clock.BlockingAdvance(time.Minute)
	Eventually(value).Should(Receive(BeFalse()))
}

func (s *SemaphoreSuite) TestNoWait(t sweet.T) {
	var (
		clock     = glock.NewMockClock()
		semaphore = newSemaphore(clock, 3)
	)

	Expect(semaphore.wait(0)).To(BeTrue())
	Expect(semaphore.wait(0)).To(BeTrue())
	Expect(semaphore.wait(0)).To(BeTrue())
	Expect(semaphore.wait(0)).To(BeFalse())

	semaphore.signal()
	Expect(semaphore.wait(0)).To(BeTrue())
	Expect(semaphore.wait(0)).To(BeFalse())
}
