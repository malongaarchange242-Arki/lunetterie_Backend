-- Proforma émise au Présentoir quand un client choisit une monture, puis arbitrée à la Caisse.
--
-- Les attributs de la monture sont recopiés dans proforma_items plutôt que seulement
-- référencés : le document doit rester lisible même si la monture est ensuite vendue,
-- déplacée ou supprimée. Même choix que send_list_items.
CREATE TABLE IF NOT EXISTS proformas (
    id BIGSERIAL PRIMARY KEY,
    code VARCHAR(40) NOT NULL UNIQUE,
    station_id BIGINT NOT NULL REFERENCES stations(id),
    client_name VARCHAR(160) NOT NULL,
    client_phone VARCHAR(40) NULL,
    total_amount NUMERIC(12,2) NOT NULL DEFAULT 0,
    -- EN_ATTENTE : créée, montures bloquées, la Caisse doit trancher.
    -- REGLEE     : encaissée (les montures partent au Laboratoire via la vente).
    -- ANNULEE    : le client a renoncé, les montures retournent au Présentoir.
    status VARCHAR(30) NOT NULL DEFAULT 'EN_ATTENTE',
    note TEXT NULL,
    created_by BIGINT NULL,
    settled_by BIGINT NULL,
    settled_at TIMESTAMPTZ NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS proforma_items (
    id BIGSERIAL PRIMARY KEY,
    proforma_id BIGINT NOT NULL REFERENCES proformas(id) ON DELETE CASCADE,
    glass_id BIGINT NULL REFERENCES glasses(id) ON DELETE SET NULL,
    barcode VARCHAR(80) NULL,
    reference VARCHAR(120) NULL,
    brand VARCHAR(120) NULL,
    unit_price NUMERIC(12,2) NOT NULL DEFAULT 0,
    -- La Caisse tranche ligne par ligne : un client peut garder une paire et renoncer à
    -- l'autre. La décision vit donc sur l'item, pas sur l'en-tête.
    --   NULL              : pas encore arbitrée
    --   VENDUE            : encaissée, la monture part au Laboratoire
    --   RETOUR_PRESENTOIR : le client a renoncé, la monture retourne en rayon
    outcome VARCHAR(30) NULL,
    settled_at TIMESTAMPTZ NULL,
    -- Vrai tant que la ligne n'est pas arbitrée. Redondant avec outcome IS NULL en apparence,
    -- mais c'est ce qui rend le blocage vérifiable par un index : le WHERE d'un index partiel
    -- PostgreSQL n'accepte pas de sous-requête, on ne peut donc pas interroger
    -- proformas.status depuis ici.
    is_pending BOOLEAN NOT NULL DEFAULT TRUE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- La Caisse interroge en permanence « quelles proformas attendent une décision » :
-- l'index évite un balayage complet sur une table qui ne fait que grossir.
CREATE INDEX IF NOT EXISTS idx_proformas_status ON proformas(status, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_proformas_station ON proformas(station_id);
CREATE INDEX IF NOT EXISTS idx_proforma_items_proforma ON proforma_items(proforma_id);

-- Une monture ne peut être engagée que sur une seule proforma en attente à la fois.
-- Posé en base et pas seulement dans le code : c'est ce qui rend le blocage réellement
-- fiable si deux vendeurs valident la même monture au même instant.
CREATE UNIQUE INDEX IF NOT EXISTS idx_proforma_items_glass_pending
    ON proforma_items(glass_id)
    WHERE glass_id IS NOT NULL AND is_pending;
