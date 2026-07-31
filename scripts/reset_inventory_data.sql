-- ============================================================
-- Réinitialisation des données d'inventaire — GARDE stations, users,
-- roles, role_permissions, storage_locations. Vide tout le reste.
--
-- ⚠️ IRRÉVERSIBLE. Sauvegarder la base avant d'exécuter (Supabase >
-- Database > Backups), surtout que ce projet tourne en production.
--
-- À exécuter dans l'éditeur SQL Supabase, ou via psql :
--   psql "$DATABASE_URL" -f backend/scripts/reset_inventory_data.sql
-- ============================================================

BEGIN;

TRUNCATE TABLE
    -- Montures et tout ce qui en dépend
    movements,
    transfer_items,
    transfers,
    shape_corrections,
    glass_analysis,
    glasses,
    -- Référentiels IA / catalogue
    brands,
    shapes,
    colors,
    materials,
    mount_types,
    frame_models,
    -- Fournisseurs et livraisons (deliveries/delivery_items créées
    -- manuellement en base, absentes des migrations suivies ici)
    delivery_items,
    deliveries,
    suppliers
    RESTART IDENTITY CASCADE;

-- Optionnel : remet à zéro la numérotation des codes-barres
-- (LUN-CNG-00000001 au prochain enregistrement). Décommenter si voulu.
-- ALTER SEQUENCE barcode_seq RESTART WITH 1;

COMMIT;
