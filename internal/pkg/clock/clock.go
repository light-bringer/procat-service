package clock

import "time"

// Clock is an interface for time operations to enable testability.
type Clock interface {
	Now() time.Time
}

// RealClock is the production implementation using actual system time.
type RealClock struct{}

// NewRealClock creates a new RealClock.
func NewRealClock() Clock {
	return &RealClock{}
}

// Now returns the current system time.
func (c *RealClock) Now() time.Time {
	return time.Now()
}

// MockClock is a test implementation that allows setting the current time.
type MockClock struct {
	current time.Time
}

// NewMockClock creates a new MockClock starting at the given time.
func NewMockClock(startTime time.Time) *MockClock {
	return &MockClock{current: startTime}
}

// Now returns the mock current time.
func (m *MockClock) Now() time.Time {
	return m.current
}

// Set sets the mock current time.
func (m *MockClock) Set(t time.Time) {
	m.current = t
}

// Advance advances the mock clock by the given duration.
func (m *MockClock) Advance(d time.Duration) {
	m.current = m.current.Add(d)
}
