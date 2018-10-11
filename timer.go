package clock

import (
	"time"
)

type mockTimer struct {
	c       chan time.Time
	release chan bool

	lock   chan struct{}
	clock  *MockClock
	active bool
	target time.Time
}

var _ Timer = new(mockTimer)

func (m *mockTimer) setInactive() {
	// If a release was sent in the meantime, that means a new timer
	// was started or that we already stopped manually
	select {
	case m.lock <- struct{}{}:
		defer func() { <-m.lock }()
	case <-m.release:
		return
	}
	m.active = false
}

func (m *mockTimer) wait(now time.Time) {
	select {
	case <-m.clock.after(now, m.target.Sub(now)):
		m.setInactive()
		m.c <- m.clock.Now()
	case <-m.release:
	}
}

func (m *mockTimer) Chan() <-chan time.Time {
	return m.c
}

func (m *mockTimer) Reset(d time.Duration) bool {
	var wasActive bool
	m.lock <- struct{}{}
	defer func() { <-m.lock }()

	now := m.clock.Now()
	wasActive, m.active = m.active, true
	m.target = now.Add(d)

	if wasActive {
		m.release <- true
	}
	go m.wait(now)

	return wasActive
}

func (m *mockTimer) Stop() bool {
	var wasActive bool
	m.lock <- struct{}{}
	defer func() { <-m.lock }()

	wasActive, m.active = m.active, false
	if wasActive {
		m.release <- true
	}

	return wasActive
}

// NewMockTimer creates a new Timer using the provided Clock. You should not use this
// directly outside of unit tests; use Clock.NewTimer().
func NewMockTimer(c *MockClock) Timer {
	return &mockTimer{
		c:       make(chan time.Time, 1),
		release: make(chan bool),
		lock:    make(chan struct{}, 1),
		clock:   c,
	}
}
