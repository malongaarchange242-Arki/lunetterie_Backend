-- Le code d'un emplacement (ex: RAYON-A-ETA-01-BAC-A-POS-01) est maintenant généré avec le
-- même format dans plusieurs zones (STOCK, PRESENTOIR) pour une même station : l'unicité doit
-- donc être vérifiée par (station_id, zone, code), et non plus (station_id, code) seul.
ALTER TABLE storage_locations DROP CONSTRAINT IF EXISTS storage_locations_station_id_code_key;
ALTER TABLE storage_locations ADD CONSTRAINT storage_locations_station_id_zone_code_key UNIQUE (station_id, zone, code);
