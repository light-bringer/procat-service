package domain

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewMoney(t *testing.T) {
	t.Run("valid money creation", func(t *testing.T) {
		m, err := NewMoney(100, 1)
		require.NoError(t, err)
		assert.Equal(t, int64(100), m.Numerator())
		assert.Equal(t, int64(1), m.Denominator())
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
		assert.Equal(t, int64(100), normalized.Numerator())
		assert.Equal(t, int64(1), normalized.Denominator())
	})

	t.Run("normalizes complex fraction", func(t *testing.T) {
		// 300/6 should normalize to 50/1
		m, _ := NewMoney(300, 6)
		normalized := m.Normalize()
		assert.Equal(t, int64(50), normalized.Numerator())
		assert.Equal(t, int64(1), normalized.Denominator())
	})

	t.Run("already normalized value unchanged", func(t *testing.T) {
		// 100/1 should stay 100/1
		m, _ := NewMoney(100, 1)
		normalized := m.Normalize()
		assert.Equal(t, int64(100), normalized.Numerator())
		assert.Equal(t, int64(1), normalized.Denominator())
	})

	t.Run("normalizes negative numerator correctly", func(t *testing.T) {
		// -200/2 should normalize to -100/1
		m, _ := NewMoney(-200, 2)
		normalized := m.Normalize()
		assert.Equal(t, int64(-100), normalized.Numerator())
		assert.Equal(t, int64(1), normalized.Denominator())
	})

	t.Run("normalizes fractional prices", func(t *testing.T) {
		// 249900/100 should normalize to 2499/1
		m, _ := NewMoney(249900, 100)
		normalized := m.Normalize()
		assert.Equal(t, int64(2499), normalized.Numerator())
		assert.Equal(t, int64(1), normalized.Denominator())
	})

	t.Run("preserves value equality after normalization", func(t *testing.T) {
		// Different representations of the same value should equal after normalization
		m1, _ := NewMoney(200, 2)    // 100
		m2, _ := NewMoney(400, 4)    // 100

		normalized1 := m1.Normalize()
		normalized2 := m2.Normalize()

		assert.True(t, normalized1.Equals(normalized2))
		assert.Equal(t, normalized1.Numerator(), normalized2.Numerator())
		assert.Equal(t, normalized1.Denominator(), normalized2.Denominator())
	})
}
