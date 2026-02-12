package domain

import (
	"fmt"
	"math/big"
	"time"
)

// Discount represents a time-bound percentage discount on a product.
// Supports fractional percentages (e.g., 12.5%, 7.25%) for flexible pricing.
// Uses *big.Rat internally for precise arithmetic.
type Discount struct {
	percentage         *big.Rat // 0.0-100.0, stored as rational for precision
	startDate          time.Time
	endDate            time.Time
	discountMultiplier *big.Rat // Cached percentage/100 for performance
}

// NewDiscount creates a new Discount with validation.
// All dates must be in UTC timezone to prevent ambiguity across distributed systems.
// Discount duration is limited to 2 years maximum for business policy compliance.
// Percentage supports fractional values (e.g., 12.5 for 12.5% discount).
func NewDiscount(percentageFloat float64, startDate, endDate time.Time) (*Discount, error) {
	if percentageFloat < 0 || percentageFloat > 100 {
		return nil, fmt.Errorf("discount percentage must be between 0 and 100, got %.2f", percentageFloat)
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

	// Limit discount duration to 2 years (extract to const per code review recommendation)
	const MaxDiscountDuration = 2 * 365 * 24 * time.Hour
	if endDate.Sub(startDate) > MaxDiscountDuration {
		return nil, fmt.Errorf("discount duration cannot exceed 2 years")
	}

	// Convert percentage to *big.Rat for precise arithmetic
	percentage := new(big.Rat).SetFloat64(percentageFloat)

	// Pre-calculate discount multiplier for performance (avoids allocation on every Apply())
	hundred := big.NewRat(100, 1)
	discountMultiplier := new(big.Rat).Quo(percentage, hundred)

	return &Discount{
		percentage:         percentage,
		startDate:          startDate,
		endDate:            endDate,
		discountMultiplier: discountMultiplier,
	}, nil
}

// Percentage returns the discount percentage as float64 for external interfaces.
// For precise internal calculations, use PercentageRat() instead.
func (d *Discount) Percentage() float64 {
	f, _ := d.percentage.Float64()
	return f
}

// PercentageRat returns the discount percentage as *big.Rat for precise calculations.
func (d *Discount) PercentageRat() *big.Rat {
	return new(big.Rat).Set(d.percentage)
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
//
// Example: If endDate is 2024-12-31 23:59:59.999999999 UTC, the discount is valid through that nanosecond.
func (d *Discount) IsValidAt(t time.Time) bool {
	return !t.Before(d.startDate) && !t.After(d.endDate)
}

// Multiplier returns the cached discount multiplier (percentage/100).
// Exposed for use by domain services.
func (d *Discount) Multiplier() *big.Rat {
	return d.discountMultiplier
}

// Apply applies the discount to a price and returns the discounted price.
// Formula: discountedPrice = price - (price * percentage / 100)
// Delegates to PricingCalculator for centralized pricing logic.
func (d *Discount) Apply(price *Money) *Money {
	return defaultPricingCalculator.ApplyDiscount(price, d.discountMultiplier)
}

// CalculateDiscountAmount calculates the discount amount (not the final price).
// Delegates to PricingCalculator for centralized pricing logic.
func (d *Discount) CalculateDiscountAmount(price *Money) *Money {
	return defaultPricingCalculator.CalculateDiscountAmount(price, d.discountMultiplier)
}
