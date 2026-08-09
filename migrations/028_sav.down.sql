-- Le rôle ne peut pas disparaître tant qu'un compte le porte : users.role_id le
-- référence. On le laisse en place dans ce cas plutôt que de casser la contrainte.
DELETE FROM roles
WHERE name = 'SAV'
  AND NOT EXISTS (SELECT 1 FROM users WHERE users.role_id = roles.id);

SELECT setval(pg_get_serial_sequence('roles', 'id'), (SELECT MAX(id) FROM roles));
