ALTER TABLE websites ADD COLUMN last_unhealthy_notification_at INTEGER NOT NULL DEFAULT 0;
ALTER TABLE websites ADD COLUMN last_unhealthy_notification_type TEXT NOT NULL DEFAULT '';
