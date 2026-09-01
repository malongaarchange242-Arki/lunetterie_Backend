-- Add missing Stock Général station and storage locations
INSERT INTO stations (name, type, city, is_active)
SELECT 'Stock Général', 'STOCK_GENERAL', 'Pointe-Noire', true
WHERE NOT EXISTS (SELECT 1 FROM stations WHERE name = 'Stock Général');

-- Add storage locations for Stock Général if it now exists
-- Create VALISE containers (top-level) that can hold glasses directly
DO $$
DECLARE
  v_stock_general_id BIGINT;
  v_i INT;
BEGIN
  SELECT id INTO v_stock_general_id FROM stations WHERE name = 'Stock Général' LIMIT 1;
  
  IF v_stock_general_id IS NOT NULL THEN
    -- Create 5 VALISE entries if none exist yet for this station
    IF NOT EXISTS (SELECT 1 FROM storage_locations WHERE station_id = v_stock_general_id AND zone = 'STOCK') THEN
      FOR v_i IN 1..5 LOOP
        INSERT INTO storage_locations (station_id, zone, code, type, status, capacity)
        SELECT v_stock_general_id, 'STOCK', 'VAL-' || LPAD(v_i::TEXT, 2, '0'), 'VALISE', 'LIBRE', 100
        WHERE NOT EXISTS (
          SELECT 1 FROM storage_locations 
          WHERE station_id = v_stock_general_id 
          AND code = 'VAL-' || LPAD(v_i::TEXT, 2, '0')
        );
      END LOOP;
    END IF;
  END IF;
END $$;



