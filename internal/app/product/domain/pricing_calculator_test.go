package domain

import (
	"math/big"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPricingCalculator_CalculateDiscountAmount(t *testing.T) {
	pc := NewPricingCalculator()

	t.Run("calculates discount amount correctly", func(t *testing.T) {
		price, _ := NewMoney(10000, 100)            // $100.00
		multiplier := new(big.Rat).SetFloat64(0.20) // 20%

		discountAmount := pc.CalculateDiscountAmount(price, multiplier)

		val, _ := discountAmount.Float64()
		assert.Equal(t, 20.0, val) // 20% of $100 = $20
	})

	t.Run("zero multiplier returns zero discount", func(t *testing.T) {
		price, _ := NewMoney(10000, 100)
		multiplier := new(big.Rat).SetFloat64(0.0)

		discountAmount := pc.CalculateDiscountAmount(price, multiplier)

		assert.True(t, discountAmount.IsZero())
	})

	t.Run("fractional discount percentage", func(t *testing.T) {
		price, _ := NewMoney(10000, 100)             // $100.00
		multiplier := new(big.Rat).SetFloat64(0.125) // 12.5%

		discountAmount := pc.CalculateDiscountAmount(price, multiplier)

		val, _ := discountAmount.Float64()
		assert.Equal(t, 12.5, val) // 12.5% of $100 = $12.50
	})
}

func TestPricingCalculator_ApplyDiscount(t *testing.T) {
	pc := NewPricingCalculator()

	t.Run("applies discount correctly", func(t *testing.T) {
		price, _ := NewMoney(10000, 100)            // $100.00
		multiplier := new(big.Rat).SetFloat64(0.20) // 20%

		finalPrice := pc.ApplyDiscount(price, multiplier)

		val, _ := finalPrice.Float64()
		assert.Equal(t, 80.0, val) // $100 - ($100 * 0.20) = $80
	})

	t.Run("100% discount returns zero", func(t *testing.T) {
		price, _ := NewMoney(10000, 100)
		multiplier := new(big.Rat).SetFloat64(1.0) // 100%

		finalPrice := pc.ApplyDiscount(price, multiplier)

		assert.True(t, finalPrice.IsZero())
	})

	t.Run("zero discount returns original price", func(t *testing.T) {
		price, _ := NewMoney(10000, 100)
		multiplier := new(big.Rat).SetFloat64(0.0)

		finalPrice := pc.ApplyDiscount(price, multiplier)

		assert.True(t, finalPrice.Equals(price))
	})
}

func TestPricingCalculator_CalculateEffectivePrice(t *testing.T) {
	pc := NewPricingCalculator()
	now := time.Now().UTC()

	t.Run("applies valid discount", func(t *testing.T) {
		basePrice, _ := NewMoney(10000, 100) // $100.00
		discount, err := NewDiscount(20, now.Add(-1*time.Hour), now.Add(1*time.Hour))
		require.NoError(t, err)

		effectivePrice := pc.CalculateEffectivePrice(basePrice, discount, now)

		val, _ := effectivePrice.Float64()
		assert.Equal(t, 80.0, val) // 20% off $100 = $80
	})

	t.Run("ignores expired discount", func(t *testing.T) {
		basePrice, _ := NewMoney(10000, 100)
		// Discount expired 2 hours ago
		discount, err := NewDiscount(20, now.Add(-3*time.Hour), now.Add(-1*time.Hour))
		require.NoError(t, err)

		effectivePrice := pc.CalculateEffectivePrice(basePrice, discount, now)

		// Should return base price since discount is expired
		assert.True(t, effectivePrice.Equals(basePrice))
	})

	t.Run("ignores future discount", func(t *testing.T) {
		basePrice, _ := NewMoney(10000, 100)
		// Discount starts in 1 hour
		discount, err := NewDiscount(20, now.Add(1*time.Hour), now.Add(3*time.Hour))
		require.NoError(t, err)

		effectivePrice := pc.CalculateEffectivePrice(basePrice, discount, now)

		// Should return base price since discount hasn't started
		assert.True(t, effectivePrice.Equals(basePrice))
	})

	t.Run("returns base price when no discount", func(t *testing.T) {
		basePrice, _ := NewMoney(10000, 100)

		effectivePrice := pc.CalculateEffectivePrice(basePrice, nil, now)

		assert.True(t, effectivePrice.Equals(basePrice))
	})

	t.Run("preserves precision with fractional discount", func(t *testing.T) {
		basePrice, _ := NewMoney(249900, 100) // $2499.00
		discount, err := NewDiscount(12.5, now.Add(-1*time.Hour), now.Add(1*time.Hour))
		require.NoError(t, err)

		effectivePrice := pc.CalculateEffectivePrice(basePrice, discount, now)

		// 12.5% of $2499 = $312.375
		// $2499 - $312.375 = $2186.625
		val, _ := effectivePrice.Float64()
		assert.InDelta(t, 2186.625, val, 0.001)
	})
}

func TestPricingCalculator_EdgeCases(t *testing.T) {
	pc := NewPricingCalculator()
	now := time.Now().UTC()

	t.Run("very small price with discount", func(t *testing.T) {
		basePrice, _ := NewMoney(1, 100) // $0.01
		discount, err := NewDiscount(50, now.Add(-1*time.Hour), now.Add(1*time.Hour))
		require.NoError(t, err)

		effectivePrice := pc.CalculateEffectivePrice(basePrice, discount, now)

		val, _ := effectivePrice.Float64()
		assert.Equal(t, 0.005, val) // 50% of $0.01 = $0.005
	})

	t.Run("multiple discount applications preserve precision", func(t *testing.T) {
		basePrice, _ := NewMoney(10000, 100) // $100.00

		// Apply 20% discount
		multiplier1 := new(big.Rat).SetFloat64(0.20)
		afterFirst := pc.ApplyDiscount(basePrice, multiplier1)
		val1, _ := afterFirst.Float64()
		assert.Equal(t, 80.0, val1)

		// Apply another 10% discount on the discounted price
		multiplier2 := new(big.Rat).SetFloat64(0.10)
		afterSecond := pc.ApplyDiscount(afterFirst, multiplier2)
		val2, _ := afterSecond.Float64()
		assert.Equal(t, 72.0, val2) // 80 - (80 * 0.10) = 72
	})
}
