ALTER TABLE send_lists
    ADD COLUMN IF NOT EXISTS sent_count INTEGER NOT NULL DEFAULT 0,
    ADD COLUMN IF NOT EXISTS destination_station_name VARCHAR(160) NULL;

CREATE INDEX IF NOT EXISTS idx_send_lists_dispatch_metadata
    ON send_lists(status, sent_count, destination_station_name);
