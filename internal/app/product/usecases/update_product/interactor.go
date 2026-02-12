package update_product

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/light-bringer/procat-service/internal/app/product/contracts"
	"github.com/light-bringer/procat-service/internal/app/product/domain"
	"github.com/light-bringer/procat-service/internal/pkg/clock"
	"github.com/light-bringer/procat-service/internal/pkg/committer"
)

// Request contains the data to update a product.
type Request struct {
	ProductID   string
	Name        *string // nil = no change
	Description *string // nil = no change
	Category    *string // nil = no change
}

// Interactor handles the update product use case.
type Interactor struct {
	repo       contracts.ProductRepository
	outboxRepo contracts.OutboxRepository
	committer  *committer.Committer
	clock      clock.Clock
}

// NewInteractor creates a new update product interactor.
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

// Execute updates a product following the Golden Mutation Pattern.
func (i *Interactor) Execute(ctx context.Context, req *Request) error {
	// 1. Load aggregate
	product, err := i.repo.GetByID(ctx, req.ProductID)
	if err != nil {
		return err
	}

	// Clear events on function exit to prevent duplicates on retry
	defer product.ClearEvents()

	// 2. Call domain methods
	hasChanges := false

	if req.Name != nil {
		if err := product.SetName(*req.Name); err != nil {
			return err
		}
		hasChanges = true
	}

	if req.Description != nil {
		if err := product.SetDescription(*req.Description); err != nil {
			return err
		}
		hasChanges = true
	}

	if req.Category != nil {
		if err := product.SetCategory(*req.Category); err != nil {
			return err
		}
		hasChanges = true
	}

	// Emit a single ProductUpdatedEvent for all changes
	if hasChanges {
		product.MarkUpdated(i.clock.Now())
	}

	// 3. Create commit plan
	plan := committer.NewPlan()

	// 4. Add repository mutation (only if changes exist)
	if mut := i.repo.UpdateMut(product); mut != nil {
		plan.Add(mut)
	}

	// 5. Add outbox events (standard pattern - same as activate, apply_discount, etc.)
	for _, event := range product.DomainEvents() {
		payload, err := i.serializeEvent(event)
		if err != nil {
			return fmt.Errorf("failed to serialize event: %w", err)
		}
		outboxEvent := i.outboxRepo.EnrichEvent(event, payload)
		plan.Add(i.outboxRepo.InsertMut(outboxEvent))
	}

	// 6. Apply plan
	if plan.IsEmpty() {
		return nil // No changes
	}

	if err := i.committer.Apply(ctx, plan); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	return nil
}

// serializeEvent converts a domain event to JSON payload.
func (i *Interactor) serializeEvent(event domain.DomainEvent) (string, error) {
	data, err := json.Marshal(event)
	if err != nil {
		return "", err
	}
	return string(data), nil
}
