DROP INDEX IF EXISTS idx_send_boxes_pending;

ALTER TABLE send_boxes
    DROP COLUMN IF EXISTS opened_station_id,
    DROP COLUMN IF EXISTS opened_by,
    DROP COLUMN IF EXISTS opened_at;
