package e2e

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/light-bringer/procat-service/internal/app/product/domain"
	"github.com/light-bringer/procat-service/internal/app/product/usecases/update_price"
)

func TestUpdatePrice(t *testing.T) {
	services, cleanup := setupTest(t)
	defer cleanup()

	ctx := context.Background()

	// 1. Create a product
	createReq := NewProductBuilder().
		WithName("Test Product").
		WithCategory("category-1").
		WithPrice(100.0).
		Build()
	productID, err := services.CreateProduct.Execute(ctx, createReq)
	require.NoError(t, err)

	// 2. Update price
	newPrice, err := domain.NewMoney(150, 1) // 150.00
	require.NoError(t, err)

	req := &update_price.Request{
		ProductID:     productID,
		NewPrice:      newPrice,
		ChangedBy:     "test-user",
		ChangedReason: "inflation",
	}

	err = services.UpdatePrice.Execute(ctx, req)
	require.NoError(t, err)

	// 3. Verify update
	product, err := services.ProductRepo.GetByID(ctx, productID)
	require.NoError(t, err)

	num, err := product.BasePrice().Numerator()
	require.NoError(t, err)
	denom, err := product.BasePrice().Denominator()
	require.NoError(t, err)
	assert.Equal(t, int64(150), num)
	assert.Equal(t, int64(1), denom)

	// 4. Verify version increment
	assert.Greater(t, product.Version(), int64(0), "Version should be incremented")
}

func TestUpdatePrice_OptimisticLocking(t *testing.T) {
	services, cleanup := setupTest(t)
	defer cleanup()

	ctx := context.Background()

	// 1. Create a product
	createReq := NewProductBuilder().
		WithName("Test Product").
		WithCategory("category-1").
		WithPrice(100.0).
		Build()
	productID, err := services.CreateProduct.Execute(ctx, createReq)
	require.NoError(t, err)

	// 2. Get current version
	product, err := services.ProductRepo.GetByID(ctx, productID)
	require.NoError(t, err)
	currentVersion := product.Version()

	// 3. Update price with correct version
	newPrice1, err := domain.NewMoney(150, 1)
	require.NoError(t, err)

	req1 := &update_price.Request{
		ProductID:     productID,
		Version:       currentVersion,
		NewPrice:      newPrice1,
		ChangedBy:     "user-1",
		ChangedReason: "reason-1",
	}
	err = services.UpdatePrice.Execute(ctx, req1)
	require.NoError(t, err)

	// Verify version changed
	productAfter1, _ := services.ProductRepo.GetByID(ctx, productID)
	assert.Equal(t, currentVersion+1, productAfter1.Version())

	// 4. Try to update again with OLD version (should fail)
	newPrice2, err := domain.NewMoney(200, 1)
	require.NoError(t, err)

	req2 := &update_price.Request{
		ProductID:     productID,
		Version:       currentVersion, // old version
		NewPrice:      newPrice2,
		ChangedBy:     "user-2",
		ChangedReason: "reason-2",
	}
	err = services.UpdatePrice.Execute(ctx, req2)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "concurrent modification", "Should fail with optimistic locking error")
}

func TestUpdatePrice_InvalidPrice(t *testing.T) {
	services, cleanup := setupTest(t)
	defer cleanup()

	ctx := context.Background()

	// 1. Create a product
	createReq := NewProductBuilder().
		WithName("Test Product").
		WithCategory("category-1").
		WithPrice(100.0).
		Build()
	productID, err := services.CreateProduct.Execute(ctx, createReq)
	require.NoError(t, err)

	// 2. Try to update to negative price
	newPrice, err := domain.NewMoney(-50, 1)
	require.NoError(t, err)

	req := &update_price.Request{
		ProductID:     productID,
		NewPrice:      newPrice,
		ChangedBy:     "user-bad",
		ChangedReason: "mistake",
	}

	err = services.UpdatePrice.Execute(ctx, req)
	require.Error(t, err)
	// Expect domain error
	// domain.ErrInvalidPrice
}
