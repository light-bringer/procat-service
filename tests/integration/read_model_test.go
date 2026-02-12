//go:build integration

package integration

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/light-bringer/procat-service/internal/app/product/contracts"
	"github.com/light-bringer/procat-service/internal/app/product/repo"
	"github.com/light-bringer/procat-service/tests/testutil"
)

func TestReadModel_GetProductByID(t *testing.T) {
	client, cleanup := testutil.SetupSpannerTest(t)
	defer cleanup()

	ctx := context.Background()
	readModel := repo.NewReadModel(client)

	t.Run("product found", func(t *testing.T) {
		// Create test product
		productID := testutil.CreateTestProduct(t, client, "Test Product")

		// Query using read model
		dto, err := readModel.GetProductByID(ctx, productID)
		require.NoError(t, err)

		assert.Equal(t, productID, dto.ProductID)
		assert.Equal(t, "Test Product", dto.Name)
		assert.Equal(t, "electronics", dto.Category)
		assert.Equal(t, 100.0, dto.BasePrice)
		assert.Equal(t, 100.0, dto.EffectivePrice) // No discount
		assert.False(t, dto.DiscountActive)
	})

	t.Run("product with active discount", func(t *testing.T) {
		// Create product with 20% discount
		productID := testutil.CreateTestProductWithDiscount(t, client, "Discounted Product", 20)

		// Query using read model
		dto, err := readModel.GetProductByID(ctx, productID)
		require.NoError(t, err)

		assert.Equal(t, 100.0, dto.BasePrice)
		assert.Equal(t, 80.0, dto.EffectivePrice) // 20% off
		assert.True(t, dto.DiscountActive)
		assert.NotNil(t, dto.DiscountPercent)
		assert.Equal(t, int64(20), *dto.DiscountPercent)
	})
}

func TestReadModel_ListProducts(t *testing.T) {
	client, cleanup := testutil.SetupSpannerTest(t)
	defer cleanup()

	ctx := context.Background()
	readModel := repo.NewReadModel(client)

	// Create test products in different categories
	testutil.CreateTestProduct(t, client, "Product 1")
	testutil.CreateTestProduct(t, client, "Product 2")

	t.Run("list all products", func(t *testing.T) {
		filter := &contracts.ListFilter{
			PageSize: 10,
		}

		result, err := readModel.ListProducts(ctx, filter)
		require.NoError(t, err)

		assert.Len(t, result.Products, 2)
		assert.Equal(t, int64(2), result.TotalCount)
	})

	t.Run("list with pagination", func(t *testing.T) {
		filter := &contracts.ListFilter{
			PageSize: 1,
		}

		firstPage, err := readModel.ListProducts(ctx, filter)
		require.NoError(t, err)
		assert.Len(t, firstPage.Products, 1)
		assert.NotEmpty(t, firstPage.NextPageToken)

		secondPage, err := readModel.ListProducts(ctx, &contracts.ListFilter{
			PageSize:  1,
			PageToken: firstPage.NextPageToken,
		})
		require.NoError(t, err)
		assert.Len(t, secondPage.Products, 1)
		assert.Empty(t, secondPage.NextPageToken)
		assert.NotEqual(t, firstPage.Products[0].ProductID, secondPage.Products[0].ProductID)
	})

	t.Run("filter by category", func(t *testing.T) {
		filter := &contracts.ListFilter{
			Category: "electronics",
			PageSize: 10,
		}

		result, err := readModel.ListProducts(ctx, filter)
		require.NoError(t, err)

		assert.Len(t, result.Products, 2) // Both are electronics
		for _, product := range result.Products {
			assert.Equal(t, "electronics", product.Category)
		}
	})

	t.Run("filter by status", func(t *testing.T) {
		// Create an active product
		testutil.CreateActiveTestProduct(t, client, "Active Product")

		filter := &contracts.ListFilter{
			Status:   "active",
			PageSize: 10,
		}

		result, err := readModel.ListProducts(ctx, filter)
		require.NoError(t, err)

		assert.GreaterOrEqual(t, len(result.Products), 1)
		for _, product := range result.Products {
			assert.Equal(t, "active", product.Status)
		}
	})
}

func TestReadModel_ListProducts_EmptyResult(t *testing.T) {
	client, cleanup := testutil.SetupSpannerTest(t)
	defer cleanup()

	ctx := context.Background()
	readModel := repo.NewReadModel(client)

	filter := &contracts.ListFilter{
		Category: "non-existent-category",
		PageSize: 10,
	}

	result, err := readModel.ListProducts(ctx, filter)
	require.NoError(t, err)

	assert.Empty(t, result.Products)
	assert.Equal(t, int64(0), result.TotalCount)
}
