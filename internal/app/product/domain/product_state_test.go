package domain

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/light-bringer/procat-service/internal/pkg/clock"
)

// TestProductStateMachine verifies all valid and invalid state transitions.
func TestProductStateMachine(t *testing.T) {
	now := time.Now().UTC()
	clk := clock.NewMockClock(now)
	price, _ := NewMoney(10000, 100)

	// State transition matrix:
	// From\To    | Inactive | Active | Archived
	// -----------|----------|--------|----------
	// Inactive   | N/A      | ✓      | ✓
	// Active     | ✓        | N/A    | ✓
	// Archived   | ✗        | ✗      | N/A

	t.Run("Inactive → Active: allowed", func(t *testing.T) {
		p, _ := NewProduct("id-1", "Product", "Desc", "electronics", price, now, clk)
		assert.Equal(t, StatusInactive, p.Status())

		err := p.Activate(now)
		require.NoError(t, err)
		assert.Equal(t, StatusActive, p.Status())
	})

	t.Run("Inactive → Archived: allowed", func(t *testing.T) {
		p, _ := NewProduct("id-2", "Product", "Desc", "electronics", price, now, clk)
		assert.Equal(t, StatusInactive, p.Status())

		err := p.Archive(now)
		require.NoError(t, err)
		assert.Equal(t, StatusArchived, p.Status())
	})

	t.Run("Active → Inactive: allowed", func(t *testing.T) {
		p, _ := NewProduct("id-3", "Product", "Desc", "electronics", price, now, clk)
		p.Activate(now)
		assert.Equal(t, StatusActive, p.Status())

		err := p.Deactivate(now)
		require.NoError(t, err)
		assert.Equal(t, StatusInactive, p.Status())
	})

	t.Run("Active → Archived: allowed", func(t *testing.T) {
		p, _ := NewProduct("id-4", "Product", "Desc", "electronics", price, now, clk)
		p.Activate(now)
		assert.Equal(t, StatusActive, p.Status())

		err := p.Archive(now)
		require.NoError(t, err)
		assert.Equal(t, StatusArchived, p.Status())
	})

	t.Run("Archived → Active: forbidden", func(t *testing.T) {
		p, _ := NewProduct("id-5", "Product", "Desc", "electronics", price, now, clk)
		p.Archive(now)
		assert.Equal(t, StatusArchived, p.Status())

		err := p.Activate(now)
		assert.ErrorIs(t, err, ErrCannotModifyArchived)
		assert.Equal(t, StatusArchived, p.Status()) // Status unchanged
	})

	t.Run("Archived → Inactive: forbidden", func(t *testing.T) {
		p, _ := NewProduct("id-6", "Product", "Desc", "electronics", price, now, clk)
		p.Activate(now)
		p.Archive(now)
		assert.Equal(t, StatusArchived, p.Status())

		err := p.Deactivate(now)
		assert.ErrorIs(t, err, ErrCannotModifyArchived)
		assert.Equal(t, StatusArchived, p.Status()) // Status unchanged
	})

	t.Run("Active → Active: idempotent error", func(t *testing.T) {
		p, _ := NewProduct("id-7", "Product", "Desc", "electronics", price, now, clk)
		p.Activate(now)
		assert.Equal(t, StatusActive, p.Status())

		err := p.Activate(now)
		assert.ErrorIs(t, err, ErrAlreadyActive)
		assert.Equal(t, StatusActive, p.Status())
	})

	t.Run("Inactive → Inactive: idempotent error", func(t *testing.T) {
		p, _ := NewProduct("id-8", "Product", "Desc", "electronics", price, now, clk)
		assert.Equal(t, StatusInactive, p.Status())

		err := p.Deactivate(now)
		assert.ErrorIs(t, err, ErrAlreadyInactive)
		assert.Equal(t, StatusInactive, p.Status())
	})

	t.Run("Archived → Archived: idempotent error", func(t *testing.T) {
		p, _ := NewProduct("id-9", "Product", "Desc", "electronics", price, now, clk)
		p.Archive(now)
		assert.Equal(t, StatusArchived, p.Status())

		err := p.Archive(now)
		assert.ErrorIs(t, err, ErrAlreadyArchived)
		assert.Equal(t, StatusArchived, p.Status())
	})
}

// TestArchivedProductCannotBeModified verifies all modification operations fail on archived products.
func TestArchivedProductCannotBeModified(t *testing.T) {
	now := time.Now().UTC()
	clk := clock.NewMockClock(now)
	price, _ := NewMoney(10000, 100)

	t.Run("cannot set name", func(t *testing.T) {
		p, _ := NewProduct("id-1", "Product", "Desc", "electronics", price, now, clk)
		p.Archive(now)

		err := p.SetName("New Name")
		assert.ErrorIs(t, err, ErrCannotModifyArchived)
	})

	t.Run("cannot set description", func(t *testing.T) {
		p, _ := NewProduct("id-2", "Product", "Desc", "electronics", price, now, clk)
		p.Archive(now)

		err := p.SetDescription("New Desc")
		assert.ErrorIs(t, err, ErrCannotModifyArchived)
	})

	t.Run("cannot set category", func(t *testing.T) {
		p, _ := NewProduct("id-3", "Product", "Desc", "electronics", price, now, clk)
		p.Archive(now)

		err := p.SetCategory("furniture")
		assert.ErrorIs(t, err, ErrCannotModifyArchived)
	})

	t.Run("cannot set base price", func(t *testing.T) {
		p, _ := NewProduct("id-4", "Product", "Desc", "electronics", price, now, clk)
		p.Archive(now)

		newPrice, _ := NewMoney(15000, 100)
		err := p.SetBasePrice(newPrice)
		assert.ErrorIs(t, err, ErrCannotModifyArchived)
	})

	t.Run("cannot apply discount", func(t *testing.T) {
		p, _ := NewProduct("id-5", "Product", "Desc", "electronics", price, now, clk)
		p.Activate(now)
		p.Archive(now)

		discount, _ := NewDiscount(20, now, now.Add(24*time.Hour))
		err := p.ApplyDiscount(discount, now)
		assert.ErrorIs(t, err, ErrCannotModifyArchived)
	})

	t.Run("cannot remove discount", func(t *testing.T) {
		p, _ := NewProduct("id-6", "Product", "Desc", "electronics", price, now, clk)
		p.Activate(now)
		discount, _ := NewDiscount(20, now, now.Add(24*time.Hour))
		p.ApplyDiscount(discount, now)
		p.Archive(now) // This removes the discount

		// Archive already removed discount, but trying to remove again should still fail
		err := p.RemoveDiscount(now)
		assert.ErrorIs(t, err, ErrCannotModifyArchived)
	})

	t.Run("cannot activate", func(t *testing.T) {
		p, _ := NewProduct("id-7", "Product", "Desc", "electronics", price, now, clk)
		p.Archive(now)

		err := p.Activate(now)
		assert.ErrorIs(t, err, ErrCannotModifyArchived)
	})

	t.Run("cannot deactivate", func(t *testing.T) {
		p, _ := NewProduct("id-8", "Product", "Desc", "electronics", price, now, clk)
		p.Activate(now)
		p.Archive(now)

		err := p.Deactivate(now)
		assert.ErrorIs(t, err, ErrCannotModifyArchived)
	})
}

// TestDiscountOnlyOnActiveProducts verifies discount operations respect product status.
func TestDiscountOnlyOnActiveProducts(t *testing.T) {
	now := time.Now().UTC()
	clk := clock.NewMockClock(now)
	price, _ := NewMoney(10000, 100)
	discount, _ := NewDiscount(20, now, now.Add(24*time.Hour))

	t.Run("can apply discount to active product", func(t *testing.T) {
		p, _ := NewProduct("id-1", "Product", "Desc", "electronics", price, now, clk)
		p.Activate(now)

		err := p.ApplyDiscount(discount, now)
		require.NoError(t, err)
		assert.NotNil(t, p.Discount())
	})

	t.Run("cannot apply discount to inactive product", func(t *testing.T) {
		p, _ := NewProduct("id-2", "Product", "Desc", "electronics", price, now, clk)
		assert.Equal(t, StatusInactive, p.Status())

		err := p.ApplyDiscount(discount, now)
		assert.ErrorIs(t, err, ErrCannotApplyToInactive)
	})

	t.Run("cannot apply discount to archived product", func(t *testing.T) {
		p, _ := NewProduct("id-3", "Product", "Desc", "electronics", price, now, clk)
		p.Archive(now)

		err := p.ApplyDiscount(discount, now)
		assert.ErrorIs(t, err, ErrCannotModifyArchived)
	})

	t.Run("can remove discount from active product", func(t *testing.T) {
		p, _ := NewProduct("id-4", "Product", "Desc", "electronics", price, now, clk)
		p.Activate(now)
		p.ApplyDiscount(discount, now)

		err := p.RemoveDiscount(now)
		require.NoError(t, err)
		assert.Nil(t, p.Discount())
	})

	t.Run("cannot apply second discount when one exists", func(t *testing.T) {
		p, _ := NewProduct("id-5", "Product", "Desc", "electronics", price, now, clk)
		p.Activate(now)
		p.ApplyDiscount(discount, now)

		discount2, _ := NewDiscount(30, now, now.Add(24*time.Hour))
		err := p.ApplyDiscount(discount2, now)
		assert.ErrorIs(t, err, ErrDiscountAlreadyActive)
	})
}

// TestStateTransitionEventEmission verifies correct events are emitted for each transition.
func TestStateTransitionEventEmission(t *testing.T) {
	now := time.Now().UTC()
	clk := clock.NewMockClock(now)
	price, _ := NewMoney(10000, 100)

	t.Run("Inactive → Active emits ProductActivatedEvent", func(t *testing.T) {
		p, _ := NewProduct("id-1", "Product", "Desc", "electronics", price, now, clk)
		p.ClearEvents()

		p.Activate(now)

		events := p.DomainEvents()
		require.Len(t, events, 1)
		assert.Equal(t, "product.activated", events[0].EventType())
	})

	t.Run("Active → Inactive emits ProductDeactivatedEvent", func(t *testing.T) {
		p, _ := NewProduct("id-2", "Product", "Desc", "electronics", price, now, clk)
		p.Activate(now)
		p.ClearEvents()

		p.Deactivate(now)

		events := p.DomainEvents()
		require.Len(t, events, 1)
		assert.Equal(t, "product.deactivated", events[0].EventType())
	})

	t.Run("Any → Archived emits ProductArchivedEvent", func(t *testing.T) {
		p, _ := NewProduct("id-3", "Product", "Desc", "electronics", price, now, clk)
		p.ClearEvents()

		p.Archive(now)

		events := p.DomainEvents()
		hasArchivedEvent := false
		for _, event := range events {
			if event.EventType() == "product.archived" {
				hasArchivedEvent = true
			}
		}
		assert.True(t, hasArchivedEvent)
	})

	t.Run("Archived with discount emits both DiscountRemoved and ProductArchived", func(t *testing.T) {
		p, _ := NewProduct("id-4", "Product", "Desc", "electronics", price, now, clk)
		p.Activate(now)
		discount, _ := NewDiscount(20, now, now.Add(24*time.Hour))
		p.ApplyDiscount(discount, now)
		p.ClearEvents()

		p.Archive(now)

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
		assert.True(t, hasDiscountRemoved, "Should emit DiscountRemovedEvent")
		assert.True(t, hasArchived, "Should emit ProductArchivedEvent")
	})
}
