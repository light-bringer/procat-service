package apply_discount

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

// Request contains the data to apply a discount.
type Request struct {
	ProductID       string
	DiscountPercent int64
	StartDate       time.Time
	EndDate         time.Time
}

// Interactor handles the apply discount use case.
type Interactor struct {
	repo       contracts.ProductRepository
	outboxRepo contracts.OutboxRepository
	committer  *committer.Committer
	clock      clock.Clock
}

// NewInteractor creates a new apply discount interactor.
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

// Execute applies a discount to a product following the Golden Mutation Pattern.
func (i *Interactor) Execute(ctx context.Context, req *Request) error {
	// 1. Load aggregate
	product, err := i.repo.GetByID(ctx, req.ProductID)
	if err != nil {
		return err
	}

	// 2. Create discount value object
	discount, err := domain.NewDiscount(req.DiscountPercent, req.StartDate, req.EndDate)
	if err != nil {
		return err
	}

	// 3. Call domain method
	now := i.clock.Now()
	if err := product.ApplyDiscount(discount, now); err != nil {
		return err
	}

	// 4. Create commit plan
	plan := committer.NewPlan()

	// 5. Add repository mutation
	plan.Add(i.repo.UpdateMut(product))

	// 6. Add outbox events
	for _, event := range product.DomainEvents() {
		payload, err := i.serializeEvent(event)
		if err != nil {
			return fmt.Errorf("failed to serialize event: %w", err)
		}
		outboxEvent := i.outboxRepo.EnrichEvent(event, payload)
		plan.Add(i.outboxRepo.InsertMut(outboxEvent))
	}

	// 7. Apply plan
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
