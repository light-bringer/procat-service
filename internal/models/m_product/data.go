package m_product

import (
	"time"

	"cloud.google.com/go/spanner"
)

// Data represents the database model for the products table.
type Data struct {
	ProductID            string            `spanner:"product_id"`
	Name                 string            `spanner:"name"`
	Description          string            `spanner:"description"`
	Category             string            `spanner:"category"`
	BasePriceNumerator   int64             `spanner:"base_price_numerator"`
	BasePriceDenominator int64             `spanner:"base_price_denominator"`
	DiscountPercent      spanner.NullInt64 `spanner:"discount_percent"`
	DiscountStartDate    spanner.NullTime  `spanner:"discount_start_date"`
	DiscountEndDate      spanner.NullTime  `spanner:"discount_end_date"`
	Status               string            `spanner:"status"`
	Version              int64             `spanner:"version"`
	CreatedAt            time.Time         `spanner:"created_at"`
	UpdatedAt            time.Time         `spanner:"updated_at"`
	ArchivedAt           spanner.NullTime  `spanner:"archived_at"`
}
