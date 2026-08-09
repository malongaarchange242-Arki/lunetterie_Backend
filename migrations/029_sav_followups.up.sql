-- ============================================
-- SUIVI CLIENT DU POSTE SAV
--
-- Pas de table « clients » ici, volontairement : le client, sa commande et son paiement
-- vivent déjà dans proformas (client_name, client_phone, status, settled_at), et l'état
-- de sa monture dans glasses.status. Les dupliquer créerait deux vérités.
--
-- Ce que le SAV possède en propre, c'est la relation : a-t-on appelé, est-ce que ça a
-- répondu, qu'a-t-on noté, quand rappeler. Une ligne par proforma suivie, créée à la
-- première action du SAV — un client jamais appelé n'a simplement pas de ligne.
-- ============================================

CREATE TABLE IF NOT EXISTS sav_followups (
    id BIGSERIAL PRIMARY KEY,
    proforma_id BIGINT NOT NULL UNIQUE REFERENCES proformas(id) ON DELETE CASCADE,
    called BOOLEAN NOT NULL DEFAULT FALSE,
    called_at TIMESTAMPTZ NULL,
    -- Distinct de called : on a composé le numéro, personne n'a décroché. C'est ce qui
    -- sépare « à rappeler » de « pas encore tenté ».
    no_answer BOOLEAN NOT NULL DEFAULT FALSE,
    relance_at DATE NULL,
    observations TEXT NULL,
    message TEXT NULL,
    updated_by BIGINT NULL REFERENCES users(id) ON DELETE SET NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Le SAV ouvre son écran sur « qui reste à rappeler aujourd'hui » : sans cet index,
-- chaque chargement balaierait une table qui ne fait que grossir.
CREATE INDEX IF NOT EXISTS idx_sav_followups_relance
    ON sav_followups(relance_at)
    WHERE relance_at IS NOT NULL;

CREATE INDEX IF NOT EXISTS idx_sav_followups_called ON sav_followups(called);
