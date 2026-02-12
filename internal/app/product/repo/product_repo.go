package repo

import (
	"context"
	"fmt"
	"time"

	"cloud.google.com/go/spanner"
	"google.golang.org/grpc/codes"

	"github.com/light-bringer/procat-service/internal/app/product/contracts"
	"github.com/light-bringer/procat-service/internal/app/product/domain"
	"github.com/light-bringer/procat-service/internal/models/m_product"
	"github.com/light-bringer/procat-service/internal/pkg/clock"
)

// ProductRepo implements ProductRepository for Spanner.
type ProductRepo struct {
	client *spanner.Client
	model  *m_product.Model
	clock  clock.Clock
}

// NewProductRepo creates a new ProductRepo.
func NewProductRepo(client *spanner.Client, clk clock.Clock) contracts.ProductRepository {
	return &ProductRepo{
		client: client,
		model:  m_product.NewModel(),
		clock:  clk,
	}
}

// InsertMut creates a mutation for inserting a new product.
func (r *ProductRepo) InsertMut(product *domain.Product) (*spanner.Mutation, error) {
	data, err := r.domainToData(product)
	if err != nil {
		return nil, err
	}
	return r.model.InsertMut(data), nil
}

// UpdateMut creates a mutation for updating a product (only dirty fields).
func (r *ProductRepo) UpdateMut(product *domain.Product) (*spanner.Mutation, error) {
	changes := product.Changes()
	if !changes.HasChanges() {
		return nil, nil
	}

	updates := make(map[string]interface{})

	if changes.Dirty(domain.FieldName) {
		updates[m_product.Name] = product.Name()
	}

	if changes.Dirty(domain.FieldDescription) {
		updates[m_product.Description] = product.Description()
	}

	if changes.Dirty(domain.FieldCategory) {
		updates[m_product.Category] = product.Category()
	}

	if changes.Dirty(domain.FieldBasePrice) {
		basePrice := product.BasePrice().Normalize()
		if !basePrice.IsSafeForStorage() {
			return nil, fmt.Errorf("base price exceeds storage capacity: %w", domain.ErrMoneyOverflow)
		}
		num, _ := basePrice.Numerator()
		denom, _ := basePrice.Denominator()
		updates[m_product.BasePriceNumerator] = num
		updates[m_product.BasePriceDenominator] = denom
	}

	if changes.Dirty(domain.FieldDiscount) {
		discount := product.DiscountCopy() // Use DiscountCopy() instead of deprecated Discount()
		if discount != nil {
			percentRat := discount.PercentageRat() // Get *big.Rat directly for precision
			updates[m_product.DiscountPercent] = spanner.NullNumeric{Numeric: *percentRat, Valid: true}
			updates[m_product.DiscountStartDate] = discount.StartDate()
			updates[m_product.DiscountEndDate] = discount.EndDate()
		} else {
			updates[m_product.DiscountPercent] = spanner.NullNumeric{}
			updates[m_product.DiscountStartDate] = spanner.NullTime{}
			updates[m_product.DiscountEndDate] = spanner.NullTime{}
		}
	}

	if changes.Dirty(domain.FieldStatus) {
		updates[m_product.Status] = string(product.Status())
	}

	if changes.Dirty(domain.FieldArchivedAt) {
		if archivedAt := product.ArchivedAt(); archivedAt != nil {
			updates[m_product.ArchivedAt] = *archivedAt
		} else {
			updates[m_product.ArchivedAt] = spanner.NullTime{}
		}
	}

	if len(updates) == 0 {
		return nil, nil
	}

	// Always update the updated_at timestamp when any field changes
	updates[m_product.UpdatedAt] = r.clock.Now()

	// Increment version for optimistic locking
	updates[m_product.Version] = product.Version() + 1

	return r.model.UpdateMut(product.ID(), updates), nil
}

// GetByID retrieves a product by ID, reconstructing the domain aggregate.
func (r *ProductRepo) GetByID(ctx context.Context, productID string) (*domain.Product, error) {
	row, err := r.client.Single().ReadRow(ctx, m_product.TableName, spanner.Key{productID}, []string{
		m_product.ProductID,
		m_product.Name,
		m_product.Description,
		m_product.Category,
		m_product.BasePriceNumerator,
		m_product.BasePriceDenominator,
		m_product.DiscountPercent,
		m_product.DiscountStartDate,
		m_product.DiscountEndDate,
		m_product.Status,
		m_product.Version,
		m_product.CreatedAt,
		m_product.UpdatedAt,
		m_product.ArchivedAt,
	})
	if err != nil {
		if spanner.ErrCode(err) == codes.NotFound {
			return nil, domain.ErrProductNotFound
		}
		return nil, fmt.Errorf("failed to read product: %w", err)
	}

	var data m_product.Data
	if err := row.ToStruct(&data); err != nil {
		return nil, fmt.Errorf("failed to parse product: %w", err)
	}

	return r.dataToDomain(&data)
}

// Exists checks if a product exists.
func (r *ProductRepo) Exists(ctx context.Context, productID string) (bool, error) {
	row, err := r.client.Single().ReadRow(ctx, m_product.TableName, spanner.Key{productID}, []string{m_product.ProductID})
	if err != nil {
		if spanner.ErrCode(err) == codes.NotFound {
			return false, nil
		}
		return false, fmt.Errorf("failed to check product existence: %w", err)
	}
	return row != nil, nil
}

// domainToData converts a domain Product to database Data.
func (r *ProductRepo) domainToData(product *domain.Product) (*m_product.Data, error) {
	// Normalize price to ensure consistent storage (200/2 → 100/1)
	normalizedPrice := product.BasePrice().Normalize()

	// Check if price values fit within int64 bounds before storing
	if !normalizedPrice.IsSafeForStorage() {
		return nil, fmt.Errorf("price exceeds storage capacity: %w", domain.ErrMoneyOverflow)
	}

	num, _ := normalizedPrice.Numerator()
	denom, _ := normalizedPrice.Denominator()

	data := &m_product.Data{
		ProductID:            product.ID(),
		Name:                 product.Name(),
		Description:          product.Description(),
		Category:             product.Category(),
		BasePriceNumerator:   num,
		BasePriceDenominator: denom,
		Status:               string(product.Status()),
		Version:              product.Version(),
		CreatedAt:            product.CreatedAt(),
		UpdatedAt:            product.UpdatedAt(),
	}

	// Handle discount (nullable)
	if discount := product.DiscountCopy(); discount != nil { // Use DiscountCopy() instead of deprecated Discount()
		percentRat := discount.PercentageRat() // Get *big.Rat directly for precision
		data.DiscountPercent = spanner.NullNumeric{Numeric: *percentRat, Valid: true}
		data.DiscountStartDate = spanner.NullTime{Time: discount.StartDate(), Valid: true}
		data.DiscountEndDate = spanner.NullTime{Time: discount.EndDate(), Valid: true}
	}

	// Handle archived_at (nullable)
	if archivedAt := product.ArchivedAt(); archivedAt != nil {
		data.ArchivedAt = spanner.NullTime{Time: *archivedAt, Valid: true}
	}

	return data, nil
}

// dataToDomain converts database Data to a domain Product.
func (r *ProductRepo) dataToDomain(data *m_product.Data) (*domain.Product, error) {
	basePrice, err := domain.NewMoney(data.BasePriceNumerator, data.BasePriceDenominator)
	if err != nil {
		return nil, fmt.Errorf("invalid base price: %w", err)
	}

	var discount *domain.Discount
	if data.DiscountPercent.Valid {
		percent, _ := data.DiscountPercent.Numeric.Float64()
		discount, err = domain.NewDiscount(
			percent,
			data.DiscountStartDate.Time,
			data.DiscountEndDate.Time,
		)
		if err != nil {
			return nil, fmt.Errorf("invalid discount: %w", err)
		}
	}

	var archivedAt *time.Time
	if data.ArchivedAt.Valid {
		archivedAt = &data.ArchivedAt.Time
	}

	// Use injected clock for reconstructed products
	return domain.ReconstructProduct(
		data.ProductID,
		data.Name,
		data.Description,
		data.Category,
		basePrice,
		discount,
		domain.ProductStatus(data.Status),
		data.Version,
		data.CreatedAt,
		data.UpdatedAt,
		archivedAt,
		r.clock,
	), nil
}
