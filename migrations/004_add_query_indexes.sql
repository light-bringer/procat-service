-- Migration 004: Add indexes for common query patterns
-- Purpose: Optimize list/filter operations on products and events

-- Index for filtering products by category and sorting by created_at
-- Supports query: SELECT * FROM products WHERE category = ? ORDER BY created_at DESC
CREATE INDEX idx_products_category_created
ON products(category, created_at DESC);

-- Index for filtering products by status and sorting by created_at
-- Supports query: SELECT * FROM products WHERE status = ? ORDER BY created_at DESC
CREATE INDEX idx_products_status_created
ON products(status, created_at DESC);

-- Index for general product listing sorted by created_at
-- Supports query: SELECT * FROM products ORDER BY created_at DESC
CREATE INDEX idx_products_created
ON products(created_at DESC);

-- Index for outbox events by status and created_at (for event polling)
-- Supports query: SELECT * FROM outbox_events WHERE status = ? ORDER BY created_at DESC
CREATE INDEX idx_outbox_status_created
ON outbox_events(status, created_at DESC);
