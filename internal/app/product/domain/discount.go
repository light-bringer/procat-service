package domain

import (
	"fmt"
	"math/big"
	"time"
)

// Discount represents a time-bound percentage discount on a product.
type Discount struct {
	percentage int64      // 0-100
	startDate  time.Time
	endDate    time.Time
}

// NewDiscount creates a new Discount with validation.
func NewDiscount(percentage int64, startDate, endDate time.Time) (*Discount, error) {
	if percentage < 0 || percentage > 100 {
		return nil, fmt.Errorf("discount percentage must be between 0 and 100, got %d", percentage)
	}

	if endDate.Before(startDate) {
		return nil, ErrInvalidDiscountPeriod
	}

	return &Discount{
		percentage: percentage,
		startDate:  startDate,
		endDate:    endDate,
	}, nil
}

// Percentage returns the discount percentage.
func (d *Discount) Percentage() int64 {
	return d.percentage
}

// StartDate returns the discount start date.
func (d *Discount) StartDate() time.Time {
	return d.startDate
}

// EndDate returns the discount end date.
func (d *Discount) EndDate() time.Time {
	return d.endDate
}

// IsValidAt checks if the discount is valid at the given time.
func (d *Discount) IsValidAt(t time.Time) bool {
	return !t.Before(d.startDate) && !t.After(d.endDate)
}

// Apply applies the discount to a price and returns the discounted price.
// Formula: discountedPrice = price - (price * percentage / 100)
func (d *Discount) Apply(price *Money) *Money {
	// Calculate discount amount: price * (percentage / 100)
	discountRat := big.NewRat(d.percentage, 100)
	discountAmount := price.MultiplyByRat(discountRat)

	// Return: price - discountAmount
	return price.Subtract(discountAmount)
}

// CalculateDiscountAmount calculates the discount amount (not the final price).
func (d *Discount) CalculateDiscountAmount(price *Money) *Money {
	discountRat := big.NewRat(d.percentage, 100)
	return price.MultiplyByRat(discountRat)
}
