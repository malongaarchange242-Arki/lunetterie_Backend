-- Rollback de la refonte du modèle de stockage.
-- Restaure le comportement historique de compatibilité.

BEGIN;

DROP TRIGGER IF EXISTS trg_validate_storage_parent ON storage_locations;
DROP FUNCTION IF EXISTS validate_storage_parent();

ALTER TABLE storage_locations DROP CONSTRAINT IF EXISTS storage_locations_type_check_new;
ALTER TABLE storage_locations
    ADD CONSTRAINT storage_locations_type_check CHECK (
        type IN ('MEUBLE', 'VALISE', 'CARTON', 'PRESENTOIR', 'ARMOIRE', 'SALLE')
    );

ALTER TABLE storage_locations
    DROP CONSTRAINT IF EXISTS storage_locations_capacity_check;

CREATE UNIQUE INDEX IF NOT EXISTS idx_storage_locations_barcode
    ON storage_locations (barcode)
    WHERE barcode IS NOT NULL;

ALTER TABLE glasses DROP COLUMN IF EXISTS photo_branche_url;

COMMIT;
