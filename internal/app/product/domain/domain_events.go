package domain

import "time"

// DomainEvent is the base interface for all domain events.
type DomainEvent interface {
	EventType() string
	AggregateID() string
}

// ProductCreatedEvent is emitted when a product is created.
type ProductCreatedEvent struct {
	ProductID   string
	Name        string
	Description string
	Category    string
	BasePrice   *Money
	Status      string
	CreatedAt   time.Time
}

func (e *ProductCreatedEvent) EventType() string {
	return "product.created"
}

func (e *ProductCreatedEvent) AggregateID() string {
	return e.ProductID
}

// ProductUpdatedEvent is emitted when product details are updated.
type ProductUpdatedEvent struct {
	ProductID   string
	Name        string
	Description string
	Category    string
	UpdatedAt   time.Time
}

func (e *ProductUpdatedEvent) EventType() string {
	return "product.updated"
}

func (e *ProductUpdatedEvent) AggregateID() string {
	return e.ProductID
}

// ProductActivatedEvent is emitted when a product is activated.
type ProductActivatedEvent struct {
	ProductID string
	Timestamp time.Time
}

func (e *ProductActivatedEvent) EventType() string {
	return "product.activated"
}

func (e *ProductActivatedEvent) AggregateID() string {
	return e.ProductID
}

// ProductDeactivatedEvent is emitted when a product is deactivated.
type ProductDeactivatedEvent struct {
	ProductID string
	Timestamp time.Time
}

func (e *ProductDeactivatedEvent) EventType() string {
	return "product.deactivated"
}

func (e *ProductDeactivatedEvent) AggregateID() string {
	return e.ProductID
}

// DiscountAppliedEvent is emitted when a discount is applied to a product.
type DiscountAppliedEvent struct {
	ProductID        string
	DiscountPercent  int64
	DiscountStartDate time.Time
	DiscountEndDate   time.Time
	AppliedAt        time.Time
}

func (e *DiscountAppliedEvent) EventType() string {
	return "product.discount.applied"
}

func (e *DiscountAppliedEvent) AggregateID() string {
	return e.ProductID
}

// DiscountRemovedEvent is emitted when a discount is removed from a product.
type DiscountRemovedEvent struct {
	ProductID string
	RemovedAt time.Time
}

func (e *DiscountRemovedEvent) EventType() string {
	return "product.discount.removed"
}

func (e *DiscountRemovedEvent) AggregateID() string {
	return e.ProductID
}

// ProductArchivedEvent is emitted when a product is archived (soft deleted).
type ProductArchivedEvent struct {
	ProductID  string
	ArchivedAt time.Time
}

func (e *ProductArchivedEvent) EventType() string {
	return "product.archived"
}

func (e *ProductArchivedEvent) AggregateID() string {
	return e.ProductID
}
