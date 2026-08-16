-- Les montures encore réservées redeviennent EN_STOCK_GENERAL : c'est l'état d'où elles
-- viennent, et la contrainte rétablie ci-dessous refuserait RESERVEE_ENVOI.
UPDATE glasses SET status = 'EN_STOCK_GENERAL' WHERE status = 'RESERVEE_ENVOI';

ALTER TABLE glasses DROP CONSTRAINT IF EXISTS glasses_status_check;
ALTER TABLE glasses ADD CONSTRAINT glasses_status_check CHECK (status IN (
    'RECU_FOURNISSEUR', 'EN_STOCK_GENERAL', 'EN_TRANSIT',
    'EN_STOCK_SOUS_STATION', 'EN_PRESENTOIR', 'EN_CAISSE', 'RESERVEE',
    'EN_LABORATOIRE', 'PRETE_A_LIVRER', 'LIVREE', 'VENDUE',
    'PERDUE', 'CASSEE', 'RETOURNEE'
));
