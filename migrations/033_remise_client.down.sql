-- Les montures déjà remises redeviennent PRETE_A_LIVRER : c'est l'état d'où elles
-- viennent, et la contrainte rétablie ci-dessous refuserait LIVREE. Leur mouvement de
-- remise est supprimé pour la même raison.
UPDATE glasses SET status = 'PRETE_A_LIVRER' WHERE status = 'LIVREE';
DELETE FROM movements WHERE action = 'REMISE_CLIENT';

DROP INDEX IF EXISTS idx_delivery_items_pending;
ALTER TABLE delivery_items DROP COLUMN IF EXISTS handed_over_at;

ALTER TABLE glasses DROP CONSTRAINT IF EXISTS glasses_status_check;
ALTER TABLE glasses ADD CONSTRAINT glasses_status_check CHECK (status IN (
    'RECU_FOURNISSEUR', 'EN_STOCK_GENERAL', 'EN_TRANSIT',
    'EN_STOCK_SOUS_STATION', 'EN_PRESENTOIR', 'EN_CAISSE', 'RESERVEE',
    'EN_LABORATOIRE', 'PRETE_A_LIVRER', 'VENDUE',
    'PERDUE', 'CASSEE', 'RETOURNEE'
));

ALTER TABLE movements DROP CONSTRAINT IF EXISTS movements_action_check;
ALTER TABLE movements ADD CONSTRAINT movements_action_check CHECK (action IN (
    'RECEPTION_FOURNISSEUR', 'RANGEMENT', 'EXPEDITION', 'RECEPTION_STATION',
    'PRESENTOIR', 'MISE_EN_CAISSE', 'RETRAIT_PRESENTOIR', 'RESERVATION', 'LABORATOIRE',
    'CONTROLE_QUALITE', 'LIVRAISON', 'RETOUR', 'INVENTAIRE', 'PERTE', 'CASSE'
));
