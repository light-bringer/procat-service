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
	repo             contracts.ProductRepository
	outboxRepo       contracts.OutboxRepository
	priceHistoryRepo contracts.PriceHistoryRepository
	committer        *committer.Committer
	clock            clock.Clock
}

// NewInteractor creates a new create product interactor.
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
		i.clock,
	)
	if err != nil {
		return "", fmt.Errorf("failed to create product: %w", err)
	}

	// Note: ClearEvents() is called after successful commit, not in defer
	// This prevents event loss if the commit fails and the operation is retried

	// 3. Create commit plan
	plan := committer.NewPlan()

	// 4. Add repository mutation
	mut, err := i.repo.InsertMut(product)
	if err != nil {
		return "", fmt.Errorf("failed to create product mutation: %w", err)
	}
	plan.Add(mut)

	// 5. Add initial price history record (old price = nil for creation)
	historyID := uuid.New().String()
	historyMut, err := i.priceHistoryRepo.InsertMut(
		historyID,
		productID,
		nil, // oldPrice is nil for initial creation
		req.BasePrice,
		"system",        // changedBy - system for initial creation
		"Initial price", // changedReason
		now,
	)
	if err != nil {
		return "", fmt.Errorf("failed to create price history mutation: %w", err)
	}
	plan.Add(historyMut)

	// 6. Add outbox events
	for _, event := range product.DomainEvents() {
		payload, err := i.serializeEvent(event)
		if err != nil {
			return "", fmt.Errorf("failed to serialize event: %w", err)
		}
		outboxEvent := i.outboxRepo.EnrichEvent(event, payload)
		plan.Add(i.outboxRepo.InsertMut(outboxEvent))
	}

	// 7. Apply plan (usecase applies, not handler)
	if err := i.committer.Apply(ctx, plan); err != nil {
		return "", fmt.Errorf("failed to commit transaction: %w", err)
	}

	// Clear events only after successful commit to prevent loss on retry
	product.ClearEvents()

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
