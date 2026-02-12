//go:build integration

package integration

import (
	"context"
	"testing"
	"time"

	"cloud.google.com/go/spanner"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/light-bringer/procat-service/internal/app/product/domain"
	"github.com/light-bringer/procat-service/internal/app/product/repo"
	"github.com/light-bringer/procat-service/internal/pkg/clock"
	"github.com/light-bringer/procat-service/tests/testutil"
)

func TestProductRepository_InsertMut(t *testing.T) {
	client, cleanup := testutil.SetupSpannerTest(t)
	defer cleanup()

	ctx := context.Background()
	repository := repo.NewProductRepo(client)

	// Create a new product
	price, _ := domain.NewMoney(10000, 100) // $100.00
	now := time.Now()
	clk := clock.NewMockClock(now)
	product, err := domain.NewProduct("test-id-1", "Test Product", "Description", "electronics", price, now, clk)
	require.NoError(t, err)

	// Get mutation and apply it
	mutation := repository.InsertMut(product)
	require.NotNil(t, mutation)

	_, err = client.Apply(ctx, []*spanner.Mutation{mutation})
	require.NoError(t, err)

	// Verify product was inserted
	testutil.AssertRowCount(t, client, "products", 1)

	// Verify we can read it back
	retrieved, err := repository.GetByID(ctx, "test-id-1")
	require.NoError(t, err)
	assert.Equal(t, "Test Product", retrieved.Name())
	assert.Equal(t, "electronics", retrieved.Category())
	assert.Equal(t, domain.StatusInactive, retrieved.Status())
}

func TestProductRepository_UpdateMut(t *testing.T) {
	client, cleanup := testutil.SetupSpannerTest(t)
	defer cleanup()

	ctx := context.Background()
	repository := repo.NewProductRepo(client)

	// Create and insert a product
	price, _ := domain.NewMoney(10000, 100)
	now := time.Now()
	clk := clock.NewMockClock(now)
	product, _ := domain.NewProduct("test-id-2", "Original Name", "Description", "electronics", price, now, clk)

	_, err := client.Apply(ctx, []*spanner.Mutation{repository.InsertMut(product)})
	require.NoError(t, err)

	// Retrieve and update
	retrieved, err := repository.GetByID(ctx, "test-id-2")
	require.NoError(t, err)

	err = retrieved.SetName("Updated Name")
	require.NoError(t, err)

	err = retrieved.SetCategory("books")
	require.NoError(t, err)

	// Get update mutation - should only include dirty fields
	updateMut := repository.UpdateMut(retrieved)
	require.NotNil(t, updateMut)

	_, err = client.Apply(ctx, []*spanner.Mutation{updateMut})
	require.NoError(t, err)

	// Verify updates persisted
	final, err := repository.GetByID(ctx, "test-id-2")
	require.NoError(t, err)
	assert.Equal(t, "Updated Name", final.Name())
	assert.Equal(t, "books", final.Category())
	assert.Equal(t, "Description", final.Description()) // Unchanged
}

func TestProductRepository_UpdateMut_OnlyDirtyFields(t *testing.T) {
	client, cleanup := testutil.SetupSpannerTest(t)
	defer cleanup()

	ctx := context.Background()
	repository := repo.NewProductRepo(client)

	// Create a product
	price, _ := domain.NewMoney(10000, 100)
	now := time.Now()
	clk := clock.NewMockClock(now)
	product, _ := domain.NewProduct("test-id-3", "Test", "Desc", "electronics", price, now, clk)
	_, err := client.Apply(ctx, []*spanner.Mutation{repository.InsertMut(product)})
	require.NoError(t, err)

	// Retrieve without making changes
	retrieved, err := repository.GetByID(ctx, "test-id-3")
	require.NoError(t, err)

	// UpdateMut should return nil if no changes
	updateMut := repository.UpdateMut(retrieved)
	assert.Nil(t, updateMut, "expected nil mutation when no fields are dirty")
}

func TestProductRepository_GetByID(t *testing.T) {
	client, cleanup := testutil.SetupSpannerTest(t)
	defer cleanup()

	ctx := context.Background()
	repository := repo.NewProductRepo(client)

	t.Run("product exists", func(t *testing.T) {
		// Create product using fixture
		productID := testutil.CreateTestProduct(t, client, "Existing Product")

		// Retrieve using repository
		product, err := repository.GetByID(ctx, productID)
		require.NoError(t, err)
		assert.Equal(t, productID, product.ID())
		assert.Equal(t, "Existing Product", product.Name())
		assert.False(t, product.Changes().HasChanges(), "retrieved product should have no dirty fields")
	})

	t.Run("product not found", func(t *testing.T) {
		_, err := repository.GetByID(ctx, "non-existent-id")
		assert.ErrorIs(t, err, domain.ErrProductNotFound)
	})
}

func TestProductRepository_ReconstructProductWithDiscount(t *testing.T) {
	client, cleanup := testutil.SetupSpannerTest(t)
	defer cleanup()

	ctx := context.Background()
	repository := repo.NewProductRepo(client)

	// Create product with discount using fixture
	productID := testutil.CreateTestProductWithDiscount(t, client, "Discounted Product", 20)

	// Retrieve and verify discount
	product, err := repository.GetByID(ctx, productID)
	require.NoError(t, err)

	discount := product.Discount()
	require.NotNil(t, discount, "discount should be present")
	assert.Equal(t, int64(20), discount.Percentage())

	// Verify effective price calculation
	now := time.Now()
	effectivePrice := product.CalculateEffectivePrice(now)
	assert.Equal(t, 80.0, effectivePrice.Float64(), "20% discount on $100 should be $80")
}

func TestProductRepository_Exists(t *testing.T) {
	client, cleanup := testutil.SetupSpannerTest(t)
	defer cleanup()

	ctx := context.Background()
	repository := repo.NewProductRepo(client)

	t.Run("product exists", func(t *testing.T) {
		productID := testutil.CreateTestProduct(t, client, "Test Product")
		exists, err := repository.Exists(ctx, productID)
		require.NoError(t, err)
		assert.True(t, exists)
	})

	t.Run("product does not exist", func(t *testing.T) {
		exists, err := repository.Exists(ctx, "non-existent-id")
		require.NoError(t, err)
		assert.False(t, exists)
	})
}
