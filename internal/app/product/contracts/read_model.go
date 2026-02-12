package contracts

import (
	"context"
	"time"
)

// ProductDTO is a data transfer object for product queries.
type ProductDTO struct {
	ProductID       string
	Name            string
	Description     string
	Category        string
	BasePrice       float64  // Approximate representation for display
	EffectivePrice  float64  // Current price with discount applied
	DiscountPercent *float64 // Changed from *int64 to *float64 for fractional percentages
	DiscountActive  bool
	Status          string
	CreatedAt       time.Time
	UpdatedAt       time.Time
}

// ListFilter defines filtering options for listing products.
type ListFilter struct {
	Category  string
	Status    string
	PageSize  int
	PageToken string
}

// ListResult contains paginated product list results.
type ListResult struct {
	Products      []*ProductDTO
	NextPageToken string
	TotalCount    int64
}

// ReadModel defines the interface for product queries.
// Read models can bypass the domain layer for performance.
type ReadModel interface {
	// GetProductByID retrieves a product DTO by ID
	GetProductByID(ctx context.Context, productID string) (*ProductDTO, error)

	// ListProducts retrieves a paginated list of products with filtering
	ListProducts(ctx context.Context, filter *ListFilter) (*ListResult, error)
}
