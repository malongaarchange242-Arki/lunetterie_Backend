DROP INDEX IF EXISTS idx_storage_locations_barcode;

ALTER TABLE storage_locations
    DROP CONSTRAINT IF EXISTS storage_locations_capacity_check;

ALTER TABLE storage_locations
    DROP CONSTRAINT IF EXISTS storage_locations_type_check_v2;

ALTER TABLE storage_locations
    ADD CONSTRAINT storage_locations_type_check CHECK (
        type IN ('MEUBLE', 'VALISE', 'CARTON', 'PRESENTOIR', 'ARMOIRE', 'SALLE')
    );

ALTER TABLE storage_locations
    DROP COLUMN IF EXISTS barcode,
    DROP COLUMN IF EXISTS capacity;
