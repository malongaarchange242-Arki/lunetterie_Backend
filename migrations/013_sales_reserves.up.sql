-- ============================================
-- Ajoute les tables de ventes et réservations
-- ============================================

CREATE TABLE IF NOT EXISTS sales (
    id BIGSERIAL PRIMARY KEY,
    station_id BIGINT NOT NULL REFERENCES stations(id) ON DELETE RESTRICT,
    user_id BIGINT NOT NULL REFERENCES users(id) ON DELETE RESTRICT,
    notes TEXT,
    created_at TIMESTAMP DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS sale_items (
    id BIGSERIAL PRIMARY KEY,
    sale_id BIGINT NOT NULL REFERENCES sales(id) ON DELETE CASCADE,
    glass_id BIGINT NOT NULL REFERENCES glasses(id) ON DELETE RESTRICT
);

CREATE TABLE IF NOT EXISTS reserves (
    id BIGSERIAL PRIMARY KEY,
    station_id BIGINT NOT NULL REFERENCES stations(id) ON DELETE RESTRICT,
    user_id BIGINT NOT NULL REFERENCES users(id) ON DELETE RESTRICT,
    notes TEXT,
    created_at TIMESTAMP DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS reserve_items (
    id BIGSERIAL PRIMARY KEY,
    reserve_id BIGINT NOT NULL REFERENCES reserves(id) ON DELETE CASCADE,
    glass_id BIGINT NOT NULL REFERENCES glasses(id) ON DELETE RESTRICT
);

CREATE INDEX IF NOT EXISTS idx_sale_items_sale_id ON sale_items(sale_id);
CREATE INDEX IF NOT EXISTS idx_reserve_items_reserve_id ON reserve_items(reserve_id);
