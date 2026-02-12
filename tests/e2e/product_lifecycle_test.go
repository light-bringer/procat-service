package e2e

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/light-bringer/procat-service/internal/app/product/domain"
	"github.com/light-bringer/procat-service/internal/app/product/queries/get_product"
	"github.com/light-bringer/procat-service/internal/app/product/queries/list_products"
	"github.com/light-bringer/procat-service/internal/app/product/usecases/activate_product"
	"github.com/light-bringer/procat-service/internal/app/product/usecases/archive_product"
	"github.com/light-bringer/procat-service/internal/app/product/usecases/create_product"
	"github.com/light-bringer/procat-service/internal/app/product/usecases/deactivate_product"
	"github.com/light-bringer/procat-service/internal/app/product/usecases/update_product"
	"github.com/light-bringer/procat-service/tests/testutil"
)

func TestProductCreationFlow(t *testing.T) {
	services, cleanup := setupTest(t)
	defer cleanup()

	// Create a product
	price, _ := domain.NewMoney(249900, 100) // $2499.00
	req := &create_product.Request{
		Name:        "MacBook Pro",
		Description: "16-inch laptop",
		Category:    "electronics",
		BasePrice:   price,
	}

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
	price, _ := domain.NewMoney(10000, 100)
	productID, err := services.CreateProduct.Execute(ctx(), &create_product.Request{
		Name:        "Test Product",
		Description: "Test",
		Category:    "electronics",
		BasePrice:   price,
	})
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
	price, _ := domain.NewMoney(10000, 100)
	productID, err := services.CreateProduct.Execute(ctx(), &create_product.Request{
		Name:        "Original Name",
		Description: "Original Description",
		Category:    "electronics",
		BasePrice:   price,
	})
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
	price, _ := domain.NewMoney(10000, 100)
	productID, err := services.CreateProduct.Execute(ctx(), &create_product.Request{
		Name:        "To Archive",
		Description: "Test",
		Category:    "electronics",
		BasePrice:   price,
	})
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
}

func TestBusinessRuleValidations(t *testing.T) {
	services, cleanup := setupTest(t)
	defer cleanup()

	t.Run("cannot create product with empty name", func(t *testing.T) {
		price, _ := domain.NewMoney(10000, 100)
		_, err := services.CreateProduct.Execute(ctx(), &create_product.Request{
			Name:        "",
			Description: "Test",
			Category:    "electronics",
			BasePrice:   price,
		})
		assert.ErrorIs(t, err, domain.ErrEmptyName)
	})

	t.Run("cannot create product with negative price", func(t *testing.T) {
		price, _ := domain.NewMoney(-10000, 100)
		_, err := services.CreateProduct.Execute(ctx(), &create_product.Request{
			Name:        "Test",
			Description: "Test",
			Category:    "electronics",
			BasePrice:   price,
		})
		assert.ErrorIs(t, err, domain.ErrInvalidPrice)
	})

	t.Run("cannot activate already active product", func(t *testing.T) {
		price, _ := domain.NewMoney(10000, 100)
		productID, _ := services.CreateProduct.Execute(ctx(), &create_product.Request{
			Name:        "Test",
			Description: "Test",
			Category:    "electronics",
			BasePrice:   price,
		})

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
	price, _ := domain.NewMoney(10000, 100)
	services.CreateProduct.Execute(ctx(), &create_product.Request{
		Name: "Product 1", Description: "Test", Category: "electronics", BasePrice: price,
	})
	services.CreateProduct.Execute(ctx(), &create_product.Request{
		Name: "Product 2", Description: "Test", Category: "books", BasePrice: price,
	})
	services.CreateProduct.Execute(ctx(), &create_product.Request{
		Name: "Product 3", Description: "Test", Category: "electronics", BasePrice: price,
	})

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
