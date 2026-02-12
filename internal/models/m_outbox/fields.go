package m_outbox

// Field name constants for the outbox_events table.
const (
	TableName = "outbox_events"

	EventID      = "event_id"
	EventType    = "event_type"
	AggregateID  = "aggregate_id"
	Payload      = "payload"
	Status       = "status"
	CreatedAt    = "created_at"
	ProcessedAt  = "processed_at"
	RetryCount   = "retry_count"
	ErrorMessage = "error_message"
)

// Event status constants
const (
	StatusPending    = "pending"
	StatusProcessing = "processing"
	StatusCompleted  = "completed"
	StatusFailed     = "failed"
)
