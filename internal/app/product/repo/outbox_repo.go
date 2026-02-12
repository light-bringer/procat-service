package repo

import (
	"cloud.google.com/go/spanner"
	"github.com/google/uuid"
	"github.com/light-bringer/procat-service/internal/app/product/contracts"
	"github.com/light-bringer/procat-service/internal/app/product/domain"
	"github.com/light-bringer/procat-service/internal/models/m_outbox"
)

// OutboxRepo implements OutboxRepository for Spanner.
type OutboxRepo struct {
	client *spanner.Client
	model  *m_outbox.Model
}

// NewOutboxRepo creates a new OutboxRepo.
func NewOutboxRepo(client *spanner.Client) contracts.OutboxRepository {
	return &OutboxRepo{
		client: client,
		model:  m_outbox.NewModel(),
	}
}

// InsertMut creates a mutation for inserting an outbox event.
func (r *OutboxRepo) InsertMut(event *contracts.OutboxEvent) *spanner.Mutation {
	// Wrap payload string as JSON for Spanner
	payload := spanner.NullJSON{Value: event.Payload, Valid: event.Payload != ""}

	data := &m_outbox.Data{
		EventID:     event.EventID,
		EventType:   event.EventType,
		AggregateID: event.AggregateID,
		Payload:     payload,
		Status:      event.Status,
		RetryCount:  0,
	}

	return r.model.InsertMut(data)
}

// EnrichEvent converts a domain event to an outbox event with metadata.
func (r *OutboxRepo) EnrichEvent(event domain.DomainEvent, payload string) *contracts.OutboxEvent {
	return &contracts.OutboxEvent{
		EventID:     uuid.New().String(),
		EventType:   event.EventType(),
		AggregateID: event.AggregateID(),
		Payload:     payload,
		Status:      m_outbox.StatusPending,
	}
}
