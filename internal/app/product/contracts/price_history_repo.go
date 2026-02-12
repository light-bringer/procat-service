package contracts

import (
	"context"
	"time"

	"cloud.google.com/go/spanner"
	"github.com/light-bringer/procat-service/internal/app/product/domain"
)

// PriceHistoryRepository defines the interface for price history persistence.
type PriceHistoryRepository interface {
	// InsertMut creates a mutation for inserting a price change record.
	// oldPrice can be nil for initial product creation.
	// Returns error if money values exceed int64 bounds.
	InsertMut(
		historyID string,
		productID string,
		oldPrice *domain.Money,
		newPrice *domain.Money,
		changedBy string,
		changedReason string,
		changedAt time.Time,
	) (*spanner.Mutation, error)

	// GetByProductID retrieves price history for a product, ordered by time (most recent first).
	GetByProductID(ctx context.Context, productID string, limit int) ([]PriceHistoryRecord, error)
}

// PriceHistoryRecord represents a price change record.
type PriceHistoryRecord struct {
	HistoryID     string
	ProductID     string
	OldPrice      *domain.Money // nil for initial price
	NewPrice      *domain.Money
	ChangedBy     string
	ChangedReason string
	ChangedAt     time.Time
}
