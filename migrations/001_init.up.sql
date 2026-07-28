-- ============================================
-- MIGRATION LUNETTERIE - 001_init.up.sql
-- ============================================
mot de base de la base de données (3C1JtEZ1xdeT9QNl)
uvicorn app.api.main:app --host 0.0.0.0 --port 8000 --reload
-- Extensions
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

-- ============================================
-- AUTHENTIFICATION
-- ============================================

-- Rôles
CREATE TABLE IF NOT EXISTS roles (
    id BIGSERIAL PRIMARY KEY,
    name VARCHAR(50) NOT NULL UNIQUE,
    description TEXT,
    created_at TIMESTAMP DEFAULT NOW()
);

INSERT INTO roles (name, description) VALUES
    ('SUPER_ADMIN', 'Super administrateur - tous les droits'),
    ('ADMIN', 'Administrateur'),
    ('MAGASINIER', 'Gestionnaire de stock'),
    ('VENDEUR', 'Vendeur en magasin'),
    ('LABORATOIRE', 'Technicien de laboratoire'),
    ('RESPONSABLE_STATION', 'Responsable de station')
ON CONFLICT DO NOTHING;

-- ============================================
-- STATIONS & EMPLACEMENTS
-- ============================================

-- Stations
CREATE TABLE IF NOT EXISTS stations (
    id BIGSERIAL PRIMARY KEY,
    name VARCHAR(100) NOT NULL,
    type VARCHAR(50) NOT NULL CHECK (type IN ('STOCK_GENERAL', 'SOUS_STATION')),
    city VARCHAR(100),
    address TEXT,
    phone VARCHAR(20),
    is_active BOOLEAN DEFAULT true,
    created_at TIMESTAMP DEFAULT NOW()
);

-- Emplacements de stockage
CREATE TABLE IF NOT EXISTS storage_locations (
    id BIGSERIAL PRIMARY KEY,
    station_id BIGINT NOT NULL REFERENCES stations(id) ON DELETE CASCADE,
    parent_location_id BIGINT REFERENCES storage_locations(id) ON DELETE SET NULL,
    zone VARCHAR(50) NOT NULL CHECK (zone IN ('STOCK', 'PRESENTOIR', 'LABORATOIRE')),
    code VARCHAR(50) NOT NULL,
    type VARCHAR(50) NOT NULL CHECK (type IN ('RAYON', 'ETAGERE', 'BAC', 'POSITION', 'PRESENTOIR', 'ARMOIRE', 'SALLE')),
    status VARCHAR(20) DEFAULT 'LIBRE' CHECK (status IN ('LIBRE', 'OCCUPE', 'MAINTENANCE')),
    created_at TIMESTAMP DEFAULT NOW(),
    UNIQUE(station_id, code)
);

-- ============================================
-- UTILISATEURS
-- ============================================

CREATE TABLE IF NOT EXISTS users (
    id BIGSERIAL PRIMARY KEY,
    first_name VARCHAR(100) NOT NULL,
    last_name VARCHAR(100) NOT NULL,
    email VARCHAR(255) NOT NULL UNIQUE,
    password_hash_deprecated VARCHAR(255),
    fingerprint_hash VARCHAR(255) UNIQUE,
    phone VARCHAR(20),
    role_id BIGINT NOT NULL REFERENCES roles(id),
    station_id BIGINT REFERENCES stations(id) ON DELETE SET NULL,
    is_active BOOLEAN DEFAULT true,
    last_login TIMESTAMP,
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS biometric_devices (
    id BIGSERIAL PRIMARY KEY,
    device_id VARCHAR(255) NOT NULL UNIQUE,
    device_name VARCHAR(100),
    station_id BIGINT REFERENCES stations(id),
    is_active BOOLEAN DEFAULT true,
    last_seen TIMESTAMP,
    created_at TIMESTAMP DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS biometric_sessions (
    id BIGSERIAL PRIMARY KEY,
    user_id BIGINT NOT NULL REFERENCES users(id),
    device_id BIGINT NOT NULL REFERENCES biometric_devices(id),
    fingerprint_hash VARCHAR(255) NOT NULL,
    ip_address VARCHAR(45),
    expires_at TIMESTAMP NOT NULL,
    created_at TIMESTAMP DEFAULT NOW()
);

-- ============================================
-- CATALOGUE
-- ============================================

-- Marques
CREATE TABLE IF NOT EXISTS brands (
    id BIGSERIAL PRIMARY KEY,
    name VARCHAR(100) NOT NULL UNIQUE,
    created_at TIMESTAMP DEFAULT NOW()
);

-- Formes
CREATE TABLE IF NOT EXISTS shapes (
    id BIGSERIAL PRIMARY KEY,
    name VARCHAR(50) NOT NULL UNIQUE,
    created_at TIMESTAMP DEFAULT NOW()
);

INSERT INTO shapes (name) VALUES
    ('Rectangle'), ('Rond'), ('Ovale'), ('Carré'), ('Papillon'),
    ('Aviateur'), ('Wayfarer'), ('Cat-eye'), ('Clubmaster'), ('Browline'),
    ('Hexagonal'), ('Pantos'), ('Masque'), ('Papillon oversize')
ON CONFLICT DO NOTHING;

-- Couleurs
CREATE TABLE IF NOT EXISTS colors (
    id BIGSERIAL PRIMARY KEY,
    name VARCHAR(50) NOT NULL UNIQUE,
    hex_code VARCHAR(7),
    created_at TIMESTAMP DEFAULT NOW()
);

INSERT INTO colors (name, hex_code) VALUES
    ('Noir', '#000000'),
    ('Écaille', '#8B4513'),
    ('Marron', '#654321'),
    ('Bleu', '#0000FF'),
    ('Rouge', '#FF0000'),
    ('Vert', '#008000'),
    ('Blanc', '#FFFFFF'),
    ('Gris', '#808080')
ON CONFLICT DO NOTHING;

-- Matériaux
CREATE TABLE IF NOT EXISTS materials (
    id BIGSERIAL PRIMARY KEY,
    name VARCHAR(50) NOT NULL UNIQUE,
    created_at TIMESTAMP DEFAULT NOW()
);

INSERT INTO materials (name) VALUES
    ('Acétate'), ('Métal'), ('Titane'), ('Plastique'), ('Bois'),
    ('Corne'), ('Aluminium'), ('Acier inoxydable'), ('TR90'), ('Ultem')
ON CONFLICT DO NOTHING;

-- Types de monture
CREATE TABLE IF NOT EXISTS mount_types (
    id BIGSERIAL PRIMARY KEY,
    name VARCHAR(50) NOT NULL UNIQUE,
    created_at TIMESTAMP DEFAULT NOW()
);

INSERT INTO mount_types (name) VALUES
    ('Pleine monture'), ('Demi-cerclée'), ('Cerclée'), ('Perforée'), ('Clip-on')
ON CONFLICT DO NOTHING;

-- Modèles de montures
CREATE TABLE IF NOT EXISTS frame_models (
    id BIGSERIAL PRIMARY KEY,
    brand_id BIGINT REFERENCES brands(id) ON DELETE SET NULL,
    reference VARCHAR(100),
    shape_id BIGINT REFERENCES shapes(id) ON DELETE SET NULL,
    color_id BIGINT REFERENCES colors(id) ON DELETE SET NULL,
    material_id BIGINT REFERENCES materials(id) ON DELETE SET NULL,
    mount_type_id BIGINT REFERENCES mount_types(id) ON DELETE SET NULL,
    gender VARCHAR(20) CHECK (gender IN ('HOMME', 'FEMME', 'ENFANT', 'UNISEXE')),
    size VARCHAR(20),
    image_url TEXT,
    is_active BOOLEAN DEFAULT true,
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW(),
    UNIQUE(brand_id, reference)
);

-- ============================================
-- FOURNISSEURS & ARRIVAGES
-- ============================================

CREATE TABLE IF NOT EXISTS suppliers (
    id BIGSERIAL PRIMARY KEY,
    name VARCHAR(200) NOT NULL,
    phone VARCHAR(20),
    email VARCHAR(255),
    address TEXT,
    contact_name VARCHAR(200),
    is_active BOOLEAN DEFAULT true,
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS deliveries (
    id BIGSERIAL PRIMARY KEY,
    supplier_id BIGINT NOT NULL REFERENCES suppliers(id) ON DELETE RESTRICT,
    reference VARCHAR(100),
    received_by BIGINT REFERENCES users(id) ON DELETE SET NULL,
    station_id BIGINT NOT NULL REFERENCES stations(id) ON DELETE RESTRICT,
    notes TEXT,
    received_at TIMESTAMP DEFAULT NOW(),
    created_at TIMESTAMP DEFAULT NOW()
);

-- ============================================
-- ANALYSES IA
-- ============================================

CREATE TABLE IF NOT EXISTS glass_analysis (
    id BIGSERIAL PRIMARY KEY,
    glass_id BIGINT NOT NULL,
    model_version VARCHAR(20) NOT NULL,
    shape VARCHAR(50),
    shape_confidence DECIMAL(5,2),
    color VARCHAR(50),
    color_confidence DECIMAL(5,2),
    material VARCHAR(50),
    material_confidence DECIMAL(5,2),
    mount_type VARCHAR(50),
    mount_type_confidence DECIMAL(5,2),
    gender VARCHAR(20),
    gender_confidence DECIMAL(5,2),
    brand VARCHAR(100),
    brand_confidence DECIMAL(5,2),
    reference VARCHAR(100),
    reference_confidence DECIMAL(5,2),
    size VARCHAR(20),
    crop_image_path TEXT,
    processing_time_ms DECIMAL(10,2),
    created_at TIMESTAMP DEFAULT NOW()
);

-- ============================================
-- MONTURES PHYSIQUES
-- ============================================

CREATE TABLE IF NOT EXISTS glasses (
    id BIGSERIAL PRIMARY KEY,
    barcode VARCHAR(128) NOT NULL UNIQUE,
    serial_number VARCHAR(100),
    frame_model_id BIGINT REFERENCES frame_models(id) ON DELETE SET NULL,
    station_id BIGINT NOT NULL REFERENCES stations(id) ON DELETE RESTRICT,
    location_id BIGINT REFERENCES storage_locations(id) ON DELETE SET NULL,
    supplier_id BIGINT REFERENCES suppliers(id) ON DELETE SET NULL,
    delivery_id BIGINT REFERENCES deliveries(id) ON DELETE SET NULL,
    analysis_id BIGINT REFERENCES glass_analysis(id) ON DELETE SET NULL,
    status VARCHAR(50) NOT NULL DEFAULT 'RECU_FOURNISSEUR' 
        CHECK (status IN (
            'RECU_FOURNISSEUR', 'EN_STOCK_GENERAL', 'EN_TRANSIT', 
            'EN_STOCK_SOUS_STATION', 'EN_PRESENTOIR', 'RESERVEE', 
            'EN_LABORATOIRE', 'PRETE_A_LIVRER', 'VENDUE', 
            'PERDUE', 'CASSEE', 'RETOURNEE'
        )),
    is_reserved BOOLEAN DEFAULT FALSE,
    reserved_for_order BIGINT,
    notes TEXT,
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW()
);

-- Ajouter la contrainte FK pour glass_analysis
ALTER TABLE glass_analysis 
ADD CONSTRAINT fk_glass_analysis_glass 
FOREIGN KEY (glass_id) REFERENCES glasses(id) ON DELETE CASCADE
ON CONFLICT DO NOTHING;

-- Index sur glasses
CREATE INDEX IF NOT EXISTS idx_glasses_barcode ON glasses(barcode);
CREATE INDEX IF NOT EXISTS idx_glasses_status ON glasses(status);
CREATE INDEX IF NOT EXISTS idx_glasses_station ON glasses(station_id);
CREATE INDEX IF NOT EXISTS idx_glasses_location ON glasses(location_id);
CREATE INDEX IF NOT EXISTS idx_glasses_created ON glasses(created_at);

-- ============================================
-- MOUVEMENTS
-- ============================================

CREATE TABLE IF NOT EXISTS movements (
    id BIGSERIAL PRIMARY KEY,
    glass_id BIGINT NOT NULL REFERENCES glasses(id) ON DELETE CASCADE,
    from_station_id BIGINT REFERENCES stations(id) ON DELETE SET NULL,
    to_station_id BIGINT REFERENCES stations(id) ON DELETE SET NULL,
    from_location_id BIGINT REFERENCES storage_locations(id) ON DELETE SET NULL,
    to_location_id BIGINT REFERENCES storage_locations(id) ON DELETE SET NULL,
    action VARCHAR(50) NOT NULL CHECK (action IN (
        'RECEPTION_FOURNISSEUR', 'RANGEMENT', 'EXPEDITION', 'RECEPTION_STATION',
        'PRESENTOIR', 'RETRAIT_PRESENTOIR', 'RESERVATION', 'LABORATOIRE',
        'CONTROLE_QUALITE', 'LIVRAISON', 'RETOUR', 'INVENTAIRE', 'PERTE', 'CASSE'
    )),
    user_id BIGINT NOT NULL REFERENCES users(id) ON DELETE RESTRICT,
    notes TEXT,
    created_at TIMESTAMP DEFAULT NOW()
);

-- Index sur movements
CREATE INDEX IF NOT EXISTS idx_movements_glass ON movements(glass_id);
CREATE INDEX IF NOT EXISTS idx_movements_action ON movements(action);
CREATE INDEX IF NOT EXISTS idx_movements_created ON movements(created_at);

-- ============================================
-- FIN DE LA MIGRATION
-- ============================================




-- Fonction : générer les emplacements d'une station
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

-- Fonction : trouver un emplacement libre
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

-- Vue : arborescence complète
CREATE OR REPLACE VIEW v_location_tree AS
WITH RECURSIVE location_tree AS (
    SELECT 
        id, station_id, parent_location_id, zone, code, name, type, status,
        code AS full_path,
        COALESCE(name, code) AS full_name,
        0 AS level
    FROM storage_locations
    WHERE parent_location_id IS NULL
    
    UNION ALL
    
    SELECT 
        sl.id, sl.station_id, sl.parent_location_id, sl.zone, sl.code, sl.name, sl.type, sl.status,
        lt.full_path || ' → ' || sl.code,
        lt.full_name || ' → ' || COALESCE(sl.name, sl.code),
        lt.level + 1
    FROM storage_locations sl
    INNER JOIN location_tree lt ON sl.parent_location_id = lt.id
)
SELECT * FROM location_tree;