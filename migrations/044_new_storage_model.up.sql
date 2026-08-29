-- Refondation du modèle de stockage vers la hiérarchie cible :
-- VALISE -> CARTON -> LUNETTE
-- Aucune réintroduction de MEUBLE / RAYON / ETAGERE / BAC / POSITION.

BEGIN;

-- 1) Assurer la présence de la colonne photo branche, rétablie dans le schéma métier.
ALTER TABLE glasses ADD COLUMN IF NOT EXISTS photo_branche_url TEXT;

-- 2) Contrainte de type strictement réduite au modèle cible.
ALTER TABLE storage_locations DROP CONSTRAINT IF EXISTS storage_locations_type_check_v2;
ALTER TABLE storage_locations DROP CONSTRAINT IF EXISTS storage_locations_type_check;
ALTER TABLE storage_locations
    ADD CONSTRAINT storage_locations_type_check_new CHECK (
        type IN ('VALISE', 'CARTON', 'PRESENTOIR', 'ARMOIRE', 'SALLE')
    );

-- 3) Capacité valide et unique pour le nouveau schéma.
ALTER TABLE storage_locations
    ADD COLUMN IF NOT EXISTS capacity INTEGER;

ALTER TABLE storage_locations
    ADD COLUMN IF NOT EXISTS barcode VARCHAR(128);

ALTER TABLE storage_locations
    ADD CONSTRAINT storage_locations_capacity_check CHECK (capacity IS NULL OR capacity >= 1);

-- 4) Barcode unique sur les emplacements si renseigné.
CREATE UNIQUE INDEX IF NOT EXISTS idx_storage_locations_barcode
    ON storage_locations (barcode)
    WHERE barcode IS NOT NULL;

-- 5) Les anciens objets historiques ne sont plus admis dans le nouveau modèle.
DO $$
BEGIN
    IF EXISTS (
        SELECT 1
        FROM storage_locations
        WHERE type IN ('MEUBLE', 'RAYON', 'ETAGERE', 'BAC', 'POSITION')
    ) THEN
        RAISE EXCEPTION 'Migration bloquee: anciens types de stockage historiques detectes (MEUBLE/RAYON/ETAGERE/BAC/POSITION)';
    END IF;
END;
$$;

-- 6) Ajout d'une contrainte de parent stricte :
--    VALISE : parent null
--    CARTON : parent sur VALISE
--    Tout autre parent est invalide dans le nouveau modèle.
CREATE OR REPLACE FUNCTION validate_storage_parent()
RETURNS TRIGGER AS $$
BEGIN
    IF NEW.type = 'VALISE' AND NEW.parent_location_id IS NOT NULL THEN
        RAISE EXCEPTION 'Une VALISE ne peut pas avoir de parent';
    END IF;

    IF NEW.type = 'CARTON' THEN
        IF NEW.parent_location_id IS NULL THEN
            RAISE EXCEPTION 'Un CARTON doit appartenir à une VALISE';
        END IF;

        IF EXISTS (
            SELECT 1
            FROM storage_locations parent
            WHERE parent.id = NEW.parent_location_id
              AND parent.type <> 'VALISE'
        ) THEN
            RAISE EXCEPTION 'Un CARTON doit avoir une VALISE comme parent';
        END IF;
    END IF;

    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

DROP TRIGGER IF EXISTS trg_validate_storage_parent ON storage_locations;
CREATE TRIGGER trg_validate_storage_parent
    BEFORE INSERT OR UPDATE OF parent_location_id, type
    ON storage_locations
    FOR EACH ROW
    EXECUTE FUNCTION validate_storage_parent();

-- 7) Normaliser la création de codes physiques :
--    VAL-CNG-{id} pour une VALISE, CAR-CNG-{id} pour un CARTON.
--    Les anciennes générations MEU-CNG-{id} ne sont plus autorisées dans ce modèle.
UPDATE storage_locations
SET barcode = CONCAT(type, '-CNG-', id)
WHERE barcode IS NULL AND type IN ('VALISE', 'CARTON');

-- 8) Les codes de stock de la phase historique ne sont plus acceptés dans le nouveau monde.
--    On les laisse uniquement en pur visuel s'il existe déjà, mais l'accord métier cible est
--    de ne plus utiliser ces types ni leurs conventionnements.

COMMIT;
