package services

import (
	"math/big"
	"time"

	"github.com/light-bringer/procat-service/internal/app/product/domain"
)

// PricingCalculator is a domain service for price and discount calculations.
// Extracts pricing logic for better separation of concerns and easier testing.
type PricingCalculator struct{}

// NewPricingCalculator creates a new PricingCalculator instance.
func NewPricingCalculator() *PricingCalculator {
	return &PricingCalculator{}
}

// CalculateDiscountAmount calculates the discount amount (not the final price).
// Formula: discountAmount = price * discountMultiplier
func (pc *PricingCalculator) CalculateDiscountAmount(price *domain.Money, discountMultiplier *big.Rat) *domain.Money {
	return price.MultiplyByRat(discountMultiplier)
}

// ApplyDiscount applies a discount to a price and returns the final price.
// Formula: finalPrice = price - (price * discountMultiplier)
func (pc *PricingCalculator) ApplyDiscount(price *domain.Money, discountMultiplier *big.Rat) *domain.Money {
	discountAmount := pc.CalculateDiscountAmount(price, discountMultiplier)
	return price.Subtract(discountAmount)
}

// CalculateEffectivePrice calculates the effective price considering time-bound discounts.
// Returns the discounted price if the discount is valid at the given time, otherwise returns the base price.
func (pc *PricingCalculator) CalculateEffectivePrice(basePrice *domain.Money, discount *domain.Discount, now time.Time) *domain.Money {
	if discount != nil && discount.IsValidAt(now) {
		return pc.ApplyDiscount(basePrice, discount.Multiplier())
	}
	return basePrice.Copy()
}

// Multiplier extracts the discount multiplier from a Discount.
// This is a helper method that bridges the Discount value object with the PricingCalculator.
func (pc *PricingCalculator) Multiplier(discount *domain.Discount) *big.Rat {
	return discount.Multiplier()
}
