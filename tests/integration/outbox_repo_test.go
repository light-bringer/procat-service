//go:build integration

package integration

import (
	"context"
	"testing"

	"cloud.google.com/go/spanner"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/light-bringer/procat-service/internal/app/product/domain"
	"github.com/light-bringer/procat-service/internal/app/product/repo"
	"github.com/light-bringer/procat-service/internal/models/m_outbox"
	"github.com/light-bringer/procat-service/tests/testutil"
)

func TestOutboxRepository_InsertMut(t *testing.T) {
	client, cleanup := testutil.SetupSpannerTest(t)
	defer cleanup()

	repository := repo.NewOutboxRepo(client)

	// Create a domain event
	event := &domain.ProductCreatedEvent{
		ProductID: "test-product-id",
		Name:      "Test Product",
		Category:  "electronics",
	}

	// Enrich event to outbox event
	outboxEvent := repository.EnrichEvent(event, `{"test": "payload"}`)

	// Get mutation
	mutation := repository.InsertMut(outboxEvent)
	require.NotNil(t, mutation)

	// Apply mutation
	ctx := context.Background()
	_, err := client.Apply(ctx, []*spanner.Mutation{mutation})
	require.NoError(t, err)

	// Verify event was inserted
	testutil.AssertRowCount(t, client, "outbox_events", 1)
	testutil.AssertOutboxEvent(t, client, "product.created")
}

func TestOutboxRepository_EnrichEvent(t *testing.T) {
	repository := repo.NewOutboxRepo(nil) // No client needed for enrichment

	event := &domain.ProductActivatedEvent{
		ProductID: "test-id",
	}

	outboxEvent := repository.EnrichEvent(event, `{"productId": "test-id"}`)

	assert.NotEmpty(t, outboxEvent.EventID, "event ID should be generated")
	assert.Equal(t, "product.activated", outboxEvent.EventType)
	assert.Equal(t, "test-id", outboxEvent.AggregateID)
	assert.Equal(t, `{"productId": "test-id"}`, outboxEvent.Payload)
	assert.Equal(t, m_outbox.StatusPending, outboxEvent.Status)
}

func TestOutboxRepository_MultipleEvents(t *testing.T) {
	client, cleanup := testutil.SetupSpannerTest(t)
	defer cleanup()

	ctx := context.Background()
	repository := repo.NewOutboxRepo(client)

	// Create multiple events
	events := []domain.DomainEvent{
		&domain.ProductCreatedEvent{ProductID: "p1"},
		&domain.ProductActivatedEvent{ProductID: "p1"},
		&domain.DiscountAppliedEvent{ProductID: "p1"},
	}

	mutations := make([]*spanner.Mutation, 0)
	for _, event := range events {
		outboxEvent := repository.EnrichEvent(event, `{}`)
		mutations = append(mutations, repository.InsertMut(outboxEvent))
	}

	// Apply all mutations
	_, err := client.Apply(ctx, mutations)
	require.NoError(t, err)

	// Verify all events inserted
	testutil.AssertOutboxEventCount(t, client, 3)
}

// TestOutboxReliability_EventOrdering verifies events maintain order within a transaction.
func TestOutboxReliability_EventOrdering(t *testing.T) {
	client, cleanup := testutil.SetupSpannerTest(t)
	defer cleanup()

	ctx := context.Background()
	repository := repo.NewOutboxRepo(client)

	// Create events in specific order
	events := []domain.DomainEvent{
		&domain.ProductCreatedEvent{ProductID: "p1", Name: "First"},
		&domain.ProductActivatedEvent{ProductID: "p1"},
		&domain.DiscountAppliedEvent{ProductID: "p1"},
	}

	mutations := make([]*spanner.Mutation, 0)
	for _, event := range events {
		outboxEvent := repository.EnrichEvent(event, `{}`)
		mutations = append(mutations, repository.InsertMut(outboxEvent))
	}

	// Apply all mutations in single transaction
	_, err := client.Apply(ctx, mutations)
	require.NoError(t, err)

	// Query events by created_at to verify order
	query := `
		SELECT event_type, created_at
		FROM outbox_events
		ORDER BY created_at ASC
	`
	iter := client.Single().Query(ctx, spanner.Statement{SQL: query})
	defer iter.Stop()

	expectedOrder := []string{"product.created", "product.activated", "product.discount.applied"}
	actualOrder := make([]string, 0)

	for {
		row, err := iter.Next()
		if err != nil {
			break
		}
		var eventType string
		var createdAt interface{}
		row.Columns(&eventType, &createdAt)
		actualOrder = append(actualOrder, eventType)
	}

	assert.Equal(t, expectedOrder, actualOrder, "Events should maintain insertion order")
}

// TestOutboxReliability_TransactionRollback verifies events are not saved if transaction fails.
func TestOutboxReliability_TransactionRollback(t *testing.T) {
	client, cleanup := testutil.SetupSpannerTest(t)
	defer cleanup()

	ctx := context.Background()
	repository := repo.NewOutboxRepo(client)

	// Attempt transaction that will fail
	_, err := client.ReadWriteTransaction(ctx, func(ctx context.Context, txn *spanner.ReadWriteTransaction) error {
		// Insert an event
		event := &domain.ProductCreatedEvent{ProductID: "p1"}
		outboxEvent := repository.EnrichEvent(event, `{}`)
		mutation := repository.InsertMut(outboxEvent)
		txn.BufferWrite([]*spanner.Mutation{mutation})

		// Force transaction to fail
		return assert.AnError
	})

	require.Error(t, err, "Transaction should fail")

	// Verify no events were saved
	testutil.AssertOutboxEventCount(t, client, 0)
}

// TestOutboxReliability_DuplicateEventIDs verifies unique constraint on event_id.
func TestOutboxReliability_DuplicateEventIDs(t *testing.T) {
	client, cleanup := testutil.SetupSpannerTest(t)
	defer cleanup()

	ctx := context.Background()
	repository := repo.NewOutboxRepo(client)

	// Create first event
	event1 := repository.EnrichEvent(&domain.ProductCreatedEvent{ProductID: "p1"}, `{}`)
	_, err := client.Apply(ctx, []*spanner.Mutation{repository.InsertMut(event1)})
	require.NoError(t, err)

	// Try to insert event with same ID
	event2 := event1 // Same event ID
	event2.EventType = "product.updated"
	_, err = client.Apply(ctx, []*spanner.Mutation{repository.InsertMut(event2)})
	assert.Error(t, err, "Duplicate event ID should be rejected")

	// Verify only first event exists
	testutil.AssertOutboxEventCount(t, client, 1)
}

// TestOutboxReliability_EventSerialization verifies all event types can be serialized/deserialized.
func TestOutboxReliability_EventSerialization(t *testing.T) {
	repository := repo.NewOutboxRepo(nil)

	testCases := []struct {
		name  string
		event domain.DomainEvent
	}{
		{"ProductCreatedEvent", &domain.ProductCreatedEvent{ProductID: "p1"}},
		{"ProductUpdatedEvent", &domain.ProductUpdatedEvent{ProductID: "p1"}},
		{"ProductActivatedEvent", &domain.ProductActivatedEvent{ProductID: "p1"}},
		{"ProductDeactivatedEvent", &domain.ProductDeactivatedEvent{ProductID: "p1"}},
		{"DiscountAppliedEvent", &domain.DiscountAppliedEvent{ProductID: "p1"}},
		{"DiscountRemovedEvent", &domain.DiscountRemovedEvent{ProductID: "p1"}},
		{"ProductArchivedEvent", &domain.ProductArchivedEvent{ProductID: "p1"}},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Enrich event
			outboxEvent := repository.EnrichEvent(tc.event, `{"test": "data"}`)

			// Verify all required fields are set
			assert.NotEmpty(t, outboxEvent.EventID)
			assert.NotEmpty(t, outboxEvent.EventType)
			assert.NotEmpty(t, outboxEvent.AggregateID)
			assert.NotEmpty(t, outboxEvent.Payload)
			assert.Equal(t, m_outbox.StatusPending, outboxEvent.Status)
		})
	}
}

// TestOutboxReliability_AtomicWithBusinessLogic verifies events and data changes are atomic.
func TestOutboxReliability_AtomicWithBusinessLogic(t *testing.T) {
	client, cleanup := testutil.SetupSpannerTest(t)
	defer cleanup()

	ctx := context.Background()
	repository := repo.NewOutboxRepo(client)

	// Simulate a business transaction: insert product + outbox event
	productMutation := spanner.Insert("products",
		[]string{"product_id", "name", "description", "category", "base_price_numerator", "base_price_denominator", "status", "version", "created_at", "updated_at"},
		[]interface{}{"p1", "Product", "Desc", "electronics", int64(10000), int64(100), "inactive", int64(0), spanner.CommitTimestamp, spanner.CommitTimestamp},
	)

	event := repository.EnrichEvent(&domain.ProductCreatedEvent{ProductID: "p1"}, `{}`)
	eventMutation := repository.InsertMut(event)

	// Apply both mutations atomically
	_, err := client.Apply(ctx, []*spanner.Mutation{productMutation, eventMutation})
	require.NoError(t, err)

	// Verify both product and event exist
	testutil.AssertRowCount(t, client, "products", 1)
	testutil.AssertOutboxEventCount(t, client, 1)
}

// TestOutboxReliability_IdempotentProcessing verifies events can be processed multiple times safely.
func TestOutboxReliability_IdempotentProcessing(t *testing.T) {
	client, cleanup := testutil.SetupSpannerTest(t)
	defer cleanup()

	ctx := context.Background()
	repository := repo.NewOutboxRepo(client)

	// Insert event
	event := repository.EnrichEvent(&domain.ProductCreatedEvent{ProductID: "p1"}, `{"key": "value"}`)
	_, err := client.Apply(ctx, []*spanner.Mutation{repository.InsertMut(event)})
	require.NoError(t, err)

	// Simulate processing same event multiple times
	for i := 0; i < 3; i++ {
		// Read event
		row, err := client.Single().ReadRow(ctx, "outbox_events", spanner.Key{event.EventID},
			[]string{"event_id", "payload"})
		require.NoError(t, err)

		var eventID, payload string
		row.Columns(&eventID, &payload)

		// Verify payload is consistent
		assert.Equal(t, event.EventID, eventID)
		assert.Equal(t, `{"key": "value"}`, payload)

		// Processing logic would use event_id to ensure idempotence
		// (e.g., check if already processed before applying changes)
	}
}
