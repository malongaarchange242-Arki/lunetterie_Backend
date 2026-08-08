-- ============================================
-- POSTE CAISSE
-- Le poste d'encaissement réutilise presentoir.html, comme le Laboratoire et les
-- magasins : il lui faut donc sa station, son rôle, son statut et son action de
-- mouvement. La monture y arrive au scan (le client l'apporte au comptoir), puis
-- l'encaissement la passe en VENDUE via le flux de vente déjà en place.
-- ============================================

-- 1. La station. Même forme que 009_seed_stations : réexécutable sans doublon.
INSERT INTO stations (name, type, city)
SELECT 'Caisse', 'SOUS_STATION', 'Brazzaville'
WHERE NOT EXISTS (SELECT 1 FROM stations WHERE name = 'Caisse');

-- 2. Le rôle. Frontend/admin.js associe les rôles à des identifiants écrits en dur :
-- laisser la séquence décider rendrait la création d'un caissier dépendante de l'ordre
-- d'exécution des migrations. On fixe donc l'id à 9, sans écraser ce qui l'occuperait.
INSERT INTO roles (id, name, description)
SELECT 9, 'CAISSIER', 'Caissier - encaissement en magasin'
WHERE NOT EXISTS (SELECT 1 FROM roles WHERE name = 'CAISSIER')
  AND NOT EXISTS (SELECT 1 FROM roles WHERE id = 9);

-- Repli si l'id 9 était déjà pris : le rôle existe quand même, avec l'id suivant.
INSERT INTO roles (name, description)
SELECT 'CAISSIER', 'Caissier - encaissement en magasin'
WHERE NOT EXISTS (SELECT 1 FROM roles WHERE name = 'CAISSIER');

-- La séquence doit repasser devant l'id posé à la main, sinon la prochaine création de
-- rôle buterait sur une clé déjà prise.
SELECT setval(pg_get_serial_sequence('roles', 'id'), (SELECT MAX(id) FROM roles));

-- 3. Le statut EN_CAISSE. La contrainte est déclarée en ligne dans 001_init, donc
-- nommée glasses_status_check par PostgreSQL ; on la remplace en conservant la
-- liste existante à l'identique.
ALTER TABLE glasses DROP CONSTRAINT IF EXISTS glasses_status_check;
ALTER TABLE glasses ADD CONSTRAINT glasses_status_check CHECK (status IN (
    'RECU_FOURNISSEUR', 'EN_STOCK_GENERAL', 'EN_TRANSIT',
    'EN_STOCK_SOUS_STATION', 'EN_PRESENTOIR', 'EN_CAISSE', 'RESERVEE',
    'EN_LABORATOIRE', 'PRETE_A_LIVRER', 'VENDUE',
    'PERDUE', 'CASSEE', 'RETOURNEE'
));

-- 4. L'action de mouvement correspondante, pour que l'historique distingue une mise
-- en caisse d'une mise en présentoir.
ALTER TABLE movements DROP CONSTRAINT IF EXISTS movements_action_check;
ALTER TABLE movements ADD CONSTRAINT movements_action_check CHECK (action IN (
    'RECEPTION_FOURNISSEUR', 'RANGEMENT', 'EXPEDITION', 'RECEPTION_STATION',
    'PRESENTOIR', 'MISE_EN_CAISSE', 'RETRAIT_PRESENTOIR', 'RESERVATION', 'LABORATOIRE',
    'CONTROLE_QUALITE', 'LIVRAISON', 'RETOUR', 'INVENTAIRE', 'PERTE', 'CASSE'
));
