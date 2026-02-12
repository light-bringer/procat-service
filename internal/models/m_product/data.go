package m_product

import (
	"time"

	"cloud.google.com/go/spanner"
)

// Data represents the database model for the products table.
type Data struct {
	ProductID              string
	Name                   string
	Description            string
	Category               string
	BasePriceNumerator     int64
	BasePriceDenominator   int64
	DiscountPercent        spanner.NullInt64
	DiscountStartDate      spanner.NullTime
	DiscountEndDate        spanner.NullTime
	Status                 string
	CreatedAt              time.Time
	UpdatedAt              time.Time
	ArchivedAt             spanner.NullTime
}
