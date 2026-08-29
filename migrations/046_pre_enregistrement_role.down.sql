DELETE FROM roles
WHERE name = 'PRE_ENREGISTREMENT'
  AND NOT EXISTS (SELECT 1 FROM users u WHERE u.role_id = roles.id);
