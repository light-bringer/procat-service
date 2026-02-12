package domain

import (
	"math/big"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewMoney(t *testing.T) {
	t.Run("valid money creation", func(t *testing.T) {
		m, err := NewMoney(100, 1)
		require.NoError(t, err)
		num, err := m.Numerator()
		require.NoError(t, err)
		denom, err := m.Denominator()
		require.NoError(t, err)
		assert.Equal(t, int64(100), num)
		assert.Equal(t, int64(1), denom)
	})

	t.Run("zero denominator returns error", func(t *testing.T) {
		_, err := NewMoney(100, 0)
		assert.Error(t, err)
	})

	t.Run("negative denominator returns error", func(t *testing.T) {
		_, err := NewMoney(100, -1)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "positive")
	})

	t.Run("negative numerator allowed", func(t *testing.T) {
		m, err := NewMoney(-100, 1)
		require.NoError(t, err)
		assert.True(t, m.IsNegative())
	})
}

func TestMoney_Add(t *testing.T) {
	m1, _ := NewMoney(100, 1) // 100
	m2, _ := NewMoney(50, 1)  // 50

	result := m1.Add(m2)
	val, _ := result.Float64()
	assert.Equal(t, 150.0, val)
}

func TestMoney_Subtract(t *testing.T) {
	m1, _ := NewMoney(100, 1) // 100
	m2, _ := NewMoney(30, 1)  // 30

	result := m1.Subtract(m2)
	val, _ := result.Float64()
	assert.Equal(t, 70.0, val)
}

func TestMoney_Multiply(t *testing.T) {
	m1, _ := NewMoney(100, 1)  // 100
	m2, _ := NewMoney(3, 2)    // 1.5

	result := m1.Multiply(m2)
	val, _ := result.Float64()
	assert.Equal(t, 150.0, val)
}

func TestMoney_Divide(t *testing.T) {
	t.Run("valid division", func(t *testing.T) {
		m1, _ := NewMoney(100, 1) // 100
		m2, _ := NewMoney(2, 1)   // 2

		result, err := m1.Divide(m2)
		require.NoError(t, err)
		val, _ := result.Float64()
		assert.Equal(t, 50.0, val)
	})

	t.Run("division by zero returns error", func(t *testing.T) {
		m1, _ := NewMoney(100, 1)
		m2, _ := NewMoney(0, 1)

		_, err := m1.Divide(m2)
		assert.Error(t, err)
	})
}

func TestMoney_Comparisons(t *testing.T) {
	m1, _ := NewMoney(100, 1)
	m2, _ := NewMoney(50, 1)
	m3, _ := NewMoney(100, 1)

	assert.True(t, m1.GreaterThan(m2))
	assert.False(t, m2.GreaterThan(m1))

	assert.True(t, m2.LessThan(m1))
	assert.False(t, m1.LessThan(m2))

	assert.True(t, m1.Equals(m3))
	assert.False(t, m1.Equals(m2))
}

func TestMoney_Precision(t *testing.T) {
	// Test precise decimal arithmetic - no floating point errors
	m1, _ := NewMoney(249900, 100) // $2499.00
	discount, _ := NewMoney(20, 100) // 20%

	discountAmount := m1.MultiplyByRat(discount.rat)
	finalPrice := m1.Subtract(discountAmount)

	// 2499.00 - (2499.00 * 0.20) = 2499.00 - 499.80 = 1999.20
	assert.Equal(t, "1999.20", finalPrice.String())
}

func TestMoney_Normalize(t *testing.T) {
	t.Run("reduces fraction to lowest terms", func(t *testing.T) {
		// 200/2 should normalize to 100/1
		m, _ := NewMoney(200, 2)
		normalized := m.Normalize()
		num, err := normalized.Numerator()
		require.NoError(t, err)
		denom, err := normalized.Denominator()
		require.NoError(t, err)
		assert.Equal(t, int64(100), num)
		assert.Equal(t, int64(1), denom)
	})

	t.Run("normalizes complex fraction", func(t *testing.T) {
		// 300/6 should normalize to 50/1
		m, _ := NewMoney(300, 6)
		normalized := m.Normalize()
		num, err := normalized.Numerator()
		require.NoError(t, err)
		denom, err := normalized.Denominator()
		require.NoError(t, err)
		assert.Equal(t, int64(50), num)
		assert.Equal(t, int64(1), denom)
	})

	t.Run("already normalized value unchanged", func(t *testing.T) {
		// 100/1 should stay 100/1
		m, _ := NewMoney(100, 1)
		normalized := m.Normalize()
		num, err := normalized.Numerator()
		require.NoError(t, err)
		denom, err := normalized.Denominator()
		require.NoError(t, err)
		assert.Equal(t, int64(100), num)
		assert.Equal(t, int64(1), denom)
	})

	t.Run("normalizes negative numerator correctly", func(t *testing.T) {
		// -200/2 should normalize to -100/1
		m, _ := NewMoney(-200, 2)
		normalized := m.Normalize()
		num, err := normalized.Numerator()
		require.NoError(t, err)
		denom, err := normalized.Denominator()
		require.NoError(t, err)
		assert.Equal(t, int64(-100), num)
		assert.Equal(t, int64(1), denom)
	})

	t.Run("normalizes fractional prices", func(t *testing.T) {
		// 249900/100 should normalize to 2499/1
		m, _ := NewMoney(249900, 100)
		normalized := m.Normalize()
		num, err := normalized.Numerator()
		require.NoError(t, err)
		denom, err := normalized.Denominator()
		require.NoError(t, err)
		assert.Equal(t, int64(2499), num)
		assert.Equal(t, int64(1), denom)
	})

	t.Run("preserves value equality after normalization", func(t *testing.T) {
		// Different representations of the same value should equal after normalization
		m1, _ := NewMoney(200, 2)    // 100
		m2, _ := NewMoney(400, 4)    // 100

		normalized1 := m1.Normalize()
		normalized2 := m2.Normalize()

		assert.True(t, normalized1.Equals(normalized2))
		num1, err := normalized1.Numerator()
		require.NoError(t, err)
		num2, err := normalized2.Numerator()
		require.NoError(t, err)
		denom1, err := normalized1.Denominator()
		require.NoError(t, err)
		denom2, err := normalized2.Denominator()
		require.NoError(t, err)
		assert.Equal(t, num1, num2)
		assert.Equal(t, denom1, denom2)
	})
}

func TestMoney_EdgeCases(t *testing.T) {
	t.Run("very large price - near MaxInt64", func(t *testing.T) {
		// MaxInt64 = 9223372036854775807
		// Test with a very large but safe value
		largePrice, err := NewMoney(9223372036854775807, 100)
		require.NoError(t, err)

		// Verify it can be stored and retrieved
		num, err := largePrice.Numerator()
		require.NoError(t, err)
		denom, err := largePrice.Denominator()
		require.NoError(t, err)
		assert.Equal(t, int64(9223372036854775807), num)
		assert.Equal(t, int64(100), denom)

		// Verify it can be normalized
		normalized := largePrice.Normalize()
		assert.NotNil(t, normalized)
	})

	t.Run("very small price - fractional cents", func(t *testing.T) {
		// $0.01
		smallPrice, err := NewMoney(1, 100)
		require.NoError(t, err)

		val, _ := smallPrice.Float64()
		assert.Equal(t, 0.01, val)

		// Even smaller - $0.001
		tinyPrice, err := NewMoney(1, 1000)
		require.NoError(t, err)

		val, _ = tinyPrice.Float64()
		assert.InDelta(t, 0.001, val, 0.0001)
	})

	t.Run("fractional cent handling", func(t *testing.T) {
		// $10.001 - fractional cent
		price, err := NewMoney(10001, 1000)
		require.NoError(t, err)

		val, _ := price.Float64()
		assert.InDelta(t, 10.001, val, 0.00001)

		// Operations should preserve precision
		price2, _ := NewMoney(5000, 1000)
		result := price.Add(price2)

		resultVal, _ := result.Float64()
		assert.InDelta(t, 15.001, resultVal, 0.00001)
	})

	t.Run("multiple operations preserve precision", func(t *testing.T) {
		// Start with $100.00
		price, _ := NewMoney(10000, 100)

		// Apply 20% discount
		discount, _ := NewMoney(20, 100)
		discountAmount := price.MultiplyByRat(discount.rat)
		afterDiscount := price.Subtract(discountAmount)

		// Result should be $80.00
		val, _ := afterDiscount.Float64()
		assert.Equal(t, 80.0, val)

		// Apply another 10% discount
		discount2, _ := NewMoney(10, 100)
		discountAmount2 := afterDiscount.MultiplyByRat(discount2.rat)
		final := afterDiscount.Subtract(discountAmount2)

		// Result should be $72.00 (80 - 8)
		finalVal, _ := final.Float64()
		assert.Equal(t, 72.0, finalVal)
	})

	t.Run("Float64 precision indicator", func(t *testing.T) {
		// Exact representation
		exactPrice, _ := NewMoney(100, 1)
		val, exact := exactPrice.Float64()
		assert.Equal(t, 100.0, val)
		assert.True(t, exact, "100/1 should have exact float representation")

		// Non-exact representation (repeating decimal)
		nonExactPrice, _ := NewMoney(1, 3)
		val, exact = nonExactPrice.Float64()
		assert.InDelta(t, 0.333333, val, 0.00001)
		assert.False(t, exact, "1/3 should not have exact float representation")
	})

	t.Run("zero value operations", func(t *testing.T) {
		zero, _ := NewMoney(0, 1)
		price, _ := NewMoney(100, 1)

		// Adding zero
		result := price.Add(zero)
		assert.True(t, result.Equals(price))

		// Subtracting zero
		result = price.Subtract(zero)
		assert.True(t, result.Equals(price))

		// Multiplying by zero
		result = price.Multiply(zero)
		assert.True(t, result.IsZero())
	})

	t.Run("comparison edge cases", func(t *testing.T) {
		// Very close but different values
		price1, _ := NewMoney(10000, 100) // 100.00
		price2, _ := NewMoney(10001, 100) // 100.01

		assert.True(t, price2.GreaterThan(price1))
		assert.True(t, price1.LessThan(price2))
		assert.False(t, price1.Equals(price2))

		// Same value, different representations
		price3, _ := NewMoney(100, 1)
		price4, _ := NewMoney(200, 2)
		assert.True(t, price3.Equals(price4))
	})

	t.Run("copy independence", func(t *testing.T) {
		original, _ := NewMoney(100, 1)
		copied := original.Copy()

		// Verify they're equal
		assert.True(t, original.Equals(copied))

		// Modify copy
		newPrice, _ := NewMoney(50, 1)
		modified := copied.Add(newPrice)

		// Original should be unchanged
		origVal, _ := original.Float64()
		assert.Equal(t, 100.0, origVal)

		// Modified should be different
		modVal, _ := modified.Float64()
		assert.Equal(t, 150.0, modVal)
	})
}

func TestDiscount_EdgeCases(t *testing.T) {
	now := time.Now().UTC()

	t.Run("zero percent discount returns original price", func(t *testing.T) {
		discount, err := NewDiscount(0, now, now.Add(24*time.Hour))
		require.NoError(t, err)

		price, _ := NewMoney(10000, 100) // $100.00
		discounted := discount.Apply(price)

		val, _ := discounted.Float64()
		assert.Equal(t, 100.0, val)
	})

	t.Run("100 percent discount returns zero price", func(t *testing.T) {
		discount, err := NewDiscount(100, now, now.Add(24*time.Hour))
		require.NoError(t, err)

		price, _ := NewMoney(10000, 100) // $100.00
		discounted := discount.Apply(price)

		assert.True(t, discounted.IsZero())
		val, _ := discounted.Float64()
		assert.Equal(t, 0.0, val)
	})

	t.Run("discount on very small price", func(t *testing.T) {
		discount, err := NewDiscount(50, now, now.Add(24*time.Hour))
		require.NoError(t, err)

		// $0.02
		tinyPrice, _ := NewMoney(2, 100)
		discounted := discount.Apply(tinyPrice)

		val, _ := discounted.Float64()
		assert.Equal(t, 0.01, val) // 50% of $0.02 = $0.01
	})

	t.Run("discount precision with fractional results", func(t *testing.T) {
		// 33% discount
		discount, err := NewDiscount(33, now, now.Add(24*time.Hour))
		require.NoError(t, err)

		price, _ := NewMoney(10000, 100) // $100.00
		discounted := discount.Apply(price)

		// $100 * 0.67 = $67.00
		val, _ := discounted.Float64()
		assert.InDelta(t, 67.0, val, 0.01)
	})

	t.Run("multiple discount applications", func(t *testing.T) {
		discount1, err := NewDiscount(20, now, now.Add(24*time.Hour))
		require.NoError(t, err)

		discount2, err := NewDiscount(10, now, now.Add(24*time.Hour))
		require.NoError(t, err)

		price, _ := NewMoney(10000, 100) // $100.00

		// Apply 20% discount: $100 * 0.80 = $80
		afterFirst := discount1.Apply(price)
		val1, _ := afterFirst.Float64()
		assert.Equal(t, 80.0, val1)

		// Apply 10% discount to $80: $80 * 0.90 = $72
		afterSecond := discount2.Apply(afterFirst)
		val2, _ := afterSecond.Float64()
		assert.Equal(t, 72.0, val2)
	})
}

func TestMoney_Overflow(t *testing.T) {
	t.Run("numerator overflow detection", func(t *testing.T) {
		// Create a value where numerator exceeds int64 after operations
		// 2^100 is way larger than MaxInt64 (2^63 - 1)
		huge := new(big.Int).Lsh(big.NewInt(1), 100) // 2^100
		hugeRat := new(big.Rat).SetInt(huge)
		m := NewMoneyFromRat(hugeRat)

		_, err := m.Numerator()
		assert.ErrorIs(t, err, ErrMoneyOverflow, "should detect numerator overflow")
		assert.False(t, m.IsSafeForStorage(), "should not be safe for storage")
	})

	t.Run("denominator overflow detection", func(t *testing.T) {
		// Create a value where denominator exceeds int64
		// 1 / 2^100
		hugeDenom := new(big.Int).Lsh(big.NewInt(1), 100)
		rat := new(big.Rat).SetFrac(big.NewInt(1), hugeDenom)
		m := NewMoneyFromRat(rat)

		_, err := m.Denominator()
		assert.ErrorIs(t, err, ErrMoneyOverflow, "should detect denominator overflow")
		assert.False(t, m.IsSafeForStorage(), "should not be safe for storage")
	})

	t.Run("safe values within int64 bounds", func(t *testing.T) {
		// MaxInt64 = 9223372036854775807
		safeValue, err := NewMoney(9223372036854775807, 1)
		require.NoError(t, err)

		num, err := safeValue.Numerator()
		require.NoError(t, err)
		assert.Equal(t, int64(9223372036854775807), num)

		denom, err := safeValue.Denominator()
		require.NoError(t, err)
		assert.Equal(t, int64(1), denom)

		assert.True(t, safeValue.IsSafeForStorage(), "should be safe for storage")
	})

	t.Run("operations that cause overflow", func(t *testing.T) {
		// Start with MaxInt64 and multiply by a large factor
		maxInt64, _ := NewMoney(9223372036854775807, 1)
		largeFactor := new(big.Rat).SetInt64(1000000)

		result := maxInt64.MultiplyByRat(largeFactor)

		// Result exceeds int64 bounds
		_, err := result.Numerator()
		assert.ErrorIs(t, err, ErrMoneyOverflow)
		assert.False(t, result.IsSafeForStorage())
	})

	t.Run("normalized value still overflows", func(t *testing.T) {
		// Even after normalization, if the value is too large, it should still overflow
		huge := new(big.Int).Lsh(big.NewInt(1), 100)
		hugeRat := new(big.Rat).SetInt(huge)
		m := NewMoneyFromRat(hugeRat)

		normalized := m.Normalize()
		_, err := normalized.Numerator()
		assert.ErrorIs(t, err, ErrMoneyOverflow)
		assert.False(t, normalized.IsSafeForStorage())
	})
}
