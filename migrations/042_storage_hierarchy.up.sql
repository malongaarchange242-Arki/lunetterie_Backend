-- Prepare the physical storage hierarchy without changing existing locations yet.
ALTER TABLE storage_locations
    ADD COLUMN IF NOT EXISTS barcode VARCHAR(128),
    ADD COLUMN IF NOT EXISTS capacity INTEGER;

DO $$
BEGIN
    IF EXISTS (
        SELECT 1
        FROM storage_locations
        WHERE type IN ('RAYON', 'ETAGERE', 'BAC', 'POSITION')
    ) THEN
        RAISE EXCEPTION 'Migration bloquee: des emplacements historiques existent encore (RAYON/ETAGERE/BAC/POSITION)';
    END IF;
END;
$$;

ALTER TABLE storage_locations
    DROP CONSTRAINT IF EXISTS storage_locations_type_check;

ALTER TABLE storage_locations
    ADD CONSTRAINT storage_locations_type_check_v2 CHECK (
        type IN (
            'MEUBLE', 'VALISE', 'CARTON',
            'PRESENTOIR', 'ARMOIRE', 'SALLE'
        )
    );

ALTER TABLE storage_locations
    ADD CONSTRAINT storage_locations_capacity_check CHECK (capacity IS NULL OR capacity >= 0);

CREATE UNIQUE INDEX IF NOT EXISTS idx_storage_locations_barcode
    ON storage_locations (barcode)
    WHERE barcode IS NOT NULL;

-- Generate only the physical hierarchy. The fifth argument remains for API compatibility
-- with existing callers and is intentionally ignored: lunettes share their carton.
CREATE OR REPLACE FUNCTION generate_station_locations(
    p_station_id BIGINT,
    p_num_meubles INTEGER DEFAULT 5,
    p_valises_par_meuble INTEGER DEFAULT 8,
    p_cartons_par_valise INTEGER DEFAULT 4,
    p_positions_par_carton INTEGER DEFAULT 20
) RETURNS INTEGER AS $$
DECLARE
    meuble_id BIGINT;
    valise_id BIGINT;
    carton_id BIGINT;
    meuble_index INTEGER;
    valise_index INTEGER;
    carton_index INTEGER;
    total_locations INTEGER := 0;
BEGIN
    FOR meuble_index IN 1..p_num_meubles LOOP
        INSERT INTO storage_locations (
            station_id, parent_location_id, zone, code, name, type, status
        ) VALUES (
            p_station_id, NULL, 'STOCK',
            'MEUBLE-' || meuble_index,
            'Meuble ' || meuble_index,
            'MEUBLE', 'LIBRE'
        ) RETURNING id INTO meuble_id;

        UPDATE storage_locations
        SET barcode = 'MEU-CNG-' || meuble_id
        WHERE id = meuble_id;
        total_locations := total_locations + 1;

        FOR valise_index IN 1..p_valises_par_meuble LOOP
            INSERT INTO storage_locations (
                station_id, parent_location_id, zone, code, name, type, status
            ) VALUES (
                p_station_id, meuble_id, 'STOCK',
                'MEUBLE-' || meuble_index || '-VALISE-' || valise_index,
                'Valise ' || valise_index,
                'VALISE', 'LIBRE'
            ) RETURNING id INTO valise_id;

            UPDATE storage_locations
            SET barcode = 'VAL-CNG-' || valise_id
            WHERE id = valise_id;
            total_locations := total_locations + 1;

            FOR carton_index IN 1..p_cartons_par_valise LOOP
                INSERT INTO storage_locations (
                    station_id, parent_location_id, zone, code, name, type, status
                ) VALUES (
                    p_station_id, valise_id, 'STOCK',
                    'MEUBLE-' || meuble_index || '-VALISE-' || valise_index || '-CARTON-' || carton_index,
                    'Carton ' || carton_index,
                    'CARTON', 'LIBRE'
                ) RETURNING id INTO carton_id;

                UPDATE storage_locations
                SET barcode = 'CAR-CNG-' || carton_id
                WHERE id = carton_id;
                total_locations := total_locations + 1;
            END LOOP;
        END LOOP;
    END LOOP;

    RETURN total_locations;
END;
$$ LANGUAGE plpgsql;
