package repo

import (
	"context"
	"fmt"
	"time"

	"cloud.google.com/go/spanner"
	"google.golang.org/api/iterator"

	"github.com/light-bringer/procat-service/internal/app/product/contracts"
	"github.com/light-bringer/procat-service/internal/app/product/domain"
	"github.com/light-bringer/procat-service/internal/models/m_price_history"
)

// PriceHistoryRepo implements PriceHistoryRepository for Spanner.
type PriceHistoryRepo struct {
	client *spanner.Client
	model  *m_price_history.Model
}

// NewPriceHistoryRepo creates a new PriceHistoryRepo.
func NewPriceHistoryRepo(client *spanner.Client) contracts.PriceHistoryRepository {
	return &PriceHistoryRepo{
		client: client,
		model:  m_price_history.NewModel(),
	}
}

// InsertMut creates a mutation for inserting a price change record.
func (r *PriceHistoryRepo) InsertMut(
	historyID string,
	productID string,
	oldPrice *domain.Money,
	newPrice *domain.Money,
	changedBy string,
	changedReason string,
	changedAt time.Time,
) *spanner.Mutation {
	// Normalize prices for consistent storage
	normalizedNewPrice := newPrice.Normalize()

	data := &m_price_history.Data{
		HistoryID:           historyID,
		ProductID:           productID,
		NewPriceNumerator:   normalizedNewPrice.Numerator(),
		NewPriceDenominator: normalizedNewPrice.Denominator(),
		ChangedAt:           changedAt,
	}

	// oldPrice is nil for initial product creation
	if oldPrice != nil {
		normalizedOldPrice := oldPrice.Normalize()
		data.OldPriceNumerator = spanner.NullInt64{
			Int64: normalizedOldPrice.Numerator(),
			Valid: true,
		}
		data.OldPriceDenominator = spanner.NullInt64{
			Int64: normalizedOldPrice.Denominator(),
			Valid: true,
		}
	}

	// changedBy is optional
	if changedBy != "" {
		data.ChangedBy = spanner.NullString{StringVal: changedBy, Valid: true}
	}

	// changedReason is optional
	if changedReason != "" {
		data.ChangedReason = spanner.NullString{StringVal: changedReason, Valid: true}
	}

	return r.model.InsertMut(data)
}

// GetByProductID retrieves price history for a product, ordered by time (most recent first).
func (r *PriceHistoryRepo) GetByProductID(ctx context.Context, productID string, limit int) ([]contracts.PriceHistoryRecord, error) {
	query := fmt.Sprintf(`
		SELECT %s
		FROM %s
		WHERE product_id = @productID
		ORDER BY changed_at DESC
		LIMIT @limit
	`, r.buildColumnList(), m_price_history.TableName)

	stmt := spanner.Statement{
		SQL: query,
		Params: map[string]interface{}{
			"productID": productID,
			"limit":     limit,
		},
	}

	iter := r.client.Single().Query(ctx, stmt)
	defer iter.Stop()

	var records []contracts.PriceHistoryRecord
	for {
		row, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("failed to iterate price history: %w", err)
		}

		var data m_price_history.Data
		if err := row.ToStruct(&data); err != nil {
			return nil, fmt.Errorf("failed to parse price history: %w", err)
		}

		record, err := r.dataToRecord(&data)
		if err != nil {
			return nil, err
		}

		records = append(records, *record)
	}

	return records, nil
}

// buildColumnList returns comma-separated column names for SELECT queries.
func (r *PriceHistoryRepo) buildColumnList() string {
	return fmt.Sprintf("%s, %s, %s, %s, %s, %s, %s, %s, %s",
		m_price_history.HistoryID,
		m_price_history.ProductID,
		m_price_history.OldPriceNumerator,
		m_price_history.OldPriceDenominator,
		m_price_history.NewPriceNumerator,
		m_price_history.NewPriceDenominator,
		m_price_history.ChangedBy,
		m_price_history.ChangedReason,
		m_price_history.ChangedAt,
	)
}

// dataToRecord converts database Data to domain PriceHistoryRecord.
func (r *PriceHistoryRepo) dataToRecord(data *m_price_history.Data) (*contracts.PriceHistoryRecord, error) {
	// newPrice is always present
	newPrice, err := domain.NewMoney(data.NewPriceNumerator, data.NewPriceDenominator)
	if err != nil {
		return nil, fmt.Errorf("invalid new price: %w", err)
	}

	record := &contracts.PriceHistoryRecord{
		HistoryID: data.HistoryID,
		ProductID: data.ProductID,
		NewPrice:  newPrice,
		ChangedAt: data.ChangedAt,
	}

	// oldPrice is nil for initial creation
	if data.OldPriceNumerator.Valid && data.OldPriceDenominator.Valid {
		oldPrice, err := domain.NewMoney(data.OldPriceNumerator.Int64, data.OldPriceDenominator.Int64)
		if err != nil {
			return nil, fmt.Errorf("invalid old price: %w", err)
		}
		record.OldPrice = oldPrice
	}

	// changedBy is optional
	if data.ChangedBy.Valid {
		record.ChangedBy = data.ChangedBy.StringVal
	}

	// changedReason is optional
	if data.ChangedReason.Valid {
		record.ChangedReason = data.ChangedReason.StringVal
	}

	return record, nil
}
