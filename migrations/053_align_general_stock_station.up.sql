-- Align legacy general-stock glasses with the canonical Stock Général station.
-- Older imports used a different station while retaining EN_STOCK_GENERAL.
UPDATE glasses AS g
SET station_id = stock.id,
    location_id = CASE
      WHEN EXISTS (
        SELECT 1
        FROM storage_locations AS location
        WHERE location.id = g.location_id
          AND location.station_id = stock.id
      ) THEN g.location_id
      ELSE NULL
    END,
    updated_at = NOW()
FROM stations AS stock
WHERE stock.name = 'Stock Général'
  AND g.status IN ('EN_STOCK_GENERAL', 'RESERVEE_ENVOI')
  AND g.station_id <> stock.id;
