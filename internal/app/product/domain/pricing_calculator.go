package domain

import (
	"math/big"
	"time"
)

// PricingCalculator is a domain service for price and discount calculations.
//
// ARCHITECTURE NOTE:
// This service centralizes pricing logic and is used by domain objects (Discount, Product)
// to perform calculations. Benefits:
//   1. Single source of truth for pricing formulas
//   2. Easier to test pricing logic in isolation
//   3. Simplifies future pricing strategy changes
//   4. Domain objects delegate to this service for all pricing operations
type PricingCalculator struct{}

// NewPricingCalculator creates a new PricingCalculator instance.
func NewPricingCalculator() *PricingCalculator {
	return &PricingCalculator{}
}

// Package-level calculator instance for domain object use
var defaultPricingCalculator = NewPricingCalculator()

// CalculateDiscountAmount calculates the discount amount (not the final price).
// Formula: discountAmount = price * discountMultiplier
func (pc *PricingCalculator) CalculateDiscountAmount(price *Money, discountMultiplier *big.Rat) *Money {
	return price.MultiplyByRat(discountMultiplier)
}

// ApplyDiscount applies a discount to a price and returns the final price.
// Formula: finalPrice = price - (price * discountMultiplier)
func (pc *PricingCalculator) ApplyDiscount(price *Money, discountMultiplier *big.Rat) *Money {
	discountAmount := pc.CalculateDiscountAmount(price, discountMultiplier)
	return price.Subtract(discountAmount)
}

// CalculateEffectivePrice calculates the effective price considering time-bound discounts.
// Returns the discounted price if the discount is valid at the given time, otherwise returns the base price.
func (pc *PricingCalculator) CalculateEffectivePrice(basePrice *Money, discount *Discount, now time.Time) *Money {
	if discount != nil && discount.IsValidAt(now) {
		return pc.ApplyDiscount(basePrice, discount.Multiplier())
	}
	return basePrice.Copy()
}

// Multiplier extracts the discount multiplier from a Discount.
// This is a helper method that bridges the Discount value object with the PricingCalculator.
func (pc *PricingCalculator) Multiplier(discount *Discount) *big.Rat {
	return discount.Multiplier()
}
