CREATE TABLE IF NOT EXISTS claims (
    id BIGSERIAL PRIMARY KEY,
    station_id BIGINT NOT NULL REFERENCES stations(id),
    client_name VARCHAR(160) NOT NULL,
    barcode VARCHAR(80) NULL,
    motif VARCHAR(40) NOT NULL,
    detail TEXT NULL,
    status VARCHAR(30) NOT NULL DEFAULT 'OUVERTE',
    created_by BIGINT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_claims_station ON claims(station_id, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_claims_status ON claims(status, created_at DESC);
