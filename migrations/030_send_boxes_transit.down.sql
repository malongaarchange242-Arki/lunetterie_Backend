DROP INDEX IF EXISTS idx_send_boxes_transfer;

ALTER TABLE send_boxes
    DROP COLUMN IF EXISTS transfer_id,
    DROP COLUMN IF EXISTS closed_at,
    DROP COLUMN IF EXISTS closed_by,
    DROP COLUMN IF EXISTS missing_count;
