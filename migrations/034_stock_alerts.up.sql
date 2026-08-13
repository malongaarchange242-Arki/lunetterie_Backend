-- Alerte de stock à 10 %, pour le présentoir et pour le stock local d'un magasin.
--
-- Trois besoins, aucun ne trouvait de support en base :
--   1. une quantité de référence par station, contre laquelle mesurer les 10 % ;
--   2. un état d'alerte, sans lequel chaque mutation sous le seuil renotifierait ;
--   3. une table de notifications — il n'en existait aucune dans le projet.

-- ── 1. Le stock de référence ───────────────────────────────────────────────────
-- Il n'existait nulle part : ni capacité, ni quota, ni cible. RestockSuggestions
-- (send_list_repository.go) prend pour référence le dernier carton reçu, mais c'est une
-- mesure par ville, propre au réapprovisionnement, et aucun carton ne vise le présentoir.
ALTER TABLE stations
    ADD COLUMN IF NOT EXISTS presentoir_reference_qty INT NULL,
    ADD COLUMN IF NOT EXISTS local_reference_qty      INT NULL;

COMMENT ON COLUMN stations.presentoir_reference_qty IS
    'Quantité normale du présentoir. NULL = aucune référence, aucune alerte émise.';
COMMENT ON COLUMN stations.local_reference_qty IS
    'Quantité normale du stock local. NULL = aucune référence, aucune alerte émise.';

-- Amorçage : la référence part du stock constaté aujourd'hui, tenu pour la normale de
-- chaque magasin. Sans lui, toutes les colonnes resteraient NULL et aucune alerte ne
-- partirait jamais tant qu'un administrateur ne les aurait pas saisies une par une.
-- NULLIF ramène un comptage à zéro sur NULL : une station vide n'a pas de référence à 0,
-- elle n'en a pas du tout — sinon son seuil vaudrait 0 et l'alerte serait injustifiable.
UPDATE stations s SET
    presentoir_reference_qty = NULLIF((
        SELECT COUNT(*) FROM glasses g
        WHERE g.station_id = s.id AND g.status = 'EN_PRESENTOIR'
    ), 0),
    local_reference_qty = NULLIF((
        SELECT COUNT(*) FROM glasses g
        WHERE g.station_id = s.id AND g.status = 'EN_STOCK_SOUS_STATION'
    ), 0);

-- ── 2. L'état d'alerte ─────────────────────────────────────────────────────────
-- Une ligne par (station, type de stock). C'est elle qui distingue « le stock vient de
-- passer sous le seuil » de « le stock est sous le seuil depuis hier » : sans cet état,
-- 10 → 9 → 8 → 7 émettrait quatre notifications au lieu d'une.
--
-- La contrainte UNIQUE porte le mécanisme anti-doublon : l'armement est un INSERT
-- ... ON CONFLICT DO UPDATE ... WHERE active = false. Deux mutations concurrentes sur la
-- même station se sérialisent sur la ligne, et la seconde ne voit plus active = false.
CREATE TABLE IF NOT EXISTS stock_alert_states (
    id BIGSERIAL PRIMARY KEY,
    station_id BIGINT NOT NULL REFERENCES stations(id) ON DELETE CASCADE,
    stock_type VARCHAR(20) NOT NULL CHECK (stock_type IN ('PRESENTOIR', 'LOCAL')),
    active BOOLEAN NOT NULL DEFAULT false,
    triggered_at TIMESTAMPTZ NULL,
    cleared_at TIMESTAMPTZ NULL,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (station_id, stock_type)
);

-- ── 3. Les notifications ───────────────────────────────────────────────────────
-- Une ligne par destinataire, et non une notification partagée : c'est ce qui permet à
-- chacun de marquer la sienne comme lue sans toucher celle des autres.
CREATE TABLE IF NOT EXISTS notifications (
    id BIGSERIAL PRIMARY KEY,
    user_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    type VARCHAR(40) NOT NULL,
    message TEXT NOT NULL,
    station_id BIGINT NULL REFERENCES stations(id) ON DELETE SET NULL,
    -- Les chiffres qui ont motivé l'alerte, figés au moment où elle part. Les relire plus
    -- tard donnerait le stock d'aujourd'hui, pas celui qui a déclenché la notification.
    stock_type VARCHAR(20) NULL CHECK (stock_type IS NULL OR stock_type IN ('PRESENTOIR', 'LOCAL')),
    current_stock INT NULL,
    reference_stock INT NULL,
    threshold INT NULL,
    read_at TIMESTAMPTZ NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Le poste lit ses non-lues d'abord : l'index couvre `WHERE user_id = ... ORDER BY
-- created_at DESC`, la seule lecture que fait le handler.
CREATE INDEX IF NOT EXISTS idx_notifications_user ON notifications(user_id, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_notifications_unread ON notifications(user_id) WHERE read_at IS NULL;
