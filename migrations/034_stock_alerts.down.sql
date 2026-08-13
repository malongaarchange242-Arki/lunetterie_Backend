DROP INDEX IF EXISTS idx_notifications_unread;
DROP INDEX IF EXISTS idx_notifications_user;
DROP TABLE IF EXISTS notifications;
DROP TABLE IF EXISTS stock_alert_states;

ALTER TABLE stations
    DROP COLUMN IF EXISTS presentoir_reference_qty,
    DROP COLUMN IF EXISTS local_reference_qty;
