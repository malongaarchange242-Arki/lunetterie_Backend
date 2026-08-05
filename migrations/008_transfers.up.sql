-- Transferts inter-station (déjà utilisés en production, migration manquante ajoutée pour traçabilité)
CREATE TABLE IF NOT EXISTS transfers (
    id BIGSERIAL PRIMARY KEY,
    from_station_id BIGINT NOT NULL REFERENCES stations(id) ON DELETE RESTRICT,
    to_station_id BIGINT NOT NULL REFERENCES stations(id) ON DELETE RESTRICT,
    created_by BIGINT NOT NULL REFERENCES users(id) ON DELETE RESTRICT,
    received_by BIGINT REFERENCES users(id) ON DELETE SET NULL,
    status VARCHAR(20) DEFAULT 'PREPARATION' CHECK (status IN ('PREPARATION', 'IN_TRANSIT', 'RECEIVED', 'CANCELLED')),
    notes TEXT,
    created_at TIMESTAMP DEFAULT NOW(),
    received_at TIMESTAMP,
    CHECK (from_station_id <> to_station_id)
);

CREATE INDEX IF NOT EXISTS idx_transfers_from ON transfers(from_station_id);
CREATE INDEX IF NOT EXISTS idx_transfers_to ON transfers(to_station_id);
CREATE INDEX IF NOT EXISTS idx_transfers_status ON transfers(status);

CREATE TABLE IF NOT EXISTS transfer_items (
    id BIGSERIAL PRIMARY KEY,
    transfer_id BIGINT NOT NULL REFERENCES transfers(id) ON DELETE CASCADE,
    glass_id BIGINT NOT NULL REFERENCES glasses(id) ON DELETE RESTRICT,
    status VARCHAR(20) DEFAULT 'PENDING' CHECK (status IN ('PENDING', 'IN_TRANSIT', 'RECEIVED', 'MISSING')),
    scanned_at TIMESTAMP,
    UNIQUE (transfer_id, glass_id)
);

CREATE INDEX IF NOT EXISTS idx_transfer_items_transfer ON transfer_items(transfer_id);
CREATE INDEX IF NOT EXISTS idx_transfer_items_glass ON transfer_items(glass_id);
