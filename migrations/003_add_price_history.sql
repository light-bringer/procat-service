-- Migration 003: Add price history table for audit trail
-- Purpose: Track all price changes for compliance and analytics
-- Retention: Keep indefinitely for audit purposes

CREATE TABLE price_history (
    history_id STRING(36) NOT NULL,
    product_id STRING(36) NOT NULL,
    old_price_numerator INT64,
    old_price_denominator INT64,
    new_price_numerator INT64 NOT NULL,
    new_price_denominator INT64 NOT NULL,
    changed_by STRING(255),  -- User/system identifier
    changed_reason STRING(MAX),  -- Optional explanation for price change
    changed_at TIMESTAMP NOT NULL OPTIONS (allow_commit_timestamp=true),
) PRIMARY KEY (product_id, history_id),
INTERLEAVE IN PARENT products ON DELETE CASCADE;

-- Index for querying price history by time (most recent first)
CREATE INDEX idx_price_history_product_time
ON price_history(product_id, changed_at DESC);
