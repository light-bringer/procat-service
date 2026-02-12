package e2e

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/light-bringer/procat-service/internal/app/product/domain"
	"github.com/light-bringer/procat-service/internal/app/product/queries/get_product"
	"github.com/light-bringer/procat-service/internal/app/product/queries/list_products"
	"github.com/light-bringer/procat-service/internal/app/product/usecases/activate_product"
	"github.com/light-bringer/procat-service/internal/app/product/usecases/apply_discount"
	"github.com/light-bringer/procat-service/internal/app/product/usecases/archive_product"
	"github.com/light-bringer/procat-service/internal/app/product/usecases/deactivate_product"
	"github.com/light-bringer/procat-service/internal/app/product/usecases/update_product"
	"github.com/light-bringer/procat-service/tests/testutil"
)

func TestProductCreationFlow(t *testing.T) {
	services, cleanup := setupTest(t)
	defer cleanup()

	// Create a product
	req := NewProductBuilder().
		WithName("MacBook Pro").
		WithDescription("16-inch laptop").
		WithCategory("electronics").
		WithPrice(2499.00).
		Build()

	productID, err := services.CreateProduct.Execute(ctx(), req)
	require.NoError(t, err)
	assert.NotEmpty(t, productID)

	// Verify product exists via query
	dto, err := services.GetProduct.Execute(ctx(), &get_product.Request{ProductID: productID})
	require.NoError(t, err)
	assert.Equal(t, "MacBook Pro", dto.Name)
	assert.Equal(t, "electronics", dto.Category)
	assert.Equal(t, 2499.00, dto.BasePrice)
	assert.Equal(t, "inactive", dto.Status)

	// Verify outbox event created
	testutil.AssertOutboxEvent(t, services.Client, "product.created")
}

func TestProductActivationDeactivation(t *testing.T) {
	services, cleanup := setupTest(t)
	defer cleanup()

	// Create a product
	req := NewProductBuilder().
		WithName("Test Product").
		WithDescription("Test").
		WithCategory("electronics").
		WithPrice(100.00).
		Build()

	productID, err := services.CreateProduct.Execute(ctx(), req)
	require.NoError(t, err)

	// Activate product
	err = services.ActivateProduct.Execute(ctx(), &activate_product.Request{ProductID: productID})
	require.NoError(t, err)

	// Verify status changed
	dto, _ := services.GetProduct.Execute(ctx(), &get_product.Request{ProductID: productID})
	assert.Equal(t, "active", dto.Status)

	// Verify activation event
	testutil.AssertOutboxEvent(t, services.Client, "product.activated")

	// Deactivate product
	err = services.DeactivateProduct.Execute(ctx(), &deactivate_product.Request{ProductID: productID})
	require.NoError(t, err)

	// Verify status changed back
	dto, _ = services.GetProduct.Execute(ctx(), &get_product.Request{ProductID: productID})
	assert.Equal(t, "inactive", dto.Status)

	// Verify deactivation event
	testutil.AssertOutboxEvent(t, services.Client, "product.deactivated")
}

func TestProductUpdateFlow(t *testing.T) {
	services, cleanup := setupTest(t)
	defer cleanup()

	// Create a product
	req := NewProductBuilder().
		WithName("Original Name").
		WithDescription("Original Description").
		WithCategory("electronics").
		WithPrice(100.00).
		Build()

	productID, err := services.CreateProduct.Execute(ctx(), req)
	require.NoError(t, err)

	// Update product
	newName := "Updated Name"
	newCategory := "books"
	err = services.UpdateProduct.Execute(ctx(), &update_product.Request{
		ProductID: productID,
		Name:      &newName,
		Category:  &newCategory,
	})
	require.NoError(t, err)

	// Verify updates persisted
	dto, _ := services.GetProduct.Execute(ctx(), &get_product.Request{ProductID: productID})
	assert.Equal(t, "Updated Name", dto.Name)
	assert.Equal(t, "books", dto.Category)
	assert.Equal(t, "Original Description", dto.Description) // Unchanged
}

func TestProductArchiving(t *testing.T) {
	services, cleanup := setupTest(t)
	defer cleanup()

	// Create a product
	req := NewProductBuilder().
		WithName("To Archive").
		WithDescription("Test").
		WithCategory("electronics").
		WithPrice(100.00).
		Build()

	productID, err := services.CreateProduct.Execute(ctx(), req)
	require.NoError(t, err)

	// Archive product
	err = services.ArchiveProduct.Execute(ctx(), &archive_product.Request{ProductID: productID})
	require.NoError(t, err)

	// Verify status changed to archived
	dto, _ := services.GetProduct.Execute(ctx(), &get_product.Request{ProductID: productID})
	assert.Equal(t, "archived", dto.Status)

	// Verify archived event
	testutil.AssertOutboxEvent(t, services.Client, "product.archived")

	// Verify cannot modify archived product
	newName := "Should Fail"
	err = services.UpdateProduct.Execute(ctx(), &update_product.Request{
		ProductID: productID,
		Name:      &newName,
	})
	assert.Error(t, err)
	assert.ErrorIs(t, err, domain.ErrCannotModifyArchived)

	// Verify cannot activate archived product
	err = services.ActivateProduct.Execute(ctx(), &activate_product.Request{ProductID: productID})
	assert.Error(t, err)
	assert.ErrorIs(t, err, domain.ErrCannotModifyArchived)

	// Verify cannot apply discount to archived product
	err = services.ApplyDiscount.Execute(ctx(), &apply_discount.Request{
		ProductID:       productID,
		DiscountPercent: 10,
		StartDate:       time.Now(),
		EndDate:         time.Now().Add(24 * time.Hour),
	})
	assert.Error(t, err)
	assert.ErrorIs(t, err, domain.ErrCannotModifyArchived)
}

func TestArchiveActiveProduct(t *testing.T) {
	services, cleanup := setupTest(t)
	defer cleanup()

	// Create and activate product
	req := NewProductBuilder().
		WithName("Active to Archive").
		WithDescription("Test").
		WithCategory("electronics").
		WithPrice(100.00).
		Build()

	productID, err := services.CreateProduct.Execute(ctx(), req)
	require.NoError(t, err)

	err = services.ActivateProduct.Execute(ctx(), &activate_product.Request{ProductID: productID})
	require.NoError(t, err)

	// Archive active product
	err = services.ArchiveProduct.Execute(ctx(), &archive_product.Request{ProductID: productID})
	require.NoError(t, err)

	// Verify status
	dto, _ := services.GetProduct.Execute(ctx(), &get_product.Request{ProductID: productID})
	assert.Equal(t, "archived", dto.Status)
}

func TestBusinessRuleValidations(t *testing.T) {
	services, cleanup := setupTest(t)
	defer cleanup()

	t.Run("cannot create product with empty name", func(t *testing.T) {
		req := NewProductBuilder().
			WithName("").
			WithDescription("Test").
			WithCategory("electronics").
			WithPrice(100.00).
			Build()

		_, err := services.CreateProduct.Execute(ctx(), req)
		assert.ErrorIs(t, err, domain.ErrEmptyName)
	})

	t.Run("cannot create product with negative price", func(t *testing.T) {
		req := NewProductBuilder().
			WithName("Test").
			WithDescription("Test").
			WithCategory("electronics").
			WithPrice(-100.00).
			Build()

		_, err := services.CreateProduct.Execute(ctx(), req)
		assert.ErrorIs(t, err, domain.ErrInvalidPrice)
	})

	t.Run("cannot activate already active product", func(t *testing.T) {
		req := NewProductBuilder().
			WithName("Test").
			WithDescription("Test").
			WithCategory("electronics").
			WithPrice(100.00).
			Build()

		productID, _ := services.CreateProduct.Execute(ctx(), req)

		// Activate once
		_ = services.ActivateProduct.Execute(ctx(), &activate_product.Request{ProductID: productID})

		// Try to activate again
		err := services.ActivateProduct.Execute(ctx(), &activate_product.Request{ProductID: productID})
		assert.ErrorIs(t, err, domain.ErrAlreadyActive)
	})
}

func TestListProductsWithFiltering(t *testing.T) {
	services, cleanup := setupTest(t)
	defer cleanup()

	// Create multiple products
	req1 := NewProductBuilder().WithName("Product 1").WithCategory("electronics").Build()
	req2 := NewProductBuilder().WithName("Product 2").WithCategory("books").Build()
	req3 := NewProductBuilder().WithName("Product 3").WithCategory("electronics").Build()

	services.CreateProduct.Execute(ctx(), req1)
	services.CreateProduct.Execute(ctx(), req2)
	services.CreateProduct.Execute(ctx(), req3)

	t.Run("list all products", func(t *testing.T) {
		result, err := services.ListProducts.Execute(ctx(), &list_products.Request{PageSize: 10})
		require.NoError(t, err)
		assert.Len(t, result.Products, 3)
	})

	t.Run("filter by category", func(t *testing.T) {
		result, err := services.ListProducts.Execute(ctx(), &list_products.Request{
			Category: "electronics",
			PageSize: 10,
		})
		require.NoError(t, err)
		assert.Len(t, result.Products, 2)
		for _, p := range result.Products {
			assert.Equal(t, "electronics", p.Category)
		}
	})

	t.Run("filter by status", func(t *testing.T) {
		result, err := services.ListProducts.Execute(ctx(), &list_products.Request{
			Status:   "inactive",
			PageSize: 10,
		})
		require.NoError(t, err)
		assert.GreaterOrEqual(t, len(result.Products), 3) // All created products are inactive
	})
}
