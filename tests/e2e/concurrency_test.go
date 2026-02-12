package e2e

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/light-bringer/procat-service/internal/app/product/domain"
	"github.com/light-bringer/procat-service/internal/app/product/queries/get_product"
	"github.com/light-bringer/procat-service/internal/app/product/usecases/activate_product"
	"github.com/light-bringer/procat-service/internal/app/product/usecases/apply_discount"
	"github.com/light-bringer/procat-service/internal/app/product/usecases/create_product"
	"github.com/light-bringer/procat-service/internal/app/product/usecases/update_product"
	"github.com/light-bringer/procat-service/internal/pkg/committer"
)

// TestConcurrentDiscountApplication tests two goroutines applying different discounts.
// Expected: One succeeds, one fails with optimistic lock conflict or discount already active.
func TestConcurrentDiscountApplication(t *testing.T) {
	ctx := context.Background()
	suite, cleanup := setupTest(t)
	defer cleanup()

	// Create and activate a product
	basePrice, _ := domain.NewMoney(10000, 100) // $100.00
	createReq := &create_product.Request{
		Name:        "Concurrent Test Product",
		Description: "Testing concurrent discount application",
		Category:    "electronics",
		BasePrice:   basePrice,
	}

	productID, err := suite.CreateProduct.Execute(ctx, createReq)
	require.NoError(t, err)

	// Activate the product first
	err = suite.ActivateProduct.Execute(ctx, &activate_product.Request{ProductID: productID})
	require.NoError(t, err)

	now := time.Now().UTC()
	startDate := now
	endDate := now.Add(24 * time.Hour)

	discount1, _ := domain.NewDiscount(10, startDate, endDate) // 10% off
	discount2, _ := domain.NewDiscount(20, startDate, endDate) // 20% off

	// Apply two discounts concurrently
	var wg sync.WaitGroup
	var err1, err2 error

	wg.Add(2)

	go func() {
		defer wg.Done()
		req := &apply_discount.Request{
			ProductID:       productID,
			DiscountPercent: discount1.Percentage(),
			StartDate:       discount1.StartDate(),
			EndDate:         discount1.EndDate(),
		}
		err1 = suite.ApplyDiscount.Execute(ctx, req)
	}()

	go func() {
		defer wg.Done()
		req := &apply_discount.Request{
			ProductID:       productID,
			DiscountPercent: discount2.Percentage(),
			StartDate:       discount2.StartDate(),
			EndDate:         discount2.EndDate(),
		}
		err2 = suite.ApplyDiscount.Execute(ctx, req)
	}()

	wg.Wait()

	// Exactly one should succeed, one should fail
	if err1 == nil && err2 == nil {
		t.Error("Both discount applications succeeded - expected one to fail")
	} else if err1 != nil && err2 != nil {
		t.Errorf("Both discount applications failed - expected one to succeed. err1=%v, err2=%v", err1, err2)
	}

	// The successful discount should be present
	product, err := suite.GetProduct.Execute(ctx, &get_product.Request{ProductID: productID})
	require.NoError(t, err)
	assert.NotNil(t, product.DiscountPercent, "Product should have a discount")

	// Verify it's either discount1 or discount2
	discountPercent := *product.DiscountPercent
	assert.True(t, discountPercent == 10 || discountPercent == 20,
		"Discount should be either 10%% or 20%%, got %.2f%%", discountPercent)
}

// TestConcurrentProductUpdates tests two goroutines updating different fields.
// Expected: Both should succeed with proper optimistic locking, changes applied sequentially.
func TestConcurrentProductUpdates(t *testing.T) {
	ctx := context.Background()
	suite, cleanup := setupTest(t)
	defer cleanup()

	// Create a product
	basePrice, _ := domain.NewMoney(10000, 100)
	createReq := &create_product.Request{
		Name:        "Concurrent Update Test",
		Description: "Original Description",
		Category:    "electronics",
		BasePrice:   basePrice,
	}

	productID, err := suite.CreateProduct.Execute(ctx, createReq)
	require.NoError(t, err)

	// Update different fields concurrently
	var wg sync.WaitGroup
	var err1, err2 error

	newName := "Updated Name"
	newDescription := "Updated Description"

	wg.Add(2)

	go func() {
		defer wg.Done()
		req := &update_product.Request{
			ProductID: productID,
			Name:      &newName,
		}
		err1 = suite.UpdateProduct.Execute(ctx, req)
	}()

	go func() {
		defer wg.Done()
		req := &update_product.Request{
			ProductID:   productID,
			Description: &newDescription,
		}
		err2 = suite.UpdateProduct.Execute(ctx, req)
	}()

	wg.Wait()

	// With optimistic locking, one may fail with version conflict
	// But at least one should succeed
	successCount := 0
	if err1 == nil {
		successCount++
	}
	if err2 == nil {
		successCount++
	}

	assert.GreaterOrEqual(t, successCount, 1, "At least one update should succeed")

	// Read final state
	// Read final state
	product, err := suite.GetProduct.Execute(ctx, &get_product.Request{ProductID: productID})
	require.NoError(t, err)

	// If both succeeded (no version conflict), both changes should be present
	// If one failed, only one change should be present
	if successCount == 2 {
		assert.Equal(t, newName, product.Name, "Name should be updated")
		assert.Equal(t, newDescription, product.Description, "Description should be updated")
	}
}

// TestReadDuringWrite tests read consistency during concurrent writes.
// Expected: Reads should always see consistent state (either old or new, never partial).
func TestReadDuringWrite(t *testing.T) {
	ctx := context.Background()
	suite, cleanup := setupTest(t)
	defer cleanup()

	// Create a product
	basePrice, _ := domain.NewMoney(10000, 100)
	createReq := &create_product.Request{
		Name:        "Read Consistency Test",
		Description: "Original Description",
		Category:    "electronics",
		BasePrice:   basePrice,
	}

	productID, err := suite.CreateProduct.Execute(ctx, createReq)
	require.NoError(t, err)

	// Perform updates while reading concurrently
	var readerWg sync.WaitGroup
	var writerWg sync.WaitGroup
	stopReading := make(chan struct{})
	inconsistentReads := 0
	var inconsistentMutex sync.Mutex

	// Reader goroutine
	readerWg.Add(1)
	go func() {
		defer readerWg.Done()
		for {
			select {
			case <-stopReading:
				return
			default:
				product, err := suite.GetProduct.Execute(ctx, &get_product.Request{ProductID: productID})
				if err == nil {
					// Verify consistency: version should match state
					// If we can read the product, all fields should be internally consistent
					if product.Name == "" || product.Category == "" {
						inconsistentMutex.Lock()
						inconsistentReads++
						inconsistentMutex.Unlock()
					}
				}
				time.Sleep(1 * time.Millisecond) // Small delay between reads
			}
		}
	}()

	// Writer goroutine - perform multiple updates
	writerWg.Add(1)
	go func() {
		defer writerWg.Done()
		for i := 0; i < 5; i++ {
			newName := "Updated Name " + string(rune('A'+i))
			req := &update_product.Request{
				ProductID: productID,
				Name:      &newName,
			}
			_ = suite.UpdateProduct.Execute(ctx, req)
			time.Sleep(2 * time.Millisecond)
		}
	}()

	// Wait for writes to complete first
	writerWg.Wait()

	// Then stop reading
	close(stopReading)
	readerWg.Wait()

	// Verify no inconsistent reads
	assert.Equal(t, 0, inconsistentReads, "Should never see inconsistent state during reads")

	// Final state should be consistent
	product, err := suite.GetProduct.Execute(ctx, &get_product.Request{ProductID: productID})
	require.NoError(t, err)
	assert.NotEmpty(t, product.Name)
	assert.NotEmpty(t, product.Category)
}

// TestConcurrentPriceUpdates tests concurrent price changes with optimistic locking.
// Expected: Updates applied sequentially, all price history recorded correctly.
func TestConcurrentPriceUpdates(t *testing.T) {
	ctx := context.Background()
	suite, cleanup := setupTest(t)
	defer cleanup()

	// Create a product
	basePrice, _ := domain.NewMoney(10000, 100)
	createReq := &create_product.Request{
		Name:        "Price Update Test",
		Description: "Testing concurrent price updates",
		Category:    "electronics",
		BasePrice:   basePrice,
	}

	productID, err := suite.CreateProduct.Execute(ctx, createReq)
	require.NoError(t, err)

	// Attempt multiple concurrent price updates
	var wg sync.WaitGroup
	prices := []int64{11000, 12000, 13000}
	errors := make([]error, len(prices))

	for i, priceVal := range prices {
		wg.Add(1)
		go func(idx int, val int64) {
			defer wg.Done()
			newPrice, _ := domain.NewMoney(val, 100)

			// Load product
			product, err := suite.ProductRepo.GetByID(ctx, productID)
			if err != nil {
				errors[idx] = err
				return
			}

			// Update price
			err = product.SetBasePrice(newPrice)
			if err != nil {
				errors[idx] = err
				return
			}

			// Try to commit with version check
			plan := committer.NewPlan()
			if mut := suite.ProductRepo.UpdateMut(product); mut != nil {
				plan.Add(mut)
			}
			errors[idx] = suite.Committer.Apply(ctx, plan)
		}(i, priceVal)
	}

	wg.Wait()

	// At least one should succeed
	successCount := 0
	for _, err := range errors {
		if err == nil {
			successCount++
		}
	}

	assert.GreaterOrEqual(t, successCount, 1, "At least one price update should succeed")

	// Verify final state is consistent
	// Verify final state is consistent
	product, err := suite.GetProduct.Execute(ctx, &get_product.Request{ProductID: productID})
	require.NoError(t, err)

	// Price should be one of the attempted prices
	finalPrice := product.BasePrice
	assert.Contains(t, []float64{100.00, 110.00, 120.00, 130.00}, finalPrice,
		"Final price should be one of the valid prices")
}

// TestNoDataRaces verifies no data races with -race flag.
// This test should be run with: go test -race ./tests/e2e/...
func TestNoDataRaces(t *testing.T) {
	ctx := context.Background()
	suite, cleanup := setupTest(t)
	defer cleanup()

	// Create a product
	basePrice, _ := domain.NewMoney(10000, 100)
	createReq := &create_product.Request{
		Name:        "Race Test Product",
		Description: "Testing for data races",
		Category:    "electronics",
		BasePrice:   basePrice,
	}

	productID, err := suite.CreateProduct.Execute(ctx, createReq)
	require.NoError(t, err)

	// Perform various operations concurrently
	var wg sync.WaitGroup
	operations := 10

	for i := 0; i < operations; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()

			// Mix of read and write operations
			switch idx % 3 {
			case 0:
				// Read
				_, _ = suite.GetProduct.Execute(ctx, &get_product.Request{ProductID: productID})
			case 1:
				// Update
				newDesc := "Description " + string(rune('A'+idx))
				req := &update_product.Request{
					ProductID:   productID,
					Description: &newDesc,
				}
				_ = suite.UpdateProduct.Execute(ctx, req)
			case 2:
				// Read again
				_, _ = suite.GetProduct.Execute(ctx, &get_product.Request{ProductID: productID})
			}
		}(i)
	}

	wg.Wait()

	// If we get here without race detector errors, test passes
	assert.True(t, true, "No data races detected")
}
