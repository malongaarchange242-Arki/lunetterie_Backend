INSERT INTO storage_locations (
    station_id,
    parent_location_id,
    zone,
    code,
    type,
    status,
    name
)
SELECT
    bac.station_id,
    bac.id,
    bac.zone,
    bac.code || '-POS-' || LPAD(pos.n::text, 2, '0'),
    'POSITION',
    'LIBRE',
    'Position ' || pos.n
FROM storage_locations bac
CROSS JOIN generate_series(1, 20) AS pos(n)
WHERE bac.type = 'BAC'
  AND NOT EXISTS (
      SELECT 1
      FROM storage_locations existing
      WHERE existing.station_id = bac.station_id
        AND existing.zone = bac.zone
        AND existing.code = bac.code || '-POS-' || LPAD(pos.n::text, 2, '0')
  );