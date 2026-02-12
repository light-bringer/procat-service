-- Migration 005: Add outbox retention policy documentation
-- Purpose: Define retention policy for processed outbox events

-- RETENTION POLICY:
-- - Processed events (status = 'completed') are retained for 30 days
-- - Failed events (status = 'failed') are retained for 90 days for debugging
-- - Pending events (status = 'pending') are never deleted (active queue)
--
-- Cleanup is performed by a background job (cmd/cleanup_outbox)
-- scheduled to run daily via cron/Cloud Scheduler
--
-- Query to find old events for cleanup:
-- DELETE FROM outbox_events
-- WHERE (status = 'completed' AND processed_at < TIMESTAMP_SUB(CURRENT_TIMESTAMP(), INTERVAL 30 DAY))
--    OR (status = 'failed' AND processed_at < TIMESTAMP_SUB(CURRENT_TIMESTAMP(), INTERVAL 90 DAY))

-- Note: No schema changes required - this migration documents the policy only.
-- The cleanup job handles deletion based on these rules.
