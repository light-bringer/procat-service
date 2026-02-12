package get_product

import (
	"context"

	"github.com/light-bringer/procat-service/internal/app/product/contracts"
)

// Request contains the product ID to retrieve.
type Request struct {
	ProductID string
}

// Query handles the get product query use case.
type Query struct {
	readModel contracts.ReadModel
}

// NewQuery creates a new get product query.
func NewQuery(readModel contracts.ReadModel) *Query {
	return &Query{
		readModel: readModel,
	}
}

// Execute retrieves a product by ID.
func (q *Query) Execute(ctx context.Context, req *Request) (*contracts.ProductDTO, error) {
	return q.readModel.GetProductByID(ctx, req.ProductID)
}
