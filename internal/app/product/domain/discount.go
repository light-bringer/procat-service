package domain

import (
	"fmt"
	"math/big"
	"time"
)

// Discount represents a time-bound percentage discount on a product.
type Discount struct {
	percentage         int64      // 0-100
	startDate          time.Time
	endDate            time.Time
	discountMultiplier *big.Rat   // Cached percentage/100 for performance
}

// NewDiscount creates a new Discount with validation.
// All dates must be in UTC timezone to prevent ambiguity across distributed systems.
// Discount duration is limited to 2 years maximum for business policy compliance.
func NewDiscount(percentage int64, startDate, endDate time.Time) (*Discount, error) {
	if percentage < 0 || percentage > 100 {
		return nil, fmt.Errorf("discount percentage must be between 0 and 100, got %d", percentage)
	}

	// Require UTC timezone for consistency
	if startDate.Location() != time.UTC {
		return nil, fmt.Errorf("discount start date must be in UTC timezone")
	}
	if endDate.Location() != time.UTC {
		return nil, fmt.Errorf("discount end date must be in UTC timezone")
	}

	if endDate.Before(startDate) || endDate.Equal(startDate) {
		return nil, ErrInvalidDiscountPeriod
	}

	// Limit discount duration to 2 years
	maxDuration := 2 * 365 * 24 * time.Hour // 2 years
	if endDate.Sub(startDate) > maxDuration {
		return nil, fmt.Errorf("discount duration cannot exceed 2 years")
	}

	// Pre-calculate discount multiplier for performance (avoids allocation on every Apply())
	discountMultiplier := big.NewRat(percentage, 100)

	return &Discount{
		percentage:         percentage,
		startDate:          startDate,
		endDate:            endDate,
		discountMultiplier: discountMultiplier,
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
// The discount period is INCLUSIVE on both ends:
//   - startDate: Valid from startDate onwards (t >= startDate)
//   - endDate: Valid through endDate including the entire day (t <= endDate)
// Example: If endDate is 2024-12-31 23:59:59.999999999 UTC, the discount is valid through that nanosecond.
func (d *Discount) IsValidAt(t time.Time) bool {
	return !t.Before(d.startDate) && !t.After(d.endDate)
}

// Apply applies the discount to a price and returns the discounted price.
// Formula: discountedPrice = price - (price * percentage / 100)
// Uses cached discount multiplier for performance.
func (d *Discount) Apply(price *Money) *Money {
	// Calculate discount amount using cached multiplier
	discountAmount := price.MultiplyByRat(d.discountMultiplier)

	// Return: price - discountAmount
	return price.Subtract(discountAmount)
}

// CalculateDiscountAmount calculates the discount amount (not the final price).
// Uses cached discount multiplier for performance.
func (d *Discount) CalculateDiscountAmount(price *Money) *Money {
	return price.MultiplyByRat(d.discountMultiplier)
}
