package services

import (
	"time"

	"github.com/light-bringer/procat-service/internal/app/product/domain"
)

// PricingCalculator is a domain service for calculating product prices with discount logic.
type PricingCalculator struct{}

// NewPricingCalculator creates a new PricingCalculator.
func NewPricingCalculator() *PricingCalculator {
	return &PricingCalculator{}
}

// CalculatePrice calculates the effective price for a product at a given time.
// It applies the discount if one is active at the specified time.
func (pc *PricingCalculator) CalculatePrice(product *domain.Product, at time.Time) *domain.Money {
	basePrice := product.BasePrice()

	discount := product.Discount()
	if discount == nil {
		return basePrice
	}

	if !discount.IsValidAt(at) {
		return basePrice
	}

	return discount.Apply(basePrice)
}

// CalculateDiscountAmount calculates the discount amount (not the final price) at a given time.
func (pc *PricingCalculator) CalculateDiscountAmount(product *domain.Product, at time.Time) *domain.Money {
	discount := product.Discount()
	if discount == nil || !discount.IsValidAt(at) {
		zero, _ := domain.NewMoney(0, 1)
		return zero
	}

	return discount.CalculateDiscountAmount(product.BasePrice())
}

// HasActiveDiscount returns true if the product has an active discount at the given time.
func (pc *PricingCalculator) HasActiveDiscount(product *domain.Product, at time.Time) bool {
	discount := product.Discount()
	return discount != nil && discount.IsValidAt(at)
}
