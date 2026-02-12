package update_price

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/google/uuid"

	"github.com/light-bringer/procat-service/internal/app/product/contracts"
	"github.com/light-bringer/procat-service/internal/app/product/domain"
	"github.com/light-bringer/procat-service/internal/pkg/clock"
	"github.com/light-bringer/procat-service/internal/pkg/committer"
)

// Request contains the data needed to update a product's price.
type Request struct {
	ProductID     string
	Version       int64 // For optimistic locking
	NewPrice      *domain.Money
	ChangedBy     string // User/system identifier
	ChangedReason string // Optional explanation for price change
}

// Interactor handles the update price use case.
type Interactor struct {
	repo             contracts.ProductRepository
	outboxRepo       contracts.OutboxRepository
	priceHistoryRepo contracts.PriceHistoryRepository
	committer        *committer.Committer
	clock            clock.Clock
}

// NewInteractor creates a new update price interactor.
func NewInteractor(
	repo contracts.ProductRepository,
	outboxRepo contracts.OutboxRepository,
	priceHistoryRepo contracts.PriceHistoryRepository,
	committer *committer.Committer,
	clock clock.Clock,
) *Interactor {
	return &Interactor{
		repo:             repo,
		outboxRepo:       outboxRepo,
		priceHistoryRepo: priceHistoryRepo,
		committer:        committer,
		clock:            clock,
	}
}

// Execute updates a product's price following the Golden Mutation Pattern.
func (i *Interactor) Execute(ctx context.Context, req *Request) error {
	// 1. Validate request
	if err := i.validate(req); err != nil {
		return err
	}

	// 2. Load aggregate
	product, err := i.repo.GetByID(ctx, req.ProductID)
	if err != nil {
		return err
	}

	// Note: ClearEvents() is called after successful commit, not in defer
	// This prevents event loss if the commit fails and the operation is retried

	// 3. Call domain method
	oldPrice := product.BasePrice() // Capture old price before change
	if err := product.SetBasePrice(req.NewPrice); err != nil {
		return err
	}

	// 4. Create commit plan
	plan := committer.NewPlan()

	// 5. Add repository mutation (only if changes exist)
	mut, err := i.repo.UpdateMut(product)
	if err != nil {
		return fmt.Errorf("failed to create update mutation: %w", err)
	}
	if mut != nil {
		plan.Add(mut)
	}

	// 6. Add price history record
	historyID := uuid.New().String()
	now := i.clock.Now()
	historyMut, err := i.priceHistoryRepo.InsertMut(
		historyID,
		req.ProductID,
		oldPrice,
		req.NewPrice,
		req.ChangedBy,
		req.ChangedReason,
		now,
	)
	if err != nil {
		return fmt.Errorf("failed to create price history mutation: %w", err)
	}
	plan.Add(historyMut)

	// 7. Add outbox events
	for _, event := range product.DomainEvents() {
		payload, err := i.serializeEvent(event)
		if err != nil {
			return fmt.Errorf("failed to serialize event: %w", err)
		}
		outboxEvent := i.outboxRepo.EnrichEvent(event, payload)
		plan.Add(i.outboxRepo.InsertMut(outboxEvent))
	}

	// 8. Apply plan
	if plan.IsEmpty() {
		return nil // No changes
	}

	// 6. Execute transaction with optimistic locking
	// Always enforce optimistic locking for UpdatePrice
	err = i.committer.ApplyWithVersionCheck(ctx, req.ProductID, req.Version, plan)
	if err != nil {
		return fmt.Errorf("failed to update price: %w", err)
	}

	// Clear events only after successful commit to prevent loss on retry
	product.ClearEvents()

	return nil
}

// validate validates the request.
func (i *Interactor) validate(req *Request) error {
	if req.ProductID == "" {
		return fmt.Errorf("product ID is required")
	}
	if req.NewPrice == nil || req.NewPrice.IsNegative() || req.NewPrice.IsZero() {
		return domain.ErrInvalidPrice
	}
	if req.ChangedBy == "" {
		return fmt.Errorf("changedBy is required")
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
