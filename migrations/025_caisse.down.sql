-- Retour arrière du poste Caisse.
-- Les contraintes CHECK sont validées sur les lignes existantes : il faut donc
-- ramener les montures et mouvements sur les valeurs d'origine AVANT de restaurer
-- les listes, sinon l'ALTER échoue. Une monture en caisse redevient exposée.
UPDATE glasses SET status = 'EN_PRESENTOIR' WHERE status = 'EN_CAISSE';
UPDATE movements SET action = 'PRESENTOIR' WHERE action = 'MISE_EN_CAISSE';

ALTER TABLE glasses DROP CONSTRAINT IF EXISTS glasses_status_check;
ALTER TABLE glasses ADD CONSTRAINT glasses_status_check CHECK (status IN (
    'RECU_FOURNISSEUR', 'EN_STOCK_GENERAL', 'EN_TRANSIT',
    'EN_STOCK_SOUS_STATION', 'EN_PRESENTOIR', 'RESERVEE',
    'EN_LABORATOIRE', 'PRETE_A_LIVRER', 'VENDUE',
    'PERDUE', 'CASSEE', 'RETOURNEE'
));

ALTER TABLE movements DROP CONSTRAINT IF EXISTS movements_action_check;
ALTER TABLE movements ADD CONSTRAINT movements_action_check CHECK (action IN (
    'RECEPTION_FOURNISSEUR', 'RANGEMENT', 'EXPEDITION', 'RECEPTION_STATION',
    'PRESENTOIR', 'RETRAIT_PRESENTOIR', 'RESERVATION', 'LABORATOIRE',
    'CONTROLE_QUALITE', 'LIVRAISON', 'RETOUR', 'INVENTAIRE', 'PERTE', 'CASSE'
));

-- Station et rôle en dernier : les FK sont en ON DELETE RESTRICT, la suppression
-- échouerait tant qu'un utilisateur ou un mouvement y renvoie. On ne force rien —
-- un poste encore utilisé doit rester debout plutôt que d'emporter ses données.
DELETE FROM roles WHERE name = 'CAISSIER'
  AND NOT EXISTS (SELECT 1 FROM users u JOIN roles r ON r.id = u.role_id WHERE r.name = 'CAISSIER');
DELETE FROM stations WHERE name = 'Caisse'
  AND NOT EXISTS (SELECT 1 FROM glasses g WHERE g.station_id = stations.id)
  AND NOT EXISTS (SELECT 1 FROM users u WHERE u.station_id = stations.id);
