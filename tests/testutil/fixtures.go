package testutil

import (
	"context"
	"testing"
	"time"

	"cloud.google.com/go/spanner"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"github.com/light-bringer/procat-service/internal/models/m_outbox"
	"github.com/light-bringer/procat-service/internal/models/m_product"
)

// CreateTestProduct creates a test product directly in the database.
func CreateTestProduct(t *testing.T, client *spanner.Client, name string) string {
	t.Helper()

	ctx := context.Background()
	productID := uuid.New().String()
	now := time.Now()

	model := m_product.NewModel()
	data := &m_product.Data{
		ProductID:            productID,
		Name:                 name,
		Description:          "Test product description",
		Category:             "electronics",
		BasePriceNumerator:   10000,
		BasePriceDenominator: 100,
		Status:               "inactive",
		CreatedAt:            now,
		UpdatedAt:            now,
	}

	mutation := model.InsertMut(data)
	_, err := client.Apply(ctx, []*spanner.Mutation{mutation})
	require.NoError(t, err, "failed to create test product")

	return productID
}

// CreateTestProductWithDiscount creates a test product with an active discount.
func CreateTestProductWithDiscount(t *testing.T, client *spanner.Client, name string, discountPercent float64) string {
	t.Helper()

	ctx := context.Background()
	productID := uuid.New().String()
	now := time.Now()

	model := m_product.NewModel()
	data := &m_product.Data{
		ProductID:            productID,
		Name:                 name,
		Description:          "Test product with discount",
		Category:             "electronics",
		BasePriceNumerator:   10000,
		BasePriceDenominator: 100,
		DiscountPercent:      spanner.NullFloat64{Float64: discountPercent, Valid: true}, // Changed to NullFloat64
		DiscountStartDate:    spanner.NullTime{Time: now.Add(-1 * time.Hour), Valid: true},
		DiscountEndDate:      spanner.NullTime{Time: now.Add(24 * time.Hour), Valid: true},
		Status:               "active",
		CreatedAt:            now,
		UpdatedAt:            now,
	}

	mutation := model.InsertMut(data)
	_, err := client.Apply(ctx, []*spanner.Mutation{mutation})
	require.NoError(t, err, "failed to create test product with discount")

	return productID
}

// CreateActiveTestProduct creates an active test product.
func CreateActiveTestProduct(t *testing.T, client *spanner.Client, name string) string {
	t.Helper()

	ctx := context.Background()
	productID := uuid.New().String()
	now := time.Now()

	model := m_product.NewModel()
	data := &m_product.Data{
		ProductID:            productID,
		Name:                 name,
		Description:          "Active test product",
		Category:             "electronics",
		BasePriceNumerator:   10000,
		BasePriceDenominator: 100,
		Status:               "active",
		CreatedAt:            now,
		UpdatedAt:            now,
	}

	mutation := model.InsertMut(data)
	_, err := client.Apply(ctx, []*spanner.Mutation{mutation})
	require.NoError(t, err, "failed to create active test product")

	return productID
}

// AssertOutboxEvent verifies an outbox event exists with the given event type.
func AssertOutboxEvent(t *testing.T, client *spanner.Client, eventType string) {
	t.Helper()

	ctx := context.Background()
	stmt := spanner.Statement{
		SQL:    "SELECT event_id FROM outbox_events WHERE event_type = @eventType",
		Params: map[string]interface{}{"eventType": eventType},
	}

	iter := client.Single().Query(ctx, stmt)
	defer iter.Stop()

	row, err := iter.Next()
	require.NoError(t, err, "outbox event not found for type: %s", eventType)
	require.NotNil(t, row, "outbox event not found for type: %s", eventType)
}

// AssertOutboxEventCount verifies the count of outbox events.
func AssertOutboxEventCount(t *testing.T, client *spanner.Client, expectedCount int) {
	t.Helper()

	ctx := context.Background()
	stmt := spanner.Statement{
		SQL: "SELECT COUNT(*) FROM outbox_events",
	}

	iter := client.Single().Query(ctx, stmt)
	defer iter.Stop()

	row, err := iter.Next()
	require.NoError(t, err, "failed to query outbox event count")

	var count int64
	err = row.Columns(&count)
	require.NoError(t, err, "failed to parse count")

	require.Equal(t, int64(expectedCount), count, "unexpected outbox event count")
}

// GetProductByID retrieves a product from the database for verification.
func GetProductByID(t *testing.T, client *spanner.Client, productID string) *m_product.Data {
	t.Helper()

	ctx := context.Background()
	row, err := client.Single().ReadRow(ctx, m_product.TableName, spanner.Key{productID}, []string{
		m_product.ProductID,
		m_product.Name,
		m_product.Description,
		m_product.Category,
		m_product.BasePriceNumerator,
		m_product.BasePriceDenominator,
		m_product.DiscountPercent,
		m_product.DiscountStartDate,
		m_product.DiscountEndDate,
		m_product.Status,
		m_product.CreatedAt,
		m_product.UpdatedAt,
		m_product.ArchivedAt,
	})
	require.NoError(t, err, "failed to get product by id")

	var data m_product.Data
	err = row.ToStruct(&data)
	require.NoError(t, err, "failed to parse product data")

	return &data
}

// CreateTestOutboxEvent creates a test outbox event.
func CreateTestOutboxEvent(t *testing.T, client *spanner.Client, eventType string, aggregateID string) string {
	t.Helper()

	ctx := context.Background()
	eventID := uuid.New().String()

	model := m_outbox.NewModel()
	data := &m_outbox.Data{
		EventID:     eventID,
		EventType:   eventType,
		AggregateID: aggregateID,
		Payload:     spanner.NullJSON{Value: `{"test": "data"}`, Valid: true},
		Status:      m_outbox.StatusPending,
		RetryCount:  0,
	}

	mutation := model.InsertMut(data)
	_, err := client.Apply(ctx, []*spanner.Mutation{mutation})
	require.NoError(t, err, "failed to create test outbox event")

	return eventID
}

// UpdateTestProductName updates a product's name for testing.
func UpdateTestProductName(t *testing.T, client *spanner.Client, productID string, newName string) {
	t.Helper()

	ctx := context.Background()
	model := m_product.NewModel()

	updates := map[string]interface{}{
		m_product.Name:      newName,
		m_product.UpdatedAt: time.Now(),
	}

	mutation := model.UpdateMut(productID, updates)
	_, err := client.Apply(ctx, []*spanner.Mutation{mutation})
	require.NoError(t, err, "failed to update product name")
}

// CreateTestProductWithStatus creates a test product with specific status.
func CreateTestProductWithStatus(t *testing.T, client *spanner.Client, name string, status string) string {
	t.Helper()

	ctx := context.Background()
	productID := uuid.New().String()
	now := time.Now()

	model := m_product.NewModel()
	data := &m_product.Data{
		ProductID:            productID,
		Name:                 name,
		Description:          "Test product with status",
		Category:             "electronics",
		BasePriceNumerator:   10000,
		BasePriceDenominator: 100,
		Status:               status,
		CreatedAt:            now,
		UpdatedAt:            now,
	}

	mutation := model.InsertMut(data)
	_, err := client.Apply(ctx, []*spanner.Mutation{mutation})
	require.NoError(t, err, "failed to create test product with status")

	return productID
}

// CreateTestProductWithCategory creates a test product in specific category.
func CreateTestProductWithCategory(t *testing.T, client *spanner.Client, name string, category string) string {
	t.Helper()

	ctx := context.Background()
	productID := uuid.New().String()
	now := time.Now()

	model := m_product.NewModel()
	data := &m_product.Data{
		ProductID:            productID,
		Name:                 name,
		Description:          "Test product in category",
		Category:             category,
		BasePriceNumerator:   10000,
		BasePriceDenominator: 100,
		Status:               "inactive",
		CreatedAt:            now,
		UpdatedAt:            now,
	}

	mutation := model.InsertMut(data)
	_, err := client.Apply(ctx, []*spanner.Mutation{mutation})
	require.NoError(t, err, "failed to create test product with category")

	return productID
}
