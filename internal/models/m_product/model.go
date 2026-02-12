package m_product

import (
	"cloud.google.com/go/spanner"
)

// Model provides a facade for type-safe operations on the products table.
type Model struct{}

// NewModel creates a new Model instance.
func NewModel() *Model {
	return &Model{}
}

// InsertMut creates a Spanner mutation for inserting a product.
func (m *Model) InsertMut(data *Data) *spanner.Mutation {
	return spanner.InsertOrUpdate(
		TableName,
		[]string{
			ProductID,
			Name,
			Description,
			Category,
			BasePriceNumerator,
			BasePriceDenominator,
			DiscountPercent,
			DiscountStartDate,
			DiscountEndDate,
			Status,
			Version,
			CreatedAt,
			UpdatedAt,
			ArchivedAt,
		},
		[]interface{}{
			data.ProductID,
			data.Name,
			data.Description,
			data.Category,
			data.BasePriceNumerator,
			data.BasePriceDenominator,
			data.DiscountPercent,
			data.DiscountStartDate,
			data.DiscountEndDate,
			data.Status,
			data.Version,
			spanner.CommitTimestamp,
			spanner.CommitTimestamp,
			data.ArchivedAt,
		},
	)
}

// UpdateMut creates a Spanner mutation for updating specific product fields.
// The updates map should contain field names as keys and new values.
func (m *Model) UpdateMut(productID string, updates map[string]interface{}) *spanner.Mutation {
	if len(updates) == 0 {
		return nil
	}

	// Always update the UpdatedAt timestamp
	updates[UpdatedAt] = spanner.CommitTimestamp

	columns := make([]string, 0, len(updates)+1)
	values := make([]interface{}, 0, len(updates)+1)

	// Add product ID first
	columns = append(columns, ProductID)
	values = append(values, productID)

	// Add all update fields
	for col, val := range updates {
		columns = append(columns, col)
		values = append(values, val)
	}

	return spanner.Update(TableName, columns, values)
}

// DeleteMut creates a Spanner mutation for deleting a product (hard delete).
func (m *Model) DeleteMut(productID string) *spanner.Mutation {
	return spanner.Delete(TableName, spanner.Key{productID})
}
