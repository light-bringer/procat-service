//go:build integration

package integration

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/light-bringer/procat-service/internal/app/product/contracts"
	"github.com/light-bringer/procat-service/internal/app/product/repo"
	"github.com/light-bringer/procat-service/internal/pkg/clock"
	"github.com/light-bringer/procat-service/tests/testutil"
)

func TestReadModel_GetProductByID(t *testing.T) {
	client, cleanup := testutil.SetupSpannerTest(t)
	defer cleanup()

	ctx := context.Background()
	readModel := repo.NewReadModel(client, clock.NewRealClock())

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
	readModel := repo.NewReadModel(client, clock.NewRealClock())

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
	readModel := repo.NewReadModel(client, clock.NewRealClock())

	filter := &contracts.ListFilter{
		Category: "non-existent-category",
		PageSize: 10,
	}

	result, err := readModel.ListProducts(ctx, filter)
	require.NoError(t, err)

	assert.Empty(t, result.Products)
	assert.Equal(t, int64(0), result.TotalCount)
}

// TestReadConsistency_ReadYourWrites verifies writes are immediately visible in reads.
func TestReadConsistency_ReadYourWrites(t *testing.T) {
	client, cleanup := testutil.SetupSpannerTest(t)
	defer cleanup()

	ctx := context.Background()
	readModel := repo.NewReadModel(client, clock.NewRealClock())

	// Create a product
	productID := testutil.CreateTestProduct(t, client, "Consistency Test Product")

	// Immediately read it back - should be visible
	dto, err := readModel.GetProductByID(ctx, productID)
	require.NoError(t, err)
	assert.Equal(t, productID, dto.ProductID)
	assert.Equal(t, "Consistency Test Product", dto.Name)
}

// TestReadConsistency_UpdateImmediatelyVisible verifies updates are visible immediately.
func TestReadConsistency_UpdateImmediatelyVisible(t *testing.T) {
	client, cleanup := testutil.SetupSpannerTest(t)
	defer cleanup()

	ctx := context.Background()
	readModel := repo.NewReadModel(client, clock.NewRealClock())

	// Create a product
	productID := testutil.CreateTestProduct(t, client, "Original Name")

	// Update the product name
	testutil.UpdateTestProductName(t, client, productID, "Updated Name")

	// Read should immediately show new value
	dto, err := readModel.GetProductByID(ctx, productID)
	require.NoError(t, err)
	assert.Equal(t, "Updated Name", dto.Name)
}

// TestReadConsistency_ListPaginationNoMissingItems verifies pagination doesn't miss items.
func TestReadConsistency_ListPaginationNoMissingItems(t *testing.T) {
	client, cleanup := testutil.SetupSpannerTest(t)
	defer cleanup()

	ctx := context.Background()
	readModel := repo.NewReadModel(client, clock.NewRealClock())

	// Create 5 products
	expectedIDs := make(map[string]bool)
	for i := 1; i <= 5; i++ {
		productID := testutil.CreateTestProduct(t, client, "Product "+string(rune('A'+i-1)))
		expectedIDs[productID] = true
	}

	// Paginate through all products with page size 2
	seenIDs := make(map[string]bool)
	filter := &contracts.ListFilter{PageSize: 2}

	for {
		result, err := readModel.ListProducts(ctx, filter)
		require.NoError(t, err)

		for _, product := range result.Products {
			seenIDs[product.ProductID] = true
		}

		if result.NextPageToken == "" {
			break
		}
		filter.PageToken = result.NextPageToken
	}

	// Verify we saw all products exactly once
	assert.Equal(t, len(expectedIDs), len(seenIDs))
	for id := range expectedIDs {
		assert.True(t, seenIDs[id], "Product %s should be in paginated results", id)
	}
}

// TestReadConsistency_FilterCorrectness verifies filters return correct results.
func TestReadConsistency_FilterCorrectness(t *testing.T) {
	client, cleanup := testutil.SetupSpannerTest(t)
	defer cleanup()

	ctx := context.Background()
	readModel := repo.NewReadModel(client, clock.NewRealClock())

	// Create products with different attributes
	electronicsInactive := testutil.CreateTestProduct(t, client, "Laptop")
	electronicsActive := testutil.CreateTestProductWithStatus(t, client, "Phone", "active")
	furnitureInactive := testutil.CreateTestProductWithCategory(t, client, "Chair", "furniture")

	t.Run("filter by category returns only matching products", func(t *testing.T) {
		filter := &contracts.ListFilter{
			Category: "electronics",
			PageSize: 10,
		}

		result, err := readModel.ListProducts(ctx, filter)
		require.NoError(t, err)

		assert.GreaterOrEqual(t, len(result.Products), 2)

		foundLaptop := false
		foundPhone := false
		for _, product := range result.Products {
			assert.Equal(t, "electronics", product.Category)
			if product.ProductID == electronicsInactive {
				foundLaptop = true
			}
			if product.ProductID == electronicsActive {
				foundPhone = true
			}
			// Should NOT find furniture
			assert.NotEqual(t, furnitureInactive, product.ProductID)
		}
		assert.True(t, foundLaptop, "Should find laptop")
		assert.True(t, foundPhone, "Should find phone")
	})

	t.Run("filter by status returns only matching products", func(t *testing.T) {
		filter := &contracts.ListFilter{
			Status:   "active",
			PageSize: 10,
		}

		result, err := readModel.ListProducts(ctx, filter)
		require.NoError(t, err)

		foundActive := false
		for _, product := range result.Products {
			assert.Equal(t, "active", product.Status)
			if product.ProductID == electronicsActive {
				foundActive = true
			}
			// Should NOT find inactive products
			assert.NotEqual(t, electronicsInactive, product.ProductID)
			assert.NotEqual(t, furnitureInactive, product.ProductID)
		}
		assert.True(t, foundActive, "Should find active electronics product")
	})

	t.Run("combined filters work correctly", func(t *testing.T) {
		filter := &contracts.ListFilter{
			Category: "electronics",
			Status:   "active",
			PageSize: 10,
		}

		result, err := readModel.ListProducts(ctx, filter)
		require.NoError(t, err)

		foundPhone := false
		for _, product := range result.Products {
			assert.Equal(t, "electronics", product.Category)
			assert.Equal(t, "active", product.Status)
			if product.ProductID == electronicsActive {
				foundPhone = true
			}
		}
		assert.True(t, foundPhone, "Should find active electronics product")
	})
}

// TestReadConsistency_DiscountCalculations verifies effective price calculations are correct.
func TestReadConsistency_DiscountCalculations(t *testing.T) {
	client, cleanup := testutil.SetupSpannerTest(t)
	defer cleanup()

	ctx := context.Background()
	readModel := repo.NewReadModel(client, clock.NewRealClock())

	t.Run("no discount shows base price as effective price", func(t *testing.T) {
		productID := testutil.CreateTestProduct(t, client, "No Discount Product")

		dto, err := readModel.GetProductByID(ctx, productID)
		require.NoError(t, err)

		assert.Equal(t, dto.BasePrice, dto.EffectivePrice)
		assert.False(t, dto.DiscountActive)
	})

	t.Run("active discount calculates correct effective price", func(t *testing.T) {
		// 25% discount
		productID := testutil.CreateTestProductWithDiscount(t, client, "Discounted Product", 25)

		dto, err := readModel.GetProductByID(ctx, productID)
		require.NoError(t, err)

		assert.Equal(t, 100.0, dto.BasePrice)
		assert.Equal(t, 75.0, dto.EffectivePrice) // 100 - 25% = 75
		assert.True(t, dto.DiscountActive)
		assert.NotNil(t, dto.DiscountPercent)
		assert.Equal(t, int64(25), *dto.DiscountPercent)
	})

	t.Run("100% discount results in zero price", func(t *testing.T) {
		productID := testutil.CreateTestProductWithDiscount(t, client, "Free Product", 100)

		dto, err := readModel.GetProductByID(ctx, productID)
		require.NoError(t, err)

		assert.Equal(t, 100.0, dto.BasePrice)
		assert.Equal(t, 0.0, dto.EffectivePrice)
		assert.True(t, dto.DiscountActive)
	})
}

// TestReadConsistency_TotalCountAccuracy verifies total count matches actual results.
func TestReadConsistency_TotalCountAccuracy(t *testing.T) {
	client, cleanup := testutil.SetupSpannerTest(t)
	defer cleanup()

	ctx := context.Background()
	readModel := repo.NewReadModel(client, clock.NewRealClock())

	// Create known number of products
	for i := 0; i < 7; i++ {
		testutil.CreateTestProduct(t, client, "Product "+string(rune('A'+i)))
	}

	// Query with pagination
	allProducts := make([]*contracts.ProductDTO, 0)
	filter := &contracts.ListFilter{PageSize: 3}
	var reportedTotal int64

	for {
		result, err := readModel.ListProducts(ctx, filter)
		require.NoError(t, err)

		reportedTotal = result.TotalCount
		allProducts = append(allProducts, result.Products...)

		if result.NextPageToken == "" {
			break
		}
		filter.PageToken = result.NextPageToken
	}

	// Total count should match actual collected count
	assert.Equal(t, reportedTotal, int64(len(allProducts)))
	assert.GreaterOrEqual(t, len(allProducts), 7)
}
