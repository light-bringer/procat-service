package repo

import (
	"context"
	"fmt"

	"cloud.google.com/go/spanner"
	"google.golang.org/api/iterator"

	"github.com/light-bringer/procat-service/internal/app/product/queries/list_events"
	"github.com/light-bringer/procat-service/internal/models/m_outbox"
)

// EventsReadModel implements the EventsReadModel interface for Spanner.
type EventsReadModel struct {
	client *spanner.Client
}

// NewEventsReadModel creates a new EventsReadModel.
func NewEventsReadModel(client *spanner.Client) *EventsReadModel {
	return &EventsReadModel{
		client: client,
	}
}

// ListEvents retrieves events from the outbox_events table with filtering.
func (r *EventsReadModel) ListEvents(ctx context.Context, req *list_events.Request) ([]*m_outbox.Data, int64, error) {
	// Build query with filters - select all columns needed by m_outbox.Data
	query := "SELECT event_id, event_type, aggregate_id, payload, status, created_at, processed_at, retry_count, error_message FROM outbox_events WHERE 1=1"
	params := make(map[string]interface{})

	if req.EventType != nil {
		query += " AND event_type = @eventType"
		params["eventType"] = *req.EventType
	}

	if req.AggregateID != nil {
		query += " AND aggregate_id = @aggregateID"
		params["aggregateID"] = *req.AggregateID
	}

	if req.Status != nil {
		query += " AND status = @status"
		params["status"] = *req.Status
	}

	query += " ORDER BY created_at DESC LIMIT @limit"
	params["limit"] = req.Limit

	// Execute query
	stmt := spanner.Statement{
		SQL:    query,
		Params: params,
	}

	iter := r.client.Single().Query(ctx, stmt)
	defer iter.Stop()

	var events []*m_outbox.Data
	for {
		row, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, 0, fmt.Errorf("failed to iterate events: %w", err)
		}

		var event m_outbox.Data
		// Manually scan columns to handle field mapping
		if err := row.Columns(
			&event.EventID,
			&event.EventType,
			&event.AggregateID,
			&event.Payload,
			&event.Status,
			&event.CreatedAt,
			&event.ProcessedAt,
			&event.RetryCount,
			&event.ErrorMessage,
		); err != nil {
			return nil, 0, fmt.Errorf("failed to scan event: %w", err)
		}

		events = append(events, &event)
	}

	// Get total count (simplified - in production, use a separate count query)
	totalCount := int64(len(events))

	return events, totalCount, nil
}
