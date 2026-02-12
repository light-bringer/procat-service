-- Product Catalog Service - Initial Schema

-- Products table
CREATE TABLE IF NOT EXISTS products (
  product_id STRING(36) NOT NULL,
  name STRING(255) NOT NULL,
  description STRING(MAX),
  category STRING(100) NOT NULL,
  -- Price stored as rational number (numerator/denominator)
  base_price_numerator INT64 NOT NULL,
  base_price_denominator INT64 NOT NULL,
  -- Discount fields (initially INT64, migrated to NUMERIC in 002)
  discount_percent INT64,
  discount_start_date TIMESTAMP,
  discount_end_date TIMESTAMP,
  -- Status: active, inactive, archived
  status STRING(20) NOT NULL,
  -- Timestamps
  created_at TIMESTAMP NOT NULL OPTIONS (allow_commit_timestamp=true),
  updated_at TIMESTAMP NOT NULL OPTIONS (allow_commit_timestamp=true),
  archived_at TIMESTAMP,
) PRIMARY KEY (product_id);

-- Index for listing products by category
CREATE INDEX IF NOT EXISTS idx_products_category_status ON products(category, status, created_at DESC);

-- Index for listing active products
CREATE INDEX IF NOT EXISTS idx_products_status_created ON products(status, created_at DESC);

-- Outbox events table for transactional event publishing
CREATE TABLE IF NOT EXISTS outbox_events (
  event_id STRING(36) NOT NULL,
  event_type STRING(100) NOT NULL,
  aggregate_id STRING(36) NOT NULL,
  payload JSON NOT NULL,
  status STRING(20) NOT NULL,  -- pending, processing, completed, failed
  created_at TIMESTAMP NOT NULL OPTIONS (allow_commit_timestamp=true),
  processed_at TIMESTAMP,
  retry_count INT64 NOT NULL DEFAULT (0),
  error_message STRING(MAX),
) PRIMARY KEY (event_id);

-- Index for polling pending events
CREATE INDEX IF NOT EXISTS idx_outbox_status_created ON outbox_events(status, created_at);

-- Index for finding events by aggregate
CREATE INDEX IF NOT EXISTS idx_outbox_aggregate ON outbox_events(aggregate_id, created_at DESC);
