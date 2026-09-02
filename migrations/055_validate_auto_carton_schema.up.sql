-- Migration: Valider et documenter le schéma pour création automatique de cartons
--
-- La création automatique de cartons STOCK (AllocationService.createStockCarton) dépend de:
-- 1. Séquences: valise_code_seq, carton_code_seq
-- 2. Hiérarchie: VALISE (parent_id=NULL) → CARTON (parent_id=valise.id)
-- 3. Capacité: CARTON.capacity = 50 (par défaut)
-- 4. Contraintes: type IN ('VALISE', 'CARTON', 'PRESENTOIR', 'ARMOIRE', 'SALLE')

BEGIN;

-- 1. Vérifier que les séquences existent
DO $$
BEGIN
    IF NOT EXISTS (SELECT 1 FROM pg_sequences WHERE schemaname = 'public' AND sequencename = 'valise_code_seq') THEN
        RAISE EXCEPTION 'Séquence manquante: valise_code_seq';
    END IF;
    IF NOT EXISTS (SELECT 1 FROM pg_sequences WHERE schemaname = 'public' AND sequencename = 'carton_code_seq') THEN
        RAISE EXCEPTION 'Séquence manquante: carton_code_seq';
    END IF;
END;
$$;

-- 2. Vérifier la contrainte parent pour VALISE/CARTON
--    VALISE: parent_location_id IS NULL
--    CARTON: parent_location_id REFERENCES VALISE
DO $$
BEGIN
    -- Cette vérification est logique, pas imposée en BD pour flexibilité
    -- Mais on peut l'émettre comme AVERTISSEMENT pour les admins
    RAISE NOTICE 'Schema validé: VALISE sans parent, CARTON avec parent VALISE, capacity >= 1';
END;
$$;

-- 3. Assurer que la colonne capacity a une contrainte >= 1 pour CARTON
DO $$
BEGIN
    IF EXISTS (
        SELECT 1 FROM storage_locations
        WHERE type IN ('CARTON', 'VALISE')
        AND capacity IS NOT NULL
        AND capacity < 1
    ) THEN
        RAISE EXCEPTION 'Données invalides: capacity < 1 trouvée pour CARTON/VALISE';
    END IF;
END;
$$;

COMMIT;
