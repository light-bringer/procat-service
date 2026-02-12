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
