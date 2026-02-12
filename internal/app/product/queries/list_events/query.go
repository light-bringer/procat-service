package list_events

import (
	"context"

	"github.com/light-bringer/procat-service/internal/models/m_outbox"
)

// Request contains filtering parameters for listing events.
type Request struct {
	EventType   *string // Filter by event type (e.g., "product.created")
	AggregateID *string // Filter by aggregate ID
	Status      *string // Filter by status ("pending", "processed", "failed")
	Limit       int     // Max number of events to return (default: 100)
}

// EventsReadModel defines the interface for reading events.
type EventsReadModel interface {
	ListEvents(ctx context.Context, req *Request) ([]*m_outbox.Data, int64, error)
}

// Query handles the list events query use case.
type Query struct {
	readModel EventsReadModel
}

// NewQuery creates a new list events query.
func NewQuery(readModel EventsReadModel) *Query {
	return &Query{
		readModel: readModel,
	}
}

// Execute retrieves a list of events with filtering.
func (q *Query) Execute(ctx context.Context, req *Request) ([]*m_outbox.Data, int64, error) {
	if req.Limit <= 0 {
		req.Limit = 100 // Default limit
	}
	if req.Limit > 1000 {
		req.Limit = 1000 // Max limit
	}

	return q.readModel.ListEvents(ctx, req)
}
