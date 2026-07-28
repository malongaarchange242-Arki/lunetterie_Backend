-- ============================================
-- AMÉLIORATIONS STOCKAGE
-- ============================================

-- 1. Ajouter le champ name à storage_locations
ALTER TABLE storage_locations 
ADD COLUMN IF NOT EXISTS name VARCHAR(100);

-- Mettre à jour les noms existants
UPDATE storage_locations SET name = code WHERE name IS NULL;

-- 2. Table de liaison produits-emplacements
CREATE TABLE IF NOT EXISTS product_locations (
    id BIGSERIAL PRIMARY KEY,
    product_id BIGINT NOT NULL,
    location_id BIGINT NOT NULL REFERENCES storage_locations(id) ON DELETE CASCADE,
    quantity INTEGER NOT NULL DEFAULT 1 CHECK (quantity >= 0),
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW(),
    UNIQUE(product_id, location_id)
);

-- Index
CREATE INDEX IF NOT EXISTS idx_product_locations_product ON product_locations(product_id);
CREATE INDEX IF NOT EXISTS idx_product_locations_location ON product_locations(location_id);

-- 3. Table des templates de génération
CREATE TABLE IF NOT EXISTS station_templates (
    id BIGSERIAL PRIMARY KEY,
    name VARCHAR(100) NOT NULL,
    num_rayons INTEGER NOT NULL DEFAULT 5,
    etageres_par_rayon INTEGER NOT NULL DEFAULT 8,
    bacs_par_etagere INTEGER NOT NULL DEFAULT 4,
    positions_par_bac INTEGER NOT NULL DEFAULT 20,
    created_at TIMESTAMP DEFAULT NOW()
);

-- Template par défaut
INSERT INTO station_templates (name, num_rayons, etageres_par_rayon, bacs_par_etagere, positions_par_bac)
VALUES ('Standard', 5, 8, 4, 20)
ON CONFLICT DO NOTHING;

-- 4. Vue hiérarchique des emplacements (CORRIGÉE)
DROP VIEW IF EXISTS v_location_tree CASCADE;

CREATE OR REPLACE VIEW v_location_tree AS
WITH RECURSIVE location_tree AS (
    -- Racine : emplacements sans parent
    SELECT 
        id,
        station_id,
        parent_location_id,
        zone,
        code,
        name,
        type,
        status,
        code::TEXT AS full_path,
        COALESCE(name, code)::TEXT AS full_name,
        0 AS level
    FROM storage_locations
    WHERE parent_location_id IS NULL
    
    UNION ALL
    
    -- Enfants
    SELECT 
        sl.id,
        sl.station_id,
        sl.parent_location_id,
        sl.zone,
        sl.code,
        sl.name,
        sl.type,
        sl.status,
        (lt.full_path || ' → ' || sl.code)::TEXT,
        (lt.full_name || ' → ' || COALESCE(sl.name, sl.code))::TEXT,
        lt.level + 1
    FROM storage_locations sl
    INNER JOIN location_tree lt ON sl.parent_location_id = lt.id
)
SELECT * FROM location_tree;

-- 5. Vue du stock par emplacement
CREATE OR REPLACE VIEW v_stock_by_location AS
SELECT 
    sl.id AS location_id,
    sl.station_id,
    s.name AS station_name,
    sl.zone,
    sl.code,
    sl.name,
    sl.type,
    sl.status,
    COUNT(pl.product_id) AS product_count,
    COALESCE(SUM(pl.quantity), 0) AS total_quantity
FROM storage_locations sl
JOIN stations s ON sl.station_id = s.id
LEFT JOIN product_locations pl ON sl.id = pl.location_id
GROUP BY sl.id, sl.station_id, s.name, sl.zone, sl.code, sl.name, sl.type, sl.status;

-- 6. Fonction : générer les emplacements d'une station
CREATE OR REPLACE FUNCTION generate_station_locations(
    p_station_id BIGINT,
    p_num_rayons INTEGER DEFAULT 5,
    p_etageres_par_rayon INTEGER DEFAULT 8,
    p_bacs_par_etagere INTEGER DEFAULT 4,
    p_positions_par_bac INTEGER DEFAULT 20
) RETURNS INTEGER AS $$
DECLARE
    rayon_letter CHAR;
    rayon_id BIGINT;
    etagere_num INTEGER;
    etagere_id BIGINT;
    bac_letter CHAR;
    bac_id BIGINT;
    pos_num INTEGER;
    total_locations INTEGER := 0;
BEGIN
    FOR i IN 1..p_num_rayons LOOP
        rayon_letter := CHR(64 + i);
        
        INSERT INTO storage_locations (station_id, parent_location_id, zone, code, name, type, status)
        VALUES (p_station_id, NULL, 'STOCK', 
                'RAYON-' || rayon_letter, 
                'Rayon ' || rayon_letter, 
                'RAYON', 'LIBRE')
        RETURNING id INTO rayon_id;
        
        total_locations := total_locations + 1;
        
        FOR j IN 1..p_etageres_par_rayon LOOP
            etagere_num := j;
            
            INSERT INTO storage_locations (station_id, parent_location_id, zone, code, name, type, status)
            VALUES (p_station_id, rayon_id, 'STOCK',
                    'RAYON-' || rayon_letter || '-ETA-' || LPAD(etagere_num::TEXT, 2, '0'),
                    'Étagère ' || etagere_num,
                    'ETAGERE', 'LIBRE')
            RETURNING id INTO etagere_id;
            
            total_locations := total_locations + 1;
            
            FOR k IN 1..p_bacs_par_etagere LOOP
                bac_letter := CHR(64 + k);
                
                INSERT INTO storage_locations (station_id, parent_location_id, zone, code, name, type, status)
                VALUES (p_station_id, etagere_id, 'STOCK',
                        'RAYON-' || rayon_letter || '-ETA-' || LPAD(etagere_num::TEXT, 2, '0') || '-BAC-' || bac_letter,
                        'Bac ' || bac_letter,
                        'BAC', 'LIBRE')
                RETURNING id INTO bac_id;
                
                total_locations := total_locations + 1;
                
                FOR l IN 1..p_positions_par_bac LOOP
                    pos_num := l;
                    
                    INSERT INTO storage_locations (station_id, parent_location_id, zone, code, name, type, status)
                    VALUES (p_station_id, bac_id, 'STOCK',
                            'RAYON-' || rayon_letter || '-ETA-' || LPAD(etagere_num::TEXT, 2, '0') || '-BAC-' || bac_letter || '-POS-' || LPAD(pos_num::TEXT, 2, '0'),
                            'Position ' || pos_num,
                            'POSITION', 'LIBRE');
                    
                    total_locations := total_locations + 1;
                END LOOP;
            END LOOP;
        END LOOP;
    END LOOP;
    
    RETURN total_locations;
END;
$$ LANGUAGE plpgsql;

-- 7. Fonction : trouver un emplacement libre
CREATE OR REPLACE FUNCTION find_free_location(
    p_station_id BIGINT,
    p_zone VARCHAR DEFAULT 'STOCK'
) RETURNS BIGINT AS $$
DECLARE
    v_location_id BIGINT;
BEGIN
    SELECT sl.id INTO v_location_id
    FROM storage_locations sl
    WHERE sl.station_id = p_station_id
      AND sl.zone = p_zone
      AND sl.type = 'POSITION'
      AND sl.status = 'LIBRE'
    ORDER BY sl.code
    LIMIT 1;
    
    IF v_location_id IS NOT NULL THEN
        UPDATE storage_locations SET status = 'OCCUPE' WHERE id = v_location_id;
    END IF;
    
    RETURN v_location_id;
END;
$$ LANGUAGE plpgsql;

-- 8. Trigger : mise à jour updated_at
CREATE OR REPLACE FUNCTION update_product_locations_updated_at()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

DROP TRIGGER IF EXISTS update_product_locations_updated_at ON product_locations;

CREATE TRIGGER update_product_locations_updated_at
    BEFORE UPDATE ON product_locations
    FOR EACH ROW
    EXECUTE FUNCTION update_product_locations_updated_at();



-- Créer une station de test
INSERT INTO stations (name, type, city, address)
VALUES ('Stock Principal', 'STOCK_GENERAL', 'Brazzaville', 'Centre-ville');