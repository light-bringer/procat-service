package e2e

import (
	"github.com/light-bringer/procat-service/internal/app/product/domain"
	"github.com/light-bringer/procat-service/internal/app/product/usecases/create_product"
)

// ProductBuilder helps create products for tests with a fluent interface
type ProductBuilder struct {
	name        string
	description string
	category    string
	price       float64
}

// NewProductBuilder creates a new builder with default values
func NewProductBuilder() *ProductBuilder {
	return &ProductBuilder{
		name:        "Test Product",
		description: "Default Description",
		category:    "electronics",
		price:       100.00,
	}
}

// WithName sets the product name
func (b *ProductBuilder) WithName(name string) *ProductBuilder {
	b.name = name
	return b
}

// WithDescription sets the product description
func (b *ProductBuilder) WithDescription(description string) *ProductBuilder {
	b.description = description
	return b
}

// WithCategory sets the product category
func (b *ProductBuilder) WithCategory(category string) *ProductBuilder {
	b.category = category
	return b
}

// WithPrice sets the product base price
func (b *ProductBuilder) WithPrice(price float64) *ProductBuilder {
	b.price = price
	return b
}

// Build creates the create_product.Request
func (b *ProductBuilder) Build() *create_product.Request {
	// 100 as denominator for 2 decimal places precision
	numerator := int64(b.price * 100)
	price, _ := domain.NewMoney(numerator, 100)

	return &create_product.Request{
		Name:        b.name,
		Description: b.description,
		Category:    b.category,
		BasePrice:   price,
	}
}
