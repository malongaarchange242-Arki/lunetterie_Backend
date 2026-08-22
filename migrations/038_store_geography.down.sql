DROP INDEX IF EXISTS idx_stations_ville_id;
DROP INDEX IF EXISTS idx_stations_pays_id;
ALTER TABLE stations
    DROP COLUMN IF EXISTS ville_id,
    DROP COLUMN IF EXISTS pays_id;
DROP TABLE IF EXISTS villes;
DROP TABLE IF EXISTS pays;
