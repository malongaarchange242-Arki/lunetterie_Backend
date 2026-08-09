-- ============================================
-- POSTE SAV (service après-vente)
-- Le SAV suit le client une fois la vente faite : relances, appels, réclamations.
-- Il ne détient aucune monture — donc pas de station, pas de statut de monture ni
-- d'action de mouvement, contrairement au poste Caisse (025_caisse).
-- ============================================

-- Frontend/admin.js associe les rôles à des identifiants écrits en dur : laisser la
-- séquence décider rendrait la création d'un agent SAV dépendante de l'ordre
-- d'exécution des migrations. On fixe donc l'id à 10, sans écraser ce qui l'occuperait.
INSERT INTO roles (id, name, description)
SELECT 10, 'SAV', 'Service après-vente - suivi et relance client'
WHERE NOT EXISTS (SELECT 1 FROM roles WHERE name = 'SAV')
  AND NOT EXISTS (SELECT 1 FROM roles WHERE id = 10);

-- Repli si l'id 10 était déjà pris : le rôle existe quand même, avec l'id suivant.
INSERT INTO roles (name, description)
SELECT 'SAV', 'Service après-vente - suivi et relance client'
WHERE NOT EXISTS (SELECT 1 FROM roles WHERE name = 'SAV');

-- La séquence doit repasser devant l'id posé à la main, sinon la prochaine création de
-- rôle buterait sur une clé déjà prise.
SELECT setval(pg_get_serial_sequence('roles', 'id'), (SELECT MAX(id) FROM roles));
