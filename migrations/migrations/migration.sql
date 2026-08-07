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
    created_at TIMESTAMP DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS stations_principales (
    id BIGSERIAL PRIMARY KEY,
    nom VARCHAR(150) NOT NULL,
    ville_id BIGINT NOT NULL REFERENCES villes(id) ON DELETE CASCADE,
    created_at TIMESTAMP DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS stations_locales (
    id BIGSERIAL PRIMARY KEY,
    nom VARCHAR(150) NOT NULL,
    station_principale_id BIGINT NOT NULL REFERENCES stations_principales(id) ON DELETE CASCADE,
    ville_id BIGINT NOT NULL REFERENCES villes(id) ON DELETE CASCADE,
    created_at TIMESTAMP DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS espaces (
    id BIGSERIAL PRIMARY KEY,
    nom VARCHAR(100) NOT NULL,
    type VARCHAR(50) NOT NULL CHECK (type IN ('PRESENTOIR', 'RESERVE', 'LABORATOIRE')),
    station_locale_id BIGINT NOT NULL REFERENCES stations_locales(id) ON DELETE CASCADE,
    created_at TIMESTAMP DEFAULT NOW(),
    UNIQUE(station_locale_id, nom)
);


ALTER TABLE stations
ADD COLUMN IF NOT EXISTS pays_id BIGINT REFERENCES pays(id) ON DELETE SET NULL;

ALTER TABLE stations
ADD COLUMN IF NOT EXISTS ville_id BIGINT REFERENCES villes(id) ON DELETE SET NULL;

ALTER TABLE stations
ADD COLUMN IF NOT EXISTS station_principale_id BIGINT REFERENCES stations_principales(id) ON DELETE SET NULL;

ALTER TABLE stations
ADD COLUMN IF NOT EXISTS station_locale_id BIGINT REFERENCES stations_locales(id) ON DELETE SET NULL;


INSERT INTO pays (nom, code)
VALUES ('Congo', 'CG')
ON CONFLICT (nom) DO NOTHING;

INSERT INTO villes (nom, pays_id)
VALUES ('Pointe-Noire', 1)
ON CONFLICT DO NOTHING;

INSERT INTO stations_principales (nom, ville_id)
VALUES ('Station principale Pointe-Noire', 1)
ON CONFLICT DO NOTHING;

INSERT INTO stations_locales (nom, station_principale_id, ville_id)
VALUES ('Station locale Pointe-Noire', 1, 1)
ON CONFLICT DO NOTHING;

INSERT INTO espaces (nom, type, station_locale_id)
VALUES
    ('Présentoir A', 'PRESENTOIR', 1),
    ('Réserve A', 'RESERVE', 1),
    ('Laboratoire A', 'LABORATOIRE', 1)
ON CONFLICT DO NOTHING;


-- Pays
INSERT INTO pays (nom, code)
VALUES ('République démocratique du Congo', 'CD')
ON CONFLICT (nom) DO NOTHING;

-- Ville
INSERT INTO villes (nom, pays_id)
VALUES (
    'Kinshasa',
    (SELECT id FROM pays WHERE code = 'CD')
)
ON CONFLICT DO NOTHING;

-- Station principale
INSERT INTO stations_principales (nom, ville_id)
VALUES (
    'Station principale Kinshasa',
    (SELECT id FROM villes WHERE nom = 'Kinshasa'
        AND pays_id = (SELECT id FROM pays WHERE code = 'CD'))
)
ON CONFLICT DO NOTHING;

-- Station locale
INSERT INTO stations_locales (nom, station_principale_id, ville_id)
VALUES (
    'Station locale Kinshasa',
    (SELECT id FROM stations_principales
        WHERE nom = 'Station principale Kinshasa'),
    (SELECT id FROM villes WHERE nom = 'Kinshasa'
        AND pays_id = (SELECT id FROM pays WHERE code = 'CD'))
)
ON CONFLICT DO NOTHING;

-- Espaces
INSERT INTO espaces (nom, type, station_locale_id)
VALUES
(
    'Présentoir Kinshasa A',
    'PRESENTOIR',
    (SELECT id FROM stations_locales
        WHERE nom = 'Station locale Kinshasa')
),
(
    'Réserve Kinshasa A',
    'RESERVE',
    (SELECT id FROM stations_locales
        WHERE nom = 'Station locale Kinshasa')
),
(
    'Laboratoire Kinshasa A',
    'LABORATOIRE',
    (SELECT id FROM stations_locales
        WHERE nom = 'Station locale Kinshasa')
)
ON CONFLICT DO NOTHING;