-- Vérifier si l'utilisateur PRE_ENREGISTREMENT existe avec le bon rôle
SELECT 
    u.id,
    u.first_name,
    u.last_name,
    u.email,
    u.role_id,
    r.name AS role_name,
    u.station_id,
    s.name AS station_name,
    u.is_active
FROM users u
LEFT JOIN roles r ON u.role_id = r.id
LEFT JOIN stations s ON u.station_id = s.id
WHERE LOWER(u.first_name || ' ' || u.last_name) LIKE '%preparateur%arrivages%'
   OR u.email = 'preparateur@lunetterie.local';

-- Afficher tous les rôles disponibles
SELECT id, name FROM roles ORDER BY id;

-- Vérifier que le rôle 11 (PRE_ENREGISTREMENT) existe
SELECT id, name, description FROM roles WHERE id = 11 OR name = 'PRE_ENREGISTREMENT';
