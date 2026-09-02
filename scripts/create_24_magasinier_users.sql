-- Création idempotente de 24 comptes opérationnels.
-- Mot de passe initial : 1234
-- À changer après la première connexion.
-- 12 comptes PRE_ENREGISTREMENT et 12 comptes MAGASINIER (stock général).

CREATE EXTENSION IF NOT EXISTS pgcrypto;

BEGIN;

DO $$
DECLARE
    pre_role_id BIGINT;
    stock_role_id BIGINT;
    pre_station_id BIGINT;
    stock_station_id BIGINT;
    password_hash_value TEXT := crypt('1234', gen_salt('bf', 10));
    i INTEGER;
BEGIN
    SELECT id INTO pre_role_id FROM roles WHERE name = 'PRE_ENREGISTREMENT' LIMIT 1;
    SELECT id INTO stock_role_id FROM roles WHERE name = 'MAGASINIER' LIMIT 1;
    SELECT id INTO pre_station_id FROM stations WHERE name IN ('Station Pointe-Noire', 'Pointe-Noire') ORDER BY id LIMIT 1;
    SELECT id INTO stock_station_id FROM stations WHERE name IN ('Stock Général', 'Stock general') ORDER BY id LIMIT 1;

    IF pre_role_id IS NULL THEN
        RAISE EXCEPTION 'Rôle PRE_ENREGISTREMENT introuvable';
    END IF;
    IF stock_role_id IS NULL THEN
        RAISE EXCEPTION 'Rôle MAGASINIER introuvable';
    END IF;
    IF pre_station_id IS NULL THEN
        RAISE EXCEPTION 'Station de pré-enregistrement introuvable';
    END IF;
    IF stock_station_id IS NULL THEN
        RAISE EXCEPTION 'Station Stock Général introuvable';
    END IF;

    FOR i IN 1..12 LOOP
        INSERT INTO users (
            first_name, last_name, email, role_id, station_id, is_active,
            password_hash, created_at, updated_at
        )
        SELECT
            'PreEnregistrement', LPAD(i::TEXT, 2, '0'),
            'pre-enregistrement' || LPAD(i::TEXT, 2, '0') || '@lunetterie.local',
            pre_role_id, pre_station_id, true,
            password_hash_value, NOW(), NOW()
        WHERE NOT EXISTS (
            SELECT 1 FROM users
            WHERE lower(trim(email)) = 'pre-enregistrement' || LPAD(i::TEXT, 2, '0') || '@lunetterie.local'
        );

        INSERT INTO users (
            first_name, last_name, email, role_id, station_id, is_active,
            password_hash, created_at, updated_at
        )
        SELECT
            'StockGeneral', LPAD(i::TEXT, 2, '0'),
            'stock-general' || LPAD(i::TEXT, 2, '0') || '@lunetterie.local',
            stock_role_id, stock_station_id, true,
            password_hash_value, NOW(), NOW()
        WHERE NOT EXISTS (
            SELECT 1 FROM users
            WHERE lower(trim(email)) = 'stock-general' || LPAD(i::TEXT, 2, '0') || '@lunetterie.local'
        );
    END LOOP;
END $$;

COMMIT;

SELECT
    u.id,
    u.first_name,
    u.last_name,
    u.email,
    r.name AS role,
    s.name AS station,
    u.is_active,
    (u.password_hash IS NOT NULL) AS has_password
FROM users u
JOIN roles r ON r.id = u.role_id
LEFT JOIN stations s ON s.id = u.station_id
WHERE u.email LIKE 'pre-enregistrement%@lunetterie.local'
   OR u.email LIKE 'stock-general%@lunetterie.local'
ORDER BY r.name, u.email;
