-- Relie les magasins aux référentiels géographiques utilisés par Suivi Global.
CREATE TABLE IF NOT EXISTS pays (
    id BIGSERIAL PRIMARY KEY,
    nom VARCHAR(100) NOT NULL UNIQUE,
    code VARCHAR(10) UNIQUE,
    created_at TIMESTAMP DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS villes (
    id BIGSERIAL PRIMARY KEY,
    nom VARCHAR(100) NOT NULL,
    pays_id BIGINT NOT NULL REFERENCES pays(id) ON DELETE CASCADE,
    created_at TIMESTAMP DEFAULT NOW(),
    UNIQUE (nom, pays_id)
);

ALTER TABLE stations
    ADD COLUMN IF NOT EXISTS pays_id BIGINT REFERENCES pays(id) ON DELETE SET NULL,
    ADD COLUMN IF NOT EXISTS ville_id BIGINT REFERENCES villes(id) ON DELETE SET NULL;

CREATE INDEX IF NOT EXISTS idx_stations_pays_id ON stations(pays_id);
CREATE INDEX IF NOT EXISTS idx_stations_ville_id ON stations(ville_id);
