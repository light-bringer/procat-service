package m_outbox

import (
	"cloud.google.com/go/spanner"
)

// Model provides a facade for type-safe operations on the outbox_events table.
type Model struct{}

// NewModel creates a new Model instance.
func NewModel() *Model {
	return &Model{}
}

// InsertMut creates a Spanner mutation for inserting an outbox event.
func (m *Model) InsertMut(data *Data) *spanner.Mutation {
	return spanner.Insert(
		TableName,
		[]string{
			EventID,
			EventType,
			AggregateID,
			Payload,
			Status,
			CreatedAt,
			ProcessedAt,
			RetryCount,
			ErrorMessage,
		},
		[]interface{}{
			data.EventID,
			data.EventType,
			data.AggregateID,
			data.Payload,
			data.Status,
			spanner.CommitTimestamp,
			data.ProcessedAt,
			data.RetryCount,
			data.ErrorMessage,
		},
	)
}

// UpdateMut creates a Spanner mutation for updating an outbox event.
func (m *Model) UpdateMut(eventID string, updates map[string]interface{}) *spanner.Mutation {
	if len(updates) == 0 {
		return nil
	}

	columns := make([]string, 0, len(updates)+1)
	values := make([]interface{}, 0, len(updates)+1)

	// Add event ID first
	columns = append(columns, EventID)
	values = append(values, eventID)

	// Add all update fields
	for col, val := range updates {
		columns = append(columns, col)
		values = append(values, val)
	}

	return spanner.Update(TableName, columns, values)
}

// DeleteMut creates a Spanner mutation for deleting an outbox event.
func (m *Model) DeleteMut(eventID string) *spanner.Mutation {
	return spanner.Delete(TableName, spanner.Key{eventID})
}
