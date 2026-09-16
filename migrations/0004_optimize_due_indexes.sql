-- The due-delivery query filters on `delivered_at = 0` and ranged on
-- `next_attempt_at`. The old index (next_attempt_at, delivered_at) cannot use
-- the `delivered_at` equality as a seek, so every cron tick walked the entire
-- (ever-growing) index even when nothing was pending.
DROP INDEX IF EXISTS idx_webhook_deliveries_due;

CREATE INDEX IF NOT EXISTS idx_webhook_deliveries_due
  ON webhook_deliveries(delivered_at, next_attempt_at);
