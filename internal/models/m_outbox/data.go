package m_outbox

import (
	"time"

	"cloud.google.com/go/spanner"
)

// Data represents the database model for the outbox_events table.
type Data struct {
	EventID      string
	EventType    string
	AggregateID  string
	Payload      spanner.NullJSON // JSON column
	Status       string
	CreatedAt    time.Time
	ProcessedAt  spanner.NullTime
	RetryCount   int64
	ErrorMessage spanner.NullString
}
