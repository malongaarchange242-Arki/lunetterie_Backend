ALTER TABLE storage_locations DROP CONSTRAINT IF EXISTS storage_locations_station_id_zone_code_key;
ALTER TABLE storage_locations ADD CONSTRAINT storage_locations_station_id_code_key UNIQUE (station_id, code);
