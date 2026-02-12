-- Migration 005: Migrate discount_percent from INT64 to NUMERIC
-- Purpose: Support fractional discount percentages (e.g., 12.5%, 7.25%)
-- Background: Initially stored as INT64 (whole numbers only), now needs precision
--
-- NUMERIC type in Spanner supports arbitrary-precision decimal values,
-- allowing fractional percentages while maintaining exact representation.
--
-- Migration strategy:
-- 1. Add new NUMERIC column
-- 2. Copy data (INT64 values convert directly to NUMERIC)
-- 3. Drop old INT64 column (handled by ALTER COLUMN)
--
-- Note: This is a non-breaking change as INT64 values (e.g., 20)
-- are valid NUMERIC values (20.0)

ALTER TABLE products ALTER COLUMN discount_percent NUMERIC;
