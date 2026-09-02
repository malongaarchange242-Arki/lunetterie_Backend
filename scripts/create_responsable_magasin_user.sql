-- =============================================================
-- Création d'un compte Responsable magasin
-- =============================================================
-- Rôle : RESPONSABLE_STATION
-- Station : Station Pointe-Noire
-- Mot de passe : test123
--
-- À adapter si vous voulez une autre ville / autre email.
-- =============================================================

-- L'interface de connexion associe la 3e position de la molette à l'ID 6.
-- Certaines bases existantes ne contiennent pas encore ce rôle.
INSERT INTO roles (id, name, description)
SELECT 6, 'RESPONSABLE_STATION', 'Responsable de station'
WHERE NOT EXISTS (SELECT 1 FROM roles WHERE name = 'RESPONSABLE_STATION')
    AND NOT EXISTS (SELECT 1 FROM roles WHERE id = 6);

SELECT setval(
        pg_get_serial_sequence('roles', 'id'),
        GREATEST((SELECT COALESCE(MAX(id), 1) FROM roles), 1)
);

INSERT INTO users (
    first_name,
    last_name,
    email,
    phone,
    gender,
    role_id,
    station_id,
    is_active,
    password_hash,
    password_hash_deprecated,
    created_at,
    updated_at
)
SELECT
    'Responsable' AS first_name,
    'Magasin' AS last_name,
    'responsable.magasin@lunetterie.local' AS email,
    '+242 000 00 00' AS phone,
    'Homme' AS gender,
    r.id AS role_id,
    s.id AS station_id,
    true AS is_active,
    '$2a$10$N9qo8uLOickgx2ZMRZoMyeIjZAgcg7b3XeKeUxWdeS86E36CHqV36' AS password_hash,
    NULL AS password_hash_deprecated,
    NOW() AS created_at,
    NOW() AS updated_at
FROM roles r
JOIN stations s ON s.name = 'Station Pointe-Noire'
WHERE r.name = 'RESPONSABLE_STATION'
  AND NOT EXISTS (
      SELECT 1 FROM users u WHERE lower(trim(u.email)) = lower(trim('responsable.magasin@lunetterie.local'))
  );

-- Répare également un compte déjà créé avec un mauvais rôle, une mauvaise
-- station ou un ancien code de connexion.
UPDATE users u
SET role_id = r.id,
        station_id = s.id,
        is_active = true,
        password_hash = '$2a$10$N9qo8uLOickgx2ZMRZoMyeIjZAgcg7b3XeKeUxWdeS86E36CHqV36',
        password_hash_deprecated = NULL,
        updated_at = NOW()
FROM roles r
JOIN stations s ON s.name = 'Station Pointe-Noire'
WHERE r.name = 'RESPONSABLE_STATION'
    AND lower(trim(u.email)) = lower(trim('responsable.magasin@lunetterie.local'));

SELECT
    u.id,
    u.first_name,
    u.last_name,
    u.email,
    r.name AS role,
    s.name AS station,
    u.is_active,
    u.password_hash IS NOT NULL AS has_password
FROM users u
LEFT JOIN roles r ON r.id = u.role_id
LEFT JOIN stations s ON s.id = u.station_id
WHERE lower(trim(u.email)) = lower(trim('responsable.magasin@lunetterie.local'));
