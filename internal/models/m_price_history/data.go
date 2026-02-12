package m_price_history

import (
	"time"

	"cloud.google.com/go/spanner"
)

// Data represents a price history record in the database.
type Data struct {
	HistoryID           string             `spanner:"history_id"`
	ProductID           string             `spanner:"product_id"`
	OldPriceNumerator   spanner.NullInt64  `spanner:"old_price_numerator"`
	OldPriceDenominator spanner.NullInt64  `spanner:"old_price_denominator"`
	NewPriceNumerator   int64              `spanner:"new_price_numerator"`
	NewPriceDenominator int64              `spanner:"new_price_denominator"`
	ChangedBy           spanner.NullString `spanner:"changed_by"`
	ChangedReason       spanner.NullString `spanner:"changed_reason"`
	ChangedAt           time.Time          `spanner:"changed_at"`
}

// Model provides type-safe database operations for price history.
type Model struct{}

// NewModel creates a new price history model.
func NewModel() *Model {
	return &Model{}
}

// InsertMut creates a mutation for inserting a price history record.
func (m *Model) InsertMut(data *Data) *spanner.Mutation {
	mut, _ := spanner.InsertStruct(TableName, data)
	return mut
}

// ReadColumns returns the column names for reading price history.
func (m *Model) ReadColumns() []string {
	return []string{
		HistoryID,
		ProductID,
		OldPriceNumerator,
		OldPriceDenominator,
		NewPriceNumerator,
		NewPriceDenominator,
		ChangedBy,
		ChangedReason,
		ChangedAt,
	}
}
