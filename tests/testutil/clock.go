package testutil

import (
	"time"

	"github.com/light-bringer/procat-service/internal/pkg/clock"
)

// NewFixedClock creates a mock clock fixed at the given time.
func NewFixedClock(t time.Time) clock.Clock {
	return clock.NewMockClock(t)
}

// NewMockClock creates a mock clock that can be controlled in tests.
func NewMockClock() *clock.MockClock {
	return clock.NewMockClock(time.Now())
}
