-- Supprime le compte SUPER_ADMIN seedé par 018_seed_super_admin.up.sql.
DELETE FROM users WHERE email = 'superadmin@lunetterie.com';
