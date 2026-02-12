package domain

import (
	"time"

	"github.com/light-bringer/procat-service/internal/pkg/clock"
)

// Field names for change tracking
const (
	FieldName        = "name"
	FieldDescription = "description"
	FieldCategory    = "category"
	FieldBasePrice   = "base_price"
	FieldDiscount    = "discount"
	FieldStatus      = "status"
	FieldArchivedAt  = "archived_at"
)

// ProductStatus represents the lifecycle status of a product
type ProductStatus string

const (
	StatusInactive ProductStatus = "inactive"
	StatusActive   ProductStatus = "active"
	StatusArchived ProductStatus = "archived"
)

// Product is the aggregate root for product management.
// It encapsulates all business logic related to products, pricing, and discounts.
type Product struct {
	id          string
	name        string
	description string
	category    string
	basePrice   *Money
	discount    *Discount
	status      ProductStatus
	createdAt   time.Time
	updatedAt   time.Time
	archivedAt  *time.Time

	// Clock for time operations (injected for testability)
	clock clock.Clock

	// Change tracking for optimized repository updates
	changes *ChangeTracker

	// Domain events to be published
	events []DomainEvent
}

// NewProduct creates a new Product aggregate (for creation).
func NewProduct(id, name, description, category string, basePrice *Money, now time.Time, clk clock.Clock) (*Product, error) {
	if name == "" {
		return nil, ErrEmptyName
	}

	if category == "" {
		return nil, ErrInvalidCategory
	}

	if basePrice.IsNegative() || basePrice.IsZero() {
		return nil, ErrInvalidPrice
	}

	p := &Product{
		id:          id,
		name:        name,
		description: description,
		category:    category,
		basePrice:   basePrice.Copy(),
		status:      StatusInactive,
		createdAt:   now,
		updatedAt:   now,
		clock:       clk,
		changes:     NewChangeTracker(),
		events:      make([]DomainEvent, 0),
	}

	// Mark all fields as dirty for new product
	p.changes.MarkDirty(FieldName)
	p.changes.MarkDirty(FieldDescription)
	p.changes.MarkDirty(FieldCategory)
	p.changes.MarkDirty(FieldBasePrice)
	p.changes.MarkDirty(FieldStatus)

	// Emit creation event
	p.recordEvent(&ProductCreatedEvent{
		ProductID:   p.id,
		Name:        p.name,
		Description: p.description,
		Category:    p.category,
		BasePrice:   p.basePrice.Copy(),
		Status:      string(p.status),
		CreatedAt:   p.createdAt,
	})

	return p, nil
}

// ReconstructProduct reconstitutes a Product from database (for loading existing products).
func ReconstructProduct(
	id, name, description, category string,
	basePrice *Money,
	discount *Discount,
	status ProductStatus,
	createdAt, updatedAt time.Time,
	archivedAt *time.Time,
	clk clock.Clock,
) *Product {
	return &Product{
		id:          id,
		name:        name,
		description: description,
		category:    category,
		basePrice:   basePrice,
		discount:    discount,
		status:      status,
		createdAt:   createdAt,
		updatedAt:   updatedAt,
		archivedAt:  archivedAt,
		clock:       clk,
		changes:     NewChangeTracker(), // Start with clean slate
		events:      make([]DomainEvent, 0),
	}
}

// Getters
func (p *Product) ID() string                 { return p.id }
func (p *Product) Name() string               { return p.name }
func (p *Product) Description() string        { return p.description }
func (p *Product) Category() string           { return p.category }
func (p *Product) BasePrice() *Money          { return p.basePrice.Copy() }
func (p *Product) Discount() *Discount        { return p.discount }
func (p *Product) Status() ProductStatus      { return p.status }
func (p *Product) CreatedAt() time.Time       { return p.createdAt }
func (p *Product) UpdatedAt() time.Time       { return p.updatedAt }
func (p *Product) ArchivedAt() *time.Time     { return p.archivedAt }
func (p *Product) Changes() *ChangeTracker    { return p.changes }
func (p *Product) DomainEvents() []DomainEvent { return p.events }

// SetName updates the product name.
func (p *Product) SetName(name string) error {
	if err := p.checkNotArchived(); err != nil {
		return err
	}

	if name == "" {
		return ErrEmptyName
	}

	p.name = name
	p.changes.MarkDirty(FieldName)

	// Record update event (consistent with other domain operations)
	p.recordEvent(&ProductUpdatedEvent{
		ProductID:   p.id,
		Name:        p.name,
		Description: p.description,
		Category:    p.category,
		UpdatedAt:   p.clock.Now(),
	})

	return nil
}

// SetDescription updates the product description.
func (p *Product) SetDescription(description string) error {
	if err := p.checkNotArchived(); err != nil {
		return err
	}

	p.description = description
	p.changes.MarkDirty(FieldDescription)

	p.recordEvent(&ProductUpdatedEvent{
		ProductID:   p.id,
		Name:        p.name,
		Description: p.description,
		Category:    p.category,
		UpdatedAt:   p.clock.Now(),
	})

	return nil
}

// SetCategory updates the product category.
func (p *Product) SetCategory(category string) error {
	if err := p.checkNotArchived(); err != nil {
		return err
	}

	if category == "" {
		return ErrInvalidCategory
	}

	p.category = category
	p.changes.MarkDirty(FieldCategory)

	p.recordEvent(&ProductUpdatedEvent{
		ProductID:   p.id,
		Name:        p.name,
		Description: p.description,
		Category:    p.category,
		UpdatedAt:   p.clock.Now(),
	})

	return nil
}

// ApplyDiscount applies a discount to the product.
func (p *Product) ApplyDiscount(discount *Discount, now time.Time) error {
	if err := p.checkNotArchived(); err != nil {
		return err
	}

	if p.status != StatusActive {
		return ErrCannotApplyToInactive
	}

	if p.discount != nil {
		return ErrDiscountAlreadyActive
	}

	p.discount = discount
	p.changes.MarkDirty(FieldDiscount)

	p.recordEvent(&DiscountAppliedEvent{
		ProductID:         p.id,
		DiscountPercent:   discount.Percentage(),
		DiscountStartDate: discount.StartDate(),
		DiscountEndDate:   discount.EndDate(),
		AppliedAt:         now,
	})

	return nil
}

// RemoveDiscount removes the active discount from the product.
func (p *Product) RemoveDiscount(now time.Time) error {
	if err := p.checkNotArchived(); err != nil {
		return err
	}

	p.discount = nil
	p.changes.MarkDirty(FieldDiscount)

	p.recordEvent(&DiscountRemovedEvent{
		ProductID: p.id,
		RemovedAt: now,
	})

	return nil
}

// Activate activates the product.
func (p *Product) Activate(now time.Time) error {
	if err := p.checkNotArchived(); err != nil {
		return err
	}

	if p.status == StatusActive {
		return ErrAlreadyActive
	}

	p.status = StatusActive
	p.changes.MarkDirty(FieldStatus)

	p.recordEvent(&ProductActivatedEvent{
		ProductID: p.id,
		Timestamp: now,
	})

	return nil
}

// Deactivate deactivates the product.
func (p *Product) Deactivate(now time.Time) error {
	if err := p.checkNotArchived(); err != nil {
		return err
	}

	if p.status == StatusInactive {
		return ErrAlreadyInactive
	}

	p.status = StatusInactive
	p.changes.MarkDirty(FieldStatus)

	p.recordEvent(&ProductDeactivatedEvent{
		ProductID: p.id,
		Timestamp: now,
	})

	return nil
}

// Archive archives the product (soft delete).
func (p *Product) Archive(now time.Time) error {
	if p.status == StatusArchived {
		return ErrAlreadyArchived
	}

	p.status = StatusArchived
	p.archivedAt = &now
	p.changes.MarkDirty(FieldStatus)
	p.changes.MarkDirty(FieldArchivedAt)

	p.recordEvent(&ProductArchivedEvent{
		ProductID:  p.id,
		ArchivedAt: now,
	})

	return nil
}

// CalculateEffectivePrice calculates the current price considering active discounts.
func (p *Product) CalculateEffectivePrice(now time.Time) *Money {
	if p.discount != nil && p.discount.IsValidAt(now) {
		return p.discount.Apply(p.basePrice)
	}
	return p.basePrice.Copy()
}

// IsActive returns true if the product is active.
func (p *Product) IsActive() bool {
	return p.status == StatusActive
}

// IsArchived returns true if the product is archived.
func (p *Product) IsArchived() bool {
	return p.status == StatusArchived
}

// HasActiveDiscount returns true if the product has a discount valid at the given time.
func (p *Product) HasActiveDiscount(now time.Time) bool {
	return p.discount != nil && p.discount.IsValidAt(now)
}

// checkNotArchived returns an error if the product is archived.
func (p *Product) checkNotArchived() error {
	if p.status == StatusArchived {
		return ErrCannotModifyArchived
	}
	return nil
}

// recordEvent adds a domain event to the list of events.
func (p *Product) recordEvent(event DomainEvent) {
	p.events = append(p.events, event)
}

// ClearEvents clears all recorded domain events (called after publishing).
func (p *Product) ClearEvents() {
	p.events = make([]DomainEvent, 0)
}
