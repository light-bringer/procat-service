package m_price_history

// Table name constant
const TableName = "price_history"

// Field name constants for type-safe database access
const (
	HistoryID             = "history_id"
	ProductID             = "product_id"
	OldPriceNumerator     = "old_price_numerator"
	OldPriceDenominator   = "old_price_denominator"
	NewPriceNumerator     = "new_price_numerator"
	NewPriceDenominator   = "new_price_denominator"
	ChangedBy             = "changed_by"
	ChangedReason         = "changed_reason"
	ChangedAt             = "changed_at"
)
