CREATE TABLE IF NOT EXISTS websites (
  url TEXT PRIMARY KEY,
  frequency INTEGER NOT NULL,
  last_checked_at INTEGER NOT NULL DEFAULT 0,
  created_at INTEGER NOT NULL,
  updated_at INTEGER NOT NULL,
  status TEXT NOT NULL,
  response_time INTEGER NOT NULL,
  status_code INTEGER NOT NULL,
  error TEXT NOT NULL DEFAULT '',
  webhook_enabled INTEGER NOT NULL DEFAULT 0,
  webhook_url TEXT NOT NULL DEFAULT '',
  webhook_payload_template TEXT NOT NULL DEFAULT ''
);

CREATE INDEX IF NOT EXISTS idx_websites_last_checked_at ON websites(last_checked_at);

CREATE TABLE IF NOT EXISTS webhook_deliveries (
  event_id TEXT PRIMARY KEY,
  website_url TEXT NOT NULL,
  webhook_url TEXT NOT NULL,
  payload TEXT NOT NULL,
  attempt_count INTEGER NOT NULL DEFAULT 0,
  max_attempts INTEGER NOT NULL,
  initial_delay_sec INTEGER NOT NULL DEFAULT 30,
  max_delay_sec INTEGER NOT NULL DEFAULT 300,
  backoff_factor REAL NOT NULL DEFAULT 2,
  next_attempt_at INTEGER NOT NULL,
  delivered_at INTEGER NOT NULL DEFAULT 0,
  last_error TEXT NOT NULL DEFAULT '',
  created_at INTEGER NOT NULL,
  updated_at INTEGER NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_webhook_deliveries_due ON webhook_deliveries(next_attempt_at, delivered_at);
