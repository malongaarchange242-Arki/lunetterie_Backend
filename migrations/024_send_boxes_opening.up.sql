-- Réception d'un carton au magasin : on trace qui l'a ouvert, quand et à quel poste.
-- Sans ces colonnes, `status` seul ne dirait pas de qui relève un colis réceptionné.
ALTER TABLE send_boxes
    ADD COLUMN IF NOT EXISTS opened_at TIMESTAMPTZ NULL,
    ADD COLUMN IF NOT EXISTS opened_by BIGINT NULL,
    ADD COLUMN IF NOT EXISTS opened_station_id BIGINT NULL;

-- Les cartons en attente sont interrogés à chaque ouverture du poste de magasin :
-- l'index évite un balayage complet sur une table qui ne fait que grossir.
CREATE INDEX IF NOT EXISTS idx_send_boxes_pending
    ON send_boxes(status, city);
