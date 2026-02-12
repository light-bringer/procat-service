package domain

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewDiscount(t *testing.T) {
	startDate := time.Now()
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
	startDate := time.Now()
	endDate := startDate.Add(24 * time.Hour)
	discount, _ := NewDiscount(20, startDate, endDate) // 20% off

	price, _ := NewMoney(100, 1) // $100

	discountedPrice := discount.Apply(price)

	// $100 - ($100 * 20%) = $100 - $20 = $80
	val, _ := discountedPrice.Float64()
	assert.Equal(t, 80.0, val)
}
