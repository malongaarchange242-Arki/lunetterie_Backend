-- Suppression du compte directeur et du rôle DIRECTION si besoin
DELETE FROM users WHERE email = 'directeur@gmail.com';
DELETE FROM roles WHERE name = 'DIRECTION';
