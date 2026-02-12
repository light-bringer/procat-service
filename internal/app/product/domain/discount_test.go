package domain

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewDiscount(t *testing.T) {
	startDate := time.Now().UTC()
	endDate := startDate.Add(24 * time.Hour)

	t.Run("valid discount", func(t *testing.T) {
		d, err := NewDiscount(20, startDate, endDate)
		require.NoError(t, err)
		assert.Equal(t, int64(20), d.Percentage())
	})

	t.Run("percentage below 0 returns error", func(t *testing.T) {
		_, err := NewDiscount(-1, startDate, endDate)
		assert.Error(t, err)
	})

	t.Run("percentage above 100 returns error", func(t *testing.T) {
		_, err := NewDiscount(101, startDate, endDate)
		assert.Error(t, err)
	})

	t.Run("end date before start date returns error", func(t *testing.T) {
		_, err := NewDiscount(20, endDate, startDate)
		assert.ErrorIs(t, err, ErrInvalidDiscountPeriod)
	})

	t.Run("end date equal to start date returns error", func(t *testing.T) {
		_, err := NewDiscount(20, startDate, startDate)
		assert.ErrorIs(t, err, ErrInvalidDiscountPeriod)
	})

	t.Run("non-UTC start date returns error", func(t *testing.T) {
		est, _ := time.LoadLocation("America/New_York")
		nonUTCStart := time.Now().In(est)
		nonUTCEnd := nonUTCStart.Add(24 * time.Hour)
		_, err := NewDiscount(20, nonUTCStart, nonUTCEnd)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "UTC")
	})

	t.Run("non-UTC end date returns error", func(t *testing.T) {
		utcStart := time.Now().UTC()
		est, _ := time.LoadLocation("America/New_York")
		nonUTCEnd := time.Now().In(est).Add(24 * time.Hour)
		_, err := NewDiscount(20, utcStart, nonUTCEnd)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "UTC")
	})

	t.Run("duration exceeding 2 years returns error", func(t *testing.T) {
		start := time.Now().UTC()
		twoYearsPlus := start.Add((2*365 + 1) * 24 * time.Hour)
		_, err := NewDiscount(20, start, twoYearsPlus)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "2 years")
	})

	t.Run("duration of exactly 2 years is allowed", func(t *testing.T) {
		start := time.Now().UTC()
		exactlyTwoYears := start.Add(2 * 365 * 24 * time.Hour)
		_, err := NewDiscount(20, start, exactlyTwoYears)
		assert.NoError(t, err)
	})
}

func TestDiscount_IsValidAt(t *testing.T) {
	startDate := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
	endDate := time.Date(2025, 12, 31, 23, 59, 59, 0, time.UTC)
	d, _ := NewDiscount(20, startDate, endDate)

	t.Run("valid during period", func(t *testing.T) {
		checkTime := time.Date(2025, 6, 15, 12, 0, 0, 0, time.UTC)
		assert.True(t, d.IsValidAt(checkTime))
	})

	t.Run("invalid before start", func(t *testing.T) {
		checkTime := time.Date(2024, 12, 31, 23, 59, 59, 0, time.UTC)
		assert.False(t, d.IsValidAt(checkTime))
	})

	t.Run("invalid after end", func(t *testing.T) {
		checkTime := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
		assert.False(t, d.IsValidAt(checkTime))
	})

	t.Run("valid on start date", func(t *testing.T) {
		assert.True(t, d.IsValidAt(startDate))
	})

	t.Run("valid on end date", func(t *testing.T) {
		assert.True(t, d.IsValidAt(endDate))
	})
}

func TestDiscount_Apply(t *testing.T) {
	startDate := time.Now().UTC()
	endDate := startDate.Add(24 * time.Hour)
	discount, _ := NewDiscount(20, startDate, endDate) // 20% off

	price, _ := NewMoney(100, 1) // $100

	discountedPrice := discount.Apply(price)

	// $100 - ($100 * 20%) = $100 - $20 = $80
	val, _ := discountedPrice.Float64()
	assert.Equal(t, 80.0, val)
}

func TestDiscount_IsValidAt_Boundaries(t *testing.T) {
	startDate := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
	endDate := time.Date(2025, 12, 31, 23, 59, 59, 999999999, time.UTC)
	d, _ := NewDiscount(20, startDate, endDate)

	t.Run("valid at exact start nanosecond", func(t *testing.T) {
		assert.True(t, d.IsValidAt(startDate))
	})

	t.Run("valid at exact end nanosecond", func(t *testing.T) {
		assert.True(t, d.IsValidAt(endDate))
	})

	t.Run("invalid one nanosecond before start", func(t *testing.T) {
		beforeStart := startDate.Add(-1 * time.Nanosecond)
		assert.False(t, d.IsValidAt(beforeStart))
	})

	t.Run("invalid one nanosecond after end", func(t *testing.T) {
		afterEnd := endDate.Add(1 * time.Nanosecond)
		assert.False(t, d.IsValidAt(afterEnd))
	})
}
