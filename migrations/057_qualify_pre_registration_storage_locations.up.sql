-- Les codes CTN sont locaux a une valise. Les emplacements physiques doivent cependant
-- rester uniques dans la station : on les qualifie avec le code de la valise.
UPDATE storage_locations AS carton_location
SET code = case_location.code || '-' || box.code,
    name = 'Carton ' || box.code,
    barcode = case_location.code || '-' || box.code
FROM storage_locations AS case_location
JOIN pre_registration_cases AS pre_case
  ON pre_case.code = case_location.code
JOIN pre_registration_boxes AS box
  ON box.case_id = pre_case.id
WHERE carton_location.parent_location_id = case_location.id
  AND case_location.type = 'VALISE'
  AND carton_location.type = 'CARTON'
  AND carton_location.code = box.code
  AND NOT EXISTS (
      SELECT 1
      FROM storage_locations AS existing_location
      WHERE existing_location.station_id = carton_location.station_id
        AND existing_location.zone = carton_location.zone
        AND existing_location.code = case_location.code || '-' || box.code
        AND existing_location.id <> carton_location.id
  );
