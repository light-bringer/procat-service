package domain

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/light-bringer/procat-service/internal/pkg/clock"
)

func TestNewProduct(t *testing.T) {
	price, _ := NewMoney(100, 1)
	now := time.Now()
	clk := clock.NewMockClock(now)

	t.Run("valid product creation", func(t *testing.T) {
		p, err := NewProduct("id-1", "Test Product", "Description", "electronics", price, now, clk)
		require.NoError(t, err)
		assert.Equal(t, "id-1", p.ID())
		assert.Equal(t, "Test Product", p.Name())
		assert.Equal(t, StatusInactive, p.Status())
		assert.True(t, p.Changes().HasChanges())
		assert.Len(t, p.DomainEvents(), 1)
	})

	t.Run("empty name returns error", func(t *testing.T) {
		_, err := NewProduct("id-1", "", "Description", "electronics", price, now, clk)
		assert.ErrorIs(t, err, ErrEmptyName)
	})

	t.Run("empty category returns error", func(t *testing.T) {
		_, err := NewProduct("id-1", "Test", "Description", "", price, now, clk)
		assert.ErrorIs(t, err, ErrInvalidCategory)
	})

	t.Run("negative price returns error", func(t *testing.T) {
		negativePrice, _ := NewMoney(-100, 1)
		_, err := NewProduct("id-1", "Test", "Description", "electronics", negativePrice, now, clk)
		assert.ErrorIs(t, err, ErrInvalidPrice)
	})

	t.Run("zero price returns error", func(t *testing.T) {
		zeroPrice, _ := NewMoney(0, 1)
		_, err := NewProduct("id-1", "Test", "Description", "electronics", zeroPrice, now, clk)
		assert.ErrorIs(t, err, ErrInvalidPrice)
	})
}

func TestProduct_SetDescription(t *testing.T) {
	price, _ := NewMoney(100, 1)
	now := time.Now().UTC()
	clk := clock.NewMockClock(now)

	t.Run("empty description is allowed", func(t *testing.T) {
		p, _ := NewProduct("id-1", "Test Product", "Original Description", "electronics", price, now, clk)

		err := p.SetDescription("")
		require.NoError(t, err)
		assert.Equal(t, "", p.Description())
		assert.True(t, p.Changes().Dirty(FieldDescription))
	})

	t.Run("updating description marks field as dirty", func(t *testing.T) {
		p, _ := NewProduct("id-2", "Test Product", "Original", "electronics", price, now, clk)
		p.Changes().Clear() // Clear initial dirty state

		err := p.SetDescription("New Description")
		require.NoError(t, err)
		assert.Equal(t, "New Description", p.Description())
		assert.True(t, p.Changes().Dirty(FieldDescription))
	})

	t.Run("cannot update description on archived product", func(t *testing.T) {
		p, _ := NewProduct("id-3", "Test Product", "Description", "electronics", price, now, clk)
		p.Archive(now)

		err := p.SetDescription("New Description")
		assert.ErrorIs(t, err, ErrCannotModifyArchived)
	})
}

func TestProduct_SetBasePrice(t *testing.T) {
	originalPrice, _ := NewMoney(100, 1)
	now := time.Now().UTC()
	clk := clock.NewMockClock(now)

	t.Run("successfully update base price", func(t *testing.T) {
		p, _ := NewProduct("id-1", "Test Product", "Description", "electronics", originalPrice, now, clk)
		p.Changes().Clear() // Clear initial state

		newPrice, _ := NewMoney(150, 1)
		err := p.SetBasePrice(newPrice)
		require.NoError(t, err)

		// Verify price was updated
		updatedPrice := p.BasePrice()
		priceVal, _ := updatedPrice.Float64()
		assert.Equal(t, 150.0, priceVal)

		// Verify field marked as dirty
		assert.True(t, p.Changes().Dirty(FieldBasePrice))

		// Verify BasePriceChangedEvent was emitted
		events := p.DomainEvents()
		var foundPriceEvent bool
		for _, event := range events {
			if event.EventType() == "product.price.changed" {
				foundPriceEvent = true
				priceEvent := event.(*BasePriceChangedEvent)
				oldVal, _ := priceEvent.OldPrice.Float64()
				newVal, _ := priceEvent.NewPrice.Float64()
				assert.Equal(t, 100.0, oldVal)
				assert.Equal(t, 150.0, newVal)
			}
		}
		assert.True(t, foundPriceEvent, "BasePriceChangedEvent should be emitted")
	})

	t.Run("negative price returns error", func(t *testing.T) {
		p, _ := NewProduct("id-2", "Test Product", "Description", "electronics", originalPrice, now, clk)

		negativePrice, _ := NewMoney(-50, 1)
		err := p.SetBasePrice(negativePrice)
		assert.ErrorIs(t, err, ErrInvalidPrice)
	})

	t.Run("zero price returns error", func(t *testing.T) {
		p, _ := NewProduct("id-3", "Test Product", "Description", "electronics", originalPrice, now, clk)

		zeroPrice, _ := NewMoney(0, 1)
		err := p.SetBasePrice(zeroPrice)
		assert.ErrorIs(t, err, ErrInvalidPrice)
	})

	t.Run("cannot update price on archived product", func(t *testing.T) {
		p, _ := NewProduct("id-4", "Test Product", "Description", "electronics", originalPrice, now, clk)
		p.Archive(now)

		newPrice, _ := NewMoney(150, 1)
		err := p.SetBasePrice(newPrice)
		assert.ErrorIs(t, err, ErrCannotModifyArchived)
	})
}

func TestProduct_Activate(t *testing.T) {
	price, _ := NewMoney(100, 1)
	now := time.Now()
	clk := clock.NewMockClock(now)
	p, _ := NewProduct("id-1", "Test Product", "Description", "electronics", price, now, clk)

	err := p.Activate(now)
	require.NoError(t, err)
	assert.Equal(t, StatusActive, p.Status())
	assert.True(t, p.Changes().Dirty(FieldStatus))
	assert.Len(t, p.DomainEvents(), 2) // Created + Activated
}

func TestProduct_Deactivate(t *testing.T) {
	price, _ := NewMoney(100, 1)
	now := time.Now()
	clk := clock.NewMockClock(now)
	p, _ := NewProduct("id-1", "Test Product", "Description", "electronics", price, now, clk)
	p.Activate(now)

	err := p.Deactivate(now)
	require.NoError(t, err)
	assert.Equal(t, StatusInactive, p.Status())
}

func TestProduct_ApplyDiscount(t *testing.T) {
	price, _ := NewMoney(100, 1)
	now := time.Now().UTC()
	clk := clock.NewMockClock(now)
	p, _ := NewProduct("id-1", "Test Product", "Description", "electronics", price, now, clk)
	p.Activate(now)

	startDate := now
	endDate := now.Add(24 * time.Hour)
	discount, _ := NewDiscount(20, startDate, endDate)

	t.Run("apply discount to active product", func(t *testing.T) {
		err := p.ApplyDiscount(discount, now)
		require.NoError(t, err)
		assert.NotNil(t, p.Discount())
		assert.True(t, p.Changes().Dirty(FieldDiscount))
	})

	t.Run("cannot apply discount to inactive product", func(t *testing.T) {
		p2, _ := NewProduct("id-2", "Test", "Desc", "electronics", price, now, clk)
		err := p2.ApplyDiscount(discount, now)
		assert.ErrorIs(t, err, ErrCannotApplyToInactive)
	})

	t.Run("cannot apply discount when one already exists", func(t *testing.T) {
		discount2, _ := NewDiscount(30, startDate, endDate)
		err := p.ApplyDiscount(discount2, now)
		assert.ErrorIs(t, err, ErrDiscountAlreadyActive)
	})
}

func TestProduct_CalculateEffectivePrice(t *testing.T) {
	price, _ := NewMoney(100, 1)
	now := time.Now().UTC()
	clk := clock.NewMockClock(now)
	p, _ := NewProduct("id-1", "Test Product", "Description", "electronics", price, now, clk)

	t.Run("without discount returns base price", func(t *testing.T) {
		effectivePrice := p.CalculateEffectivePrice(now)
		val, _ := effectivePrice.Float64()
		assert.Equal(t, 100.0, val)
	})

	t.Run("with active discount returns discounted price", func(t *testing.T) {
		p.Activate(now)
		startDate := now.Add(-1 * time.Hour)
		endDate := now.Add(1 * time.Hour)
		discount, _ := NewDiscount(20, startDate, endDate)
		p.ApplyDiscount(discount, now)

		effectivePrice := p.CalculateEffectivePrice(now)
		val, _ := effectivePrice.Float64()
		assert.Equal(t, 80.0, val) // 100 - 20%
	})

	t.Run("with expired discount returns base price", func(t *testing.T) {
		futureTime := now.Add(2 * time.Hour)
		effectivePrice := p.CalculateEffectivePrice(futureTime)
		val, _ := effectivePrice.Float64()
		assert.Equal(t, 100.0, val)
	})
}

func TestProduct_Archive(t *testing.T) {
	price, _ := NewMoney(100, 1)
	now := time.Now()
	clk := clock.NewMockClock(now)
	p, _ := NewProduct("id-1", "Test Product", "Description", "electronics", price, now, clk)

	err := p.Archive(now)
	require.NoError(t, err)
	assert.Equal(t, StatusArchived, p.Status())
	assert.NotNil(t, p.ArchivedAt())
	assert.True(t, p.IsArchived())
}

func TestProduct_CannotModifyArchived(t *testing.T) {
	price, _ := NewMoney(100, 1)
	now := time.Now().UTC()
	clk := clock.NewMockClock(now)
	p, _ := NewProduct("id-1", "Test Product", "Description", "electronics", price, now, clk)
	p.Archive(now)

	t.Run("cannot set name", func(t *testing.T) {
		err := p.SetName("New Name")
		assert.ErrorIs(t, err, ErrCannotModifyArchived)
	})

	t.Run("cannot activate", func(t *testing.T) {
		err := p.Activate(now)
		assert.ErrorIs(t, err, ErrCannotModifyArchived)
	})
}

func TestProduct_HasDiscount(t *testing.T) {
	price, _ := NewMoney(100, 1)
	now := time.Now().UTC()
	clk := clock.NewMockClock(now)

	t.Run("returns false when no discount", func(t *testing.T) {
		p, _ := NewProduct("id-1", "Test Product", "Description", "electronics", price, now, clk)
		assert.False(t, p.HasDiscount())
	})

	t.Run("returns true when discount exists", func(t *testing.T) {
		p, _ := NewProduct("id-2", "Test Product", "Description", "electronics", price, now, clk)
		p.Activate(now)
		discount, _ := NewDiscount(20, now, now.Add(24*time.Hour))
		p.ApplyDiscount(discount, now)
		assert.True(t, p.HasDiscount())
	})
}

func TestProduct_DiscountCopy(t *testing.T) {
	price, _ := NewMoney(100, 1)
	now := time.Now().UTC()
	clk := clock.NewMockClock(now)

	t.Run("returns nil when no discount", func(t *testing.T) {
		p, _ := NewProduct("id-1", "Test Product", "Description", "electronics", price, now, clk)
		assert.Nil(t, p.DiscountCopy())
	})

	t.Run("returns copy of discount", func(t *testing.T) {
		p, _ := NewProduct("id-2", "Test Product", "Description", "electronics", price, now, clk)
		p.Activate(now)
		startDate := now
		endDate := now.Add(24 * time.Hour)
		discount, _ := NewDiscount(20, startDate, endDate)
		p.ApplyDiscount(discount, now)

		copy := p.DiscountCopy()
		require.NotNil(t, copy)
		assert.Equal(t, int64(20), copy.Percentage())
		assert.Equal(t, startDate, copy.StartDate())
		assert.Equal(t, endDate, copy.EndDate())
	})

	t.Run("returned copy is independent of original", func(t *testing.T) {
		p, _ := NewProduct("id-3", "Test Product", "Description", "electronics", price, now, clk)
		p.Activate(now)
		discount, _ := NewDiscount(20, now, now.Add(24*time.Hour))
		p.ApplyDiscount(discount, now)

		copy := p.DiscountCopy()

		// Verify the copy has same values
		assert.Equal(t, p.Discount().Percentage(), copy.Percentage())

		// Remove discount from product
		p.RemoveDiscount(now)

		// Verify copy is still valid and unchanged
		assert.Nil(t, p.DiscountCopy())
		assert.Equal(t, int64(20), copy.Percentage())
	})
}

func TestProduct_Archive_RemovesDiscount(t *testing.T) {
	price, _ := NewMoney(100, 1)
	now := time.Now().UTC()
	clk := clock.NewMockClock(now)

	t.Run("archiving product with discount removes it", func(t *testing.T) {
		p, _ := NewProduct("id-1", "Test Product", "Description", "electronics", price, now, clk)
		p.Activate(now)

		// Apply discount
		discount, _ := NewDiscount(20, now, now.Add(24*time.Hour))
		p.ApplyDiscount(discount, now)
		require.True(t, p.HasDiscount())

		// Archive product
		err := p.Archive(now)
		require.NoError(t, err)

		// Verify discount was removed
		assert.False(t, p.HasDiscount())
		assert.Nil(t, p.DiscountCopy())
		assert.Equal(t, StatusArchived, p.Status())

		// Verify DiscountRemovedEvent and ProductArchivedEvent were emitted
		events := p.DomainEvents()
		var hasDiscountRemoved, hasArchived bool
		for _, event := range events {
			switch event.EventType() {
			case "product.discount.removed":
				hasDiscountRemoved = true
			case "product.archived":
				hasArchived = true
			}
		}
		assert.True(t, hasDiscountRemoved, "DiscountRemovedEvent should be emitted")
		assert.True(t, hasArchived, "ProductArchivedEvent should be emitted")
	})

	t.Run("archiving product without discount works normally", func(t *testing.T) {
		p, _ := NewProduct("id-2", "Test Product", "Description", "electronics", price, now, clk)

		err := p.Archive(now)
		require.NoError(t, err)

		assert.False(t, p.HasDiscount())
		assert.Equal(t, StatusArchived, p.Status())

		// Verify only ProductArchivedEvent was emitted (no DiscountRemovedEvent)
		events := p.DomainEvents()
		var hasDiscountRemoved bool
		for _, event := range events {
			if event.EventType() == "product.discount.removed" {
				hasDiscountRemoved = true
			}
		}
		assert.False(t, hasDiscountRemoved, "DiscountRemovedEvent should not be emitted when no discount exists")
	})
}

func TestProduct_MarkUpdated(t *testing.T) {
	price, _ := NewMoney(100, 1)
	now := time.Now().UTC()
	clk := clock.NewMockClock(now)

	t.Run("field updates don't emit events until MarkUpdated is called", func(t *testing.T) {
		p, _ := NewProduct("id-1", "Test Product", "Description", "electronics", price, now, clk)
		p.ClearEvents() // Clear creation event

		// Update multiple fields
		err := p.SetName("New Name")
		require.NoError(t, err)
		err = p.SetDescription("New Description")
		require.NoError(t, err)
		err = p.SetCategory("furniture")
		require.NoError(t, err)

		// No events should be emitted yet
		assert.Len(t, p.DomainEvents(), 0, "No events should be emitted before MarkUpdated")

		// Call MarkUpdated to emit single consolidated event
		updatedTime := now.Add(1 * time.Hour)
		p.MarkUpdated(updatedTime)

		// Verify single ProductUpdatedEvent was emitted
		events := p.DomainEvents()
		require.Len(t, events, 1, "Should emit exactly one ProductUpdatedEvent")

		event, ok := events[0].(*ProductUpdatedEvent)
		require.True(t, ok, "Event should be ProductUpdatedEvent")
		assert.Equal(t, "product.updated", event.EventType())
		assert.Equal(t, "id-1", event.ProductID)
		assert.Equal(t, "New Name", event.Name)
		assert.Equal(t, "New Description", event.Description)
		assert.Equal(t, "furniture", event.Category)
		assert.Equal(t, updatedTime, event.UpdatedAt)
	})

	t.Run("MarkUpdated can be called multiple times", func(t *testing.T) {
		p, _ := NewProduct("id-2", "Product", "Desc", "electronics", price, now, clk)
		p.ClearEvents()

		// First update
		p.SetName("Name1")
		p.MarkUpdated(now)
		assert.Len(t, p.DomainEvents(), 1)

		p.ClearEvents()

		// Second update
		p.SetDescription("Desc2")
		p.MarkUpdated(now.Add(1 * time.Hour))
		assert.Len(t, p.DomainEvents(), 1)
	})
}
