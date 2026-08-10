-- Réception réelle d'un carton : l'expédition ne vaut plus arrivée.
--
-- Jusqu'ici, envoyer une liste faisait passer les montures EN_STOCK_SOUS_STATION dans la
-- station destinataire au moment même du départ. Un carton perdu en route laissait donc le
-- stock du magasin crédité de montures qu'il n'avait jamais reçues.
--
-- L'expédition crée désormais un vrai transfert (montures EN_TRANSIT), et c'est le pointage
-- du carton à l'arrivée qui fait entrer chaque monture en stock. Le carton doit donc savoir
-- quel transfert il transporte : c'est lui qui porte l'état de réception, monture par
-- monture, dans transfer_items.
ALTER TABLE send_boxes
    ADD COLUMN IF NOT EXISTS transfer_id BIGINT NULL REFERENCES transfers(id) ON DELETE SET NULL;

-- Clôture : un carton peut être clôturé avec des manquantes (réception partielle). Les
-- montures jamais pointées restent EN_TRANSIT — elles n'entrent pas au stock — et leur ligne
-- de transfert reste ouverte, si bien qu'un scan ultérieur les recevra encore.
-- `missing_count` fige ce qui manquait au moment de la clôture, pour le litige transporteur.
ALTER TABLE send_boxes
    ADD COLUMN IF NOT EXISTS closed_at TIMESTAMPTZ NULL,
    ADD COLUMN IF NOT EXISTS closed_by BIGINT NULL,
    ADD COLUMN IF NOT EXISTS missing_count INTEGER NOT NULL DEFAULT 0;

CREATE INDEX IF NOT EXISTS idx_send_boxes_transfer ON send_boxes(transfer_id);
