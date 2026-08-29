
-- ============================================
-- Création de l'utilisateur de test : Makaya Test
-- Rôle : PRE_ENREGISTREMENT
-- Station : Station Pointe-Noire
-- ============================================

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
    'Makayas' AS first_name,
    'Testes' AS last_name,
    'makaya.test@lunetterie.local' AS email,
    NULL AS phone,
    'Homme' AS gender,
    11 AS role_id,
    s.id AS station_id,
    true AS is_active,
    NULL AS password_hash,
    NULL AS password_hash_deprecated,
    NOW() AS created_at,
    NOW() AS updated_at
FROM stations s
WHERE s.name = 'Station Pointe-Noire'
  AND NOT EXISTS (
      SELECT 1
      FROM users
      WHERE email = 'makaya.test@lunetterie.local'
  );

-- ============================================
-- Vérification
-- ============================================

SELECT
    u.id,
    u.first_name,
    u.last_name,
    u.email,
    r.name AS role,
    s.name AS station,
    u.is_active,
    u.password_hash
FROM users u
LEFT JOIN roles r
    ON u.role_id = r.id
LEFT JOIN stations s
    ON u.station_id = s.id
WHERE u.email = 'makaya.test@lunetterie.local';

