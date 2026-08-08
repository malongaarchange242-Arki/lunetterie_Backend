ALTER TABLE send_lists
    DROP COLUMN IF EXISTS sent_count,
    DROP COLUMN IF EXISTS destination_station_name;
