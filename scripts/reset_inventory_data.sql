-- ============================================================================
-- Réinitialisation des données métier.
--
-- PRÉSERVE : users, roles, role_permissions, stations, storage_locations et
-- les tables dont ces tables dépendent par clé étrangère.
-- VIDE : toutes les autres tables de l'espace public, y compris les données
-- d'inventaire, commandes, ventes, livraisons, catalogue et notifications.
--
-- ⚠️ IRRÉVERSIBLE : faire une sauvegarde avant exécution.
-- psql "$DATABASE_URL" -v ON_ERROR_STOP=1 -f backend/scripts/reset_inventory_data.sql
-- ============================================================================

BEGIN;

DO $$
DECLARE
    tables_a_garder CONSTANT text[] := ARRAY[
        'users', 'roles', 'role_permissions', 'stations', 'storage_locations',
        'schema_migrations'
    ];
    tables_a_vider text;
BEGIN
    -- La fermeture transitive protège les tables nécessaires aux tables conservées
    -- (par exemple roles et stations référencées par users).
    WITH RECURSIVE gardees AS (
        SELECT c.oid
            FROM pg_class c
            JOIN pg_namespace n ON n.oid = c.relnamespace
         WHERE n.nspname = 'public'
             AND c.relkind IN ('r', 'p')
             AND c.relname = ANY (tables_a_garder)
        UNION
        SELECT con.confrelid
            FROM pg_constraint con
            JOIN gardees g ON g.oid = con.conrelid
         WHERE con.contype = 'f'
    )
    SELECT string_agg(format('%I.%I', n.nspname, c.relname), ', ' ORDER BY c.relname)
        INTO tables_a_vider
        FROM pg_class c
        JOIN pg_namespace n ON n.oid = c.relnamespace
     WHERE n.nspname = 'public'
         AND c.relkind IN ('r', 'p')
         AND c.oid NOT IN (SELECT oid FROM gardees);

    IF tables_a_vider IS NOT NULL THEN
        RAISE NOTICE 'Tables vidées : %', tables_a_vider;
        EXECUTE format('TRUNCATE TABLE %s RESTART IDENTITY CASCADE', tables_a_vider);
    ELSE
        RAISE NOTICE 'Aucune table à vider.';
    END IF;
END $$;

-- Les emplacements sont conservés, mais leur occupation métier est remise à zéro.
UPDATE storage_locations
     SET status = 'LIBRE'
 WHERE status <> 'MAINTENANCE';

-- Séquences indépendantes de TRUNCATE utilisées pour les identifiants métier.
ALTER SEQUENCE IF EXISTS barcode_seq RESTART WITH 1;
ALTER SEQUENCE IF EXISTS valise_code_seq RESTART WITH 1;
ALTER SEQUENCE IF EXISTS carton_code_seq RESTART WITH 1;

-- Contrôle avant validation : users/roles/stations doivent toujours exister.
SELECT 'users' AS table_name, count(*) AS row_count FROM users
UNION ALL
SELECT 'roles', count(*) FROM roles
UNION ALL
SELECT 'stations', count(*) FROM stations
UNION ALL
SELECT 'storage_locations', count(*) FROM storage_locations;

COMMIT;
