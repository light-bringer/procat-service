package create_product

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

// Request contains the data needed to create a product.
type Request struct {
	Name        string
	Description string
	Category    string
	BasePrice   *domain.Money
}

// Interactor handles the create product use case.
type Interactor struct {
	repo       contracts.ProductRepository
	outboxRepo contracts.OutboxRepository
	committer  *committer.Committer
	clock      clock.Clock
}

// NewInteractor creates a new create product interactor.
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

// Execute creates a new product following the Golden Mutation Pattern.
func (i *Interactor) Execute(ctx context.Context, req *Request) (string, error) {
	// 1. Validate request
	if err := i.validate(req); err != nil {
		return "", err
	}

	// 2. Create domain aggregate (new product)
	productID := uuid.New().String()
	now := i.clock.Now()

	product, err := domain.NewProduct(
		productID,
		req.Name,
		req.Description,
		req.Category,
		req.BasePrice,
		now,
	)
	if err != nil {
		return "", fmt.Errorf("failed to create product: %w", err)
	}

	// 3. Create commit plan
	plan := committer.NewPlan()

	// 4. Add repository mutation
	plan.Add(i.repo.InsertMut(product))

	// 5. Add outbox events
	for _, event := range product.DomainEvents() {
		payload, err := i.serializeEvent(event)
		if err != nil {
			return "", fmt.Errorf("failed to serialize event: %w", err)
		}
		outboxEvent := i.outboxRepo.EnrichEvent(event, payload)
		plan.Add(i.outboxRepo.InsertMut(outboxEvent))
	}

	// 6. Apply plan (usecase applies, not handler)
	if err := i.committer.Apply(ctx, plan); err != nil {
		return "", fmt.Errorf("failed to commit transaction: %w", err)
	}

	return product.ID(), nil
}

// validate validates the request.
func (i *Interactor) validate(req *Request) error {
	if req.Name == "" {
		return domain.ErrEmptyName
	}
	if req.Category == "" {
		return domain.ErrInvalidCategory
	}
	if req.BasePrice == nil || req.BasePrice.IsNegative() || req.BasePrice.IsZero() {
		return domain.ErrInvalidPrice
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
