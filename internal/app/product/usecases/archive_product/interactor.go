package archive_product

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/light-bringer/procat-service/internal/app/product/contracts"
	"github.com/light-bringer/procat-service/internal/app/product/domain"
	"github.com/light-bringer/procat-service/internal/pkg/clock"
	"github.com/light-bringer/procat-service/internal/pkg/committer"
)

// Request contains the product ID to archive.
type Request struct {
	ProductID string
	Version   int64 // For optimistic locking
}

// Interactor handles the archive product use case.
type Interactor struct {
	repo       contracts.ProductRepository
	outboxRepo contracts.OutboxRepository
	committer  *committer.Committer
	clock      clock.Clock
}

// NewInteractor creates a new archive product interactor.
func NewInteractor(
	repo contracts.ProductRepository,
	outboxRepo contracts.OutboxRepository,
	committer *committer.Committer,
	clock clock.Clock,
) *Interactor {
	return &Interactor{
		repo:       repo,
		outboxRepo: outboxRepo,
		committer:  committer,
		clock:      clock,
	}
}

// Execute archives a product (soft delete) following the Golden Mutation Pattern.
// Returns the timestamp when the product was archived.
func (i *Interactor) Execute(ctx context.Context, req *Request) (time.Time, error) {
	// 1. Load aggregate
	product, err := i.repo.GetByID(ctx, req.ProductID)
	if err != nil {
		return time.Time{}, err
	}

	// Note: ClearEvents() is called after successful commit, not in defer
	// This prevents event loss if the commit fails and the operation is retried

	// 2. Call domain method
	now := i.clock.Now()
	if err := product.Archive(now); err != nil {
		return time.Time{}, err
	}

	// 3. Create commit plan
	plan := committer.NewPlan()

	// 4. Add repository mutation
	mut, err := i.repo.UpdateMut(product)
	if err != nil {
		return time.Time{}, fmt.Errorf("failed to create update mutation: %w", err)
	}
	if mut != nil {
		plan.Add(mut)
	}

	// 5. Add outbox events
	for _, event := range product.DomainEvents() {
		payload, err := i.serializeEvent(event)
		if err != nil {
			return time.Time{}, fmt.Errorf("failed to serialize event: %w", err)
		}
		outboxEvent := i.outboxRepo.EnrichEvent(event, payload)
		plan.Add(i.outboxRepo.InsertMut(outboxEvent))
	}

	// 6. Apply plan with optimistic locking
	// Always enforce version checking to prevent concurrent modification issues
	if err := i.committer.ApplyWithVersionCheck(ctx, req.ProductID, req.Version, plan); err != nil {
		return time.Time{}, fmt.Errorf("failed to commit transaction: %w", err)
	}

	// Clear events only after successful commit to prevent loss on retry
	product.ClearEvents()

	return now, nil
}

// serializeEvent converts a domain event to JSON payload.
func (i *Interactor) serializeEvent(event domain.DomainEvent) (string, error) {
	data, err := json.Marshal(event)
	if err != nil {
		return "", err
	}
	return string(data), nil
}
