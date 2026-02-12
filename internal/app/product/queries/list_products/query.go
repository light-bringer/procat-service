package list_products

import (
	"context"

	"github.com/light-bringer/procat-service/internal/app/product/contracts"
)

// Request contains filtering and pagination parameters.
type Request struct {
	Category  string
	Status    string
	PageSize  int
	PageToken string
}

// Query handles the list products query use case.
type Query struct {
	readModel contracts.ReadModel
}

// NewQuery creates a new list products query.
func NewQuery(readModel contracts.ReadModel) *Query {
	return &Query{
		readModel: readModel,
	}
}

// Execute retrieves a paginated list of products with filtering.
func (q *Query) Execute(ctx context.Context, req *Request) (*contracts.ListResult, error) {
	filter := &contracts.ListFilter{
		Category:  req.Category,
		Status:    req.Status,
		PageSize:  req.PageSize,
		PageToken: req.PageToken,
	}

	return q.readModel.ListProducts(ctx, filter)
}
