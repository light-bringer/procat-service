package contracts

import (
	"context"

	"cloud.google.com/go/spanner"
	"github.com/light-bringer/procat-service/internal/app/product/domain"
)

// ProductRepository defines the interface for product persistence.
// Repositories return mutations, they don't apply them (Golden Mutation Pattern).
type ProductRepository interface {
	// InsertMut creates a mutation for inserting a new product
	// Returns error if money values exceed int64 bounds
	InsertMut(product *domain.Product) (*spanner.Mutation, error)

	// UpdateMut creates a mutation for updating a product (only dirty fields)
	// Returns error if money values exceed int64 bounds
	UpdateMut(product *domain.Product) (*spanner.Mutation, error)

	// GetByID retrieves a product by ID, reconstructing the domain aggregate
	GetByID(ctx context.Context, productID string) (*domain.Product, error)

	// Exists checks if a product exists
	Exists(ctx context.Context, productID string) (bool, error)
}
