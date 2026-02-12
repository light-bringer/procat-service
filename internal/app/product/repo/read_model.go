package repo

import (
	"context"
	"fmt"
	"time"

	"cloud.google.com/go/spanner"
	"google.golang.org/api/iterator"

	"github.com/light-bringer/procat-service/internal/app/product/contracts"
	"github.com/light-bringer/procat-service/internal/app/product/domain"
	"github.com/light-bringer/procat-service/internal/models/m_product"
)

// ReadModelImpl implements ReadModel for Spanner.
type ReadModelImpl struct {
	client *spanner.Client
}

// NewReadModel creates a new ReadModel implementation.
func NewReadModel(client *spanner.Client) contracts.ReadModel {
	return &ReadModelImpl{
		client: client,
	}
}

// GetProductByID retrieves a product DTO by ID.
func (rm *ReadModelImpl) GetProductByID(ctx context.Context, productID string) (*contracts.ProductDTO, error) {
	row, err := rm.client.Single().ReadRow(ctx, m_product.TableName, spanner.Key{productID}, []string{
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
		m_product.CreatedAt,
		m_product.UpdatedAt,
	})
	if err != nil {
		if err == iterator.Done {
			return nil, domain.ErrProductNotFound
		}
		return nil, fmt.Errorf("failed to read product: %w", err)
	}

	var data m_product.Data
	if err := row.ToStruct(&data); err != nil {
		return nil, fmt.Errorf("failed to parse product: %w", err)
	}

	return rm.dataToDTO(&data, time.Now())
}

// ListProducts retrieves a paginated list of products with filtering.
func (rm *ReadModelImpl) ListProducts(ctx context.Context, filter *contracts.ListFilter) (*contracts.ListResult, error) {
	// Build query based on filter
	query := "SELECT " +
		"product_id, name, description, category, " +
		"base_price_numerator, base_price_denominator, " +
		"discount_percent, discount_start_date, discount_end_date, " +
		"status, created_at, updated_at " +
		"FROM products WHERE 1=1"

	params := make(map[string]interface{})

	if filter.Category != "" {
		query += " AND category = @category"
		params["category"] = filter.Category
	}

	if filter.Status != "" {
		query += " AND status = @status"
		params["status"] = filter.Status
	}

	query += " ORDER BY created_at DESC"

	// Apply pagination
	pageSize := filter.PageSize
	if pageSize <= 0 {
		pageSize = 50 // Default page size
	}
	if pageSize > 100 {
		pageSize = 100 // Max page size
	}

	query += " LIMIT @limit"
	params["limit"] = int64(pageSize)

	// Execute query
	stmt := spanner.Statement{
		SQL:    query,
		Params: params,
	}

	iter := rm.client.Single().Query(ctx, stmt)
	defer iter.Stop()

	now := time.Now()
	products := make([]*contracts.ProductDTO, 0, pageSize)

	for {
		row, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("failed to iterate products: %w", err)
		}

		var data m_product.Data
		if err := row.ToStruct(&data); err != nil {
			return nil, fmt.Errorf("failed to parse product: %w", err)
		}

		dto, err := rm.dataToDTO(&data, now)
		if err != nil {
			return nil, fmt.Errorf("failed to convert to DTO: %w", err)
		}

		products = append(products, dto)
	}

	// For simplicity, not implementing cursor-based pagination
	// In production, you'd use the last product's created_at as a cursor
	return &contracts.ListResult{
		Products:      products,
		NextPageToken: "",
		TotalCount:    int64(len(products)),
	}, nil
}

// dataToDTO converts database Data to a ProductDTO.
func (rm *ReadModelImpl) dataToDTO(data *m_product.Data, now time.Time) (*contracts.ProductDTO, error) {
	// Convert base price to float for display
	basePrice, err := domain.NewMoney(data.BasePriceNumerator, data.BasePriceDenominator)
	if err != nil {
		return nil, fmt.Errorf("invalid base price: %w", err)
	}

	dto := &contracts.ProductDTO{
		ProductID:      data.ProductID,
		Name:           data.Name,
		Description:    data.Description,
		Category:       data.Category,
		BasePrice:      basePrice.Float64(),
		EffectivePrice: basePrice.Float64(),
		Status:         data.Status,
		CreatedAt:      data.CreatedAt,
		UpdatedAt:      data.UpdatedAt,
	}

	// Handle discount
	if data.DiscountPercent.Valid {
		discount, err := domain.NewDiscount(
			data.DiscountPercent.Int64,
			data.DiscountStartDate.Time,
			data.DiscountEndDate.Time,
		)
		if err == nil && discount.IsValidAt(now) {
			dto.DiscountPercent = &data.DiscountPercent.Int64
			dto.DiscountActive = true
			effectivePrice := discount.Apply(basePrice)
			dto.EffectivePrice = effectivePrice.Float64()
		}
	}

	return dto, nil
}
