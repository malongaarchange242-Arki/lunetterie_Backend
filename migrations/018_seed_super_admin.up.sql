-- Seed d'un compte SUPER_ADMIN sans mot de passe initial.
-- Permet à l'utilisateur de définir son mot de passe via /auth/set-password lors de la première connexion.

INSERT INTO users (first_name, last_name, email, phone, gender, role_id, station_id, is_active, password_hash, password_hash_deprecated, created_at, updated_at)
SELECT 'Super', 'Admin', 'superadmin@lunetterie.com', NULL, NULL, r.id, NULL, true, NULL, NULL, NOW(), NOW()
FROM roles r
WHERE r.name = 'ADMIN'
  AND NOT EXISTS (SELECT 1 FROM users u WHERE u.email = 'superadmin@lunetterie.com');
