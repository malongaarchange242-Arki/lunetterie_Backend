-- Rollback: Remove Stock Général station and its storage locations
DELETE FROM storage_locations WHERE station_id = (SELECT id FROM stations WHERE name = 'Stock Général' LIMIT 1);
DELETE FROM stations WHERE name = 'Stock Général';
