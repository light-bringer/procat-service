package domain

import (
	"fmt"
	"math/big"
)

// Money represents a monetary value with precise decimal arithmetic using big.Rat.
// It stores the value as a rational number (numerator/denominator) to avoid floating-point precision issues.
type Money struct {
	rat *big.Rat
}

// NewMoney creates a new Money instance from numerator and denominator.
// Example: NewMoney(249900, 100) represents $2499.00
func NewMoney(numerator, denominator int64) (*Money, error) {
	if denominator == 0 {
		return nil, fmt.Errorf("denominator cannot be zero")
	}

	rat := big.NewRat(numerator, denominator)
	return &Money{rat: rat}, nil
}

// NewMoneyFromRat creates a new Money instance from a big.Rat.
func NewMoneyFromRat(rat *big.Rat) *Money {
	if rat == nil {
		return &Money{rat: big.NewRat(0, 1)}
	}
	return &Money{rat: new(big.Rat).Set(rat)}
}

// Numerator returns the numerator of the rational number.
func (m *Money) Numerator() int64 {
	return m.rat.Num().Int64()
}

// Denominator returns the denominator of the rational number.
func (m *Money) Denominator() int64 {
	return m.rat.Denom().Int64()
}

// Add adds two Money values and returns a new Money instance.
func (m *Money) Add(other *Money) *Money {
	result := new(big.Rat).Add(m.rat, other.rat)
	return &Money{rat: result}
}

// Subtract subtracts another Money value from this one and returns a new Money instance.
func (m *Money) Subtract(other *Money) *Money {
	result := new(big.Rat).Sub(m.rat, other.rat)
	return &Money{rat: result}
}

// Multiply multiplies this Money value by another and returns a new Money instance.
func (m *Money) Multiply(other *Money) *Money {
	result := new(big.Rat).Mul(m.rat, other.rat)
	return &Money{rat: result}
}

// MultiplyByRat multiplies this Money value by a rational number and returns a new Money instance.
func (m *Money) MultiplyByRat(rat *big.Rat) *Money {
	result := new(big.Rat).Mul(m.rat, rat)
	return &Money{rat: result}
}

// Divide divides this Money value by another and returns a new Money instance.
func (m *Money) Divide(other *Money) (*Money, error) {
	if other.rat.Sign() == 0 {
		return nil, fmt.Errorf("cannot divide by zero")
	}
	result := new(big.Rat).Quo(m.rat, other.rat)
	return &Money{rat: result}, nil
}

// IsZero returns true if the money value is zero.
func (m *Money) IsZero() bool {
	return m.rat.Sign() == 0
}

// IsNegative returns true if the money value is negative.
func (m *Money) IsNegative() bool {
	return m.rat.Sign() < 0
}

// IsPositive returns true if the money value is positive.
func (m *Money) IsPositive() bool {
	return m.rat.Sign() > 0
}

// LessThan returns true if this Money value is less than another.
func (m *Money) LessThan(other *Money) bool {
	return m.rat.Cmp(other.rat) < 0
}

// GreaterThan returns true if this Money value is greater than another.
func (m *Money) GreaterThan(other *Money) bool {
	return m.rat.Cmp(other.rat) > 0
}

// Equals returns true if this Money value equals another.
func (m *Money) Equals(other *Money) bool {
	return m.rat.Cmp(other.rat) == 0
}

// Float64 returns an approximate float64 representation (for display only, not calculations).
func (m *Money) Float64() float64 {
	f, _ := m.rat.Float64()
	return f
}

// String returns a string representation of the money value.
func (m *Money) String() string {
	return m.rat.FloatString(2)
}

// Copy creates a deep copy of this Money instance.
func (m *Money) Copy() *Money {
	return &Money{rat: new(big.Rat).Set(m.rat)}
}
