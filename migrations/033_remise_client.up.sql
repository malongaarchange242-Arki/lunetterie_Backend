-- ============================================
-- REMISE AU CLIENT
-- Le cycle s'arrêtait à PRETE_A_LIVRER : le laboratoire annonçait qu'une monture était
-- montée, et plus rien n'était enregistré ensuite. Le client repartait avec ses lunettes
-- sans que la base le sache — une monture livrée restait comptée en stock indéfiniment,
-- et le pointage de la remise ne vivait que dans le navigateur du poste.
-- ============================================

-- 1. Le statut terminal. Même forme que la 025 : la contrainte est déclarée en ligne dans
-- 001_init, donc nommée glasses_status_check ; on la remplace en conservant la liste.
ALTER TABLE glasses DROP CONSTRAINT IF EXISTS glasses_status_check;
ALTER TABLE glasses ADD CONSTRAINT glasses_status_check CHECK (status IN (
    'RECU_FOURNISSEUR', 'EN_STOCK_GENERAL', 'EN_TRANSIT',
    'EN_STOCK_SOUS_STATION', 'EN_PRESENTOIR', 'EN_CAISSE', 'RESERVEE',
    'EN_LABORATOIRE', 'PRETE_A_LIVRER', 'LIVREE', 'VENDUE',
    'PERDUE', 'CASSEE', 'RETOURNEE'
));

-- 2. L'action de mouvement.
--
-- REMISE_CLIENT et non LIVRAISON : cette dernière est déjà écrite par le laboratoire quand
-- il termine un montage (delivery_service.go CreateDelivery). Réutiliser le même libellé
-- confondrait « le labo a fini » et « le client est reparti avec » — c'est déjà ce qui fait
-- que la tuile « Remises aujourd'hui » du poste R. Magasin compte des fins de montage.
ALTER TABLE movements DROP CONSTRAINT IF EXISTS movements_action_check;
ALTER TABLE movements ADD CONSTRAINT movements_action_check CHECK (action IN (
    'RECEPTION_FOURNISSEUR', 'RANGEMENT', 'EXPEDITION', 'RECEPTION_STATION',
    'PRESENTOIR', 'MISE_EN_CAISSE', 'RETRAIT_PRESENTOIR', 'RESERVATION', 'LABORATOIRE',
    'CONTROLE_QUALITE', 'LIVRAISON', 'REMISE_CLIENT', 'RETOUR', 'INVENTAIRE',
    'PERTE', 'CASSE'
));

-- 3. Quand la remise a eu lieu, sur la ligne de livraison.
--
-- Les tables deliveries / delivery_items existaient déjà mais ne servaient qu'à l'étape du
-- laboratoire. La colonne distingue les deux moments sur la même ligne : créée quand le
-- montage se termine, horodatée quand le client repart.
ALTER TABLE delivery_items ADD COLUMN IF NOT EXISTS handed_over_at TIMESTAMPTZ NULL;

-- « Qu'est-ce qui attend encore son client ? » : la question que pose le poste Vendeuse à
-- chaque ouverture.
CREATE INDEX IF NOT EXISTS idx_delivery_items_pending
    ON delivery_items(handed_over_at) WHERE handed_over_at IS NULL;
