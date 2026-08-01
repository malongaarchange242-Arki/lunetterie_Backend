-- Seed du compte directeur/admin
ALTER TABLE users ADD COLUMN IF NOT EXISTS password_hash VARCHAR(255);

INSERT INTO roles (name, description) VALUES
    ('ADMIN', 'Administrateur')
ON CONFLICT DO NOTHING;

INSERT INTO users (first_name, last_name, email, phone, gender, role_id, station_id, is_active, password_hash, password_hash_deprecated, created_at, updated_at)
SELECT 'Directeur', 'Admin', 'admin@gmail.com', NULL, NULL, r.id, NULL, true,
       '$2a$10$zj..qcQe1z9Sn69XvnVzbO3XETqaIecdKUN.QnIB.l6HaNIKs4rUW',
       NULL,
       NOW(), NOW()
FROM roles r
WHERE r.name = 'ADMIN'
  AND NOT EXISTS (SELECT 1 FROM users u WHERE u.email = 'admin@gmail.com');
