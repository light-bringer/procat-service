package repo

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"cloud.google.com/go/spanner"
	"github.com/light-bringer/procat-service/internal/app/product/contracts"
	"github.com/light-bringer/procat-service/internal/app/product/domain"
	"github.com/light-bringer/procat-service/internal/models/m_product"
	"github.com/light-bringer/procat-service/internal/pkg/clock"
	"github.com/light-bringer/procat-service/internal/pkg/query"
	"google.golang.org/api/iterator"
	"google.golang.org/grpc/codes"
)

// ReadModelImpl implements ReadModel for Spanner.
type ReadModelImpl struct {
	client *spanner.Client
	clock  clock.Clock
}

// NewReadModel creates a new ReadModel implementation.
func NewReadModel(client *spanner.Client, clk clock.Clock) contracts.ReadModel {
	return &ReadModelImpl{
		client: client,
		clock:  clk,
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

	return rm.dataToDTO(&data, rm.clock.Now())
}

// ListProducts retrieves a paginated list of products with filtering.
func (rm *ReadModelImpl) ListProducts(ctx context.Context, filter *contracts.ListFilter) (*contracts.ListResult, error) {
	offset, err := parsePageToken(filter.PageToken)
	if err != nil {
		return nil, err
	}

	// Build query using query builder
	builder := query.From(m_product.TableName).
		Select(
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
		).
		OrderBy(m_product.CreatedAt, query.Desc)

	// Add filters
	if filter.Category != "" {
		builder = builder.Where(query.Eq(m_product.Category, filter.Category))
	}

	if filter.Status != "" {
		builder = builder.Where(query.Eq(m_product.Status, filter.Status))
	}

	// Apply pagination
	pageSize := filter.PageSize
	if pageSize <= 0 {
		pageSize = 50 // Default page size
	}
	if pageSize > 100 {
		pageSize = 100 // Max page size
	}

	builder = builder.Limit(int64(pageSize + 1)).Offset(int64(offset)) // Fetch one extra row to compute next page token.

	// Execute query
	stmt := builder.Build()

	iter := rm.client.Single().Query(ctx, stmt)
	defer iter.Stop()

	now := rm.clock.Now()
	products := make([]*contracts.ProductDTO, 0, pageSize+1)

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

	nextPageToken := ""
	if len(products) > pageSize {
		products = products[:pageSize]
		nextPageToken = strconv.Itoa(offset + pageSize)
	}

	totalCount, err := rm.countProducts(ctx, builder)
	if err != nil {
		return nil, err
	}

	return &contracts.ListResult{
		Products:      products,
		NextPageToken: nextPageToken,
		TotalCount:    totalCount,
	}, nil
}

func parsePageToken(token string) (int, error) {
	if token == "" {
		return 0, nil
	}

	offset, err := strconv.Atoi(token)
	if err != nil {
		return 0, fmt.Errorf("invalid page token: %w", err)
	}
	if offset < 0 {
		return 0, fmt.Errorf("invalid page token: offset cannot be negative")
	}
	return offset, nil
}

func (rm *ReadModelImpl) countProducts(ctx context.Context, builder *query.Builder) (int64, error) {
	stmt := builder.Count().Build()

	iter := rm.client.Single().Query(ctx, stmt)
	defer iter.Stop()

	row, err := iter.Next()
	if err != nil {
		return 0, fmt.Errorf("failed to count products: %w", err)
	}

	var total int64
	if err := row.Columns(&total); err != nil {
		return 0, fmt.Errorf("failed to parse product count: %w", err)
	}

	return total, nil
}

// dataToDTO converts database Data to a ProductDTO.
func (rm *ReadModelImpl) dataToDTO(data *m_product.Data, now time.Time) (*contracts.ProductDTO, error) {
	// Convert base price to float for display
	basePrice, err := domain.NewMoney(data.BasePriceNumerator, data.BasePriceDenominator)
	if err != nil {
		return nil, fmt.Errorf("invalid base price: %w", err)
	}

	basePriceFloat, _ := basePrice.Float64()
	dto := &contracts.ProductDTO{
		ProductID:      data.ProductID,
		Name:           data.Name,
		Description:    data.Description,
		Category:       data.Category,
		BasePrice:      basePriceFloat,
		EffectivePrice: basePriceFloat,
		Status:         data.Status,
		Version:        data.Version,
		CreatedAt:      data.CreatedAt,
		UpdatedAt:      data.UpdatedAt,
	}

	// Handle archived_at
	if data.ArchivedAt.Valid {
		dto.ArchivedAt = &data.ArchivedAt.Time
	}

	// Handle discount
	if data.DiscountPercent.Valid {
		percent, _ := data.DiscountPercent.Numeric.Float64()
		discount, err := domain.NewDiscount(
			percent, // Changed from Int64 to Float64
			data.DiscountStartDate.Time,
			data.DiscountEndDate.Time,
		)
		if err == nil && discount.IsValidAt(now) {
			dto.DiscountPercent = &percent // Changed from Int64 to Float64
			dto.DiscountActive = true
			effectivePrice := discount.Apply(basePrice)
			effectivePriceFloat, _ := effectivePrice.Float64()
			dto.EffectivePrice = effectivePriceFloat
		}
	}

	return dto, nil
}
