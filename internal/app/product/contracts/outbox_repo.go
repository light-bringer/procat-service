package contracts

import (
	"cloud.google.com/go/spanner"
	"github.com/light-bringer/procat-service/internal/app/product/domain"
)

// OutboxEvent represents an enriched domain event ready for persistence.
type OutboxEvent struct {
	EventID     string
	EventType   string
	AggregateID string
	Payload     string // JSON
	Status      string
}

// OutboxRepository defines the interface for outbox event persistence.
type OutboxRepository interface {
	// InsertMut creates a mutation for inserting an outbox event
	InsertMut(event *OutboxEvent) *spanner.Mutation

	// EnrichEvent converts a domain event to an outbox event with metadata
	EnrichEvent(event domain.DomainEvent, payload string) *OutboxEvent
}
