ALTER TABLE reception_commands
    ADD COLUMN IF NOT EXISTS shipment_status VARCHAR(24) NOT NULL DEFAULT 'not_shipped',
    ADD COLUMN IF NOT EXISTS dispatched_at TIMESTAMP WITH TIME ZONE NULL,
    ADD COLUMN IF NOT EXISTS arrived_at TIMESTAMP WITH TIME ZONE NULL;

ALTER TABLE pre_registration_cases
    ADD COLUMN IF NOT EXISTS shipment_scanned BOOLEAN NOT NULL DEFAULT FALSE,
    ADD COLUMN IF NOT EXISTS shipment_scanned_at TIMESTAMP WITH TIME ZONE NULL;

CREATE INDEX IF NOT EXISTS idx_reception_commands_shipment_status
    ON reception_commands(shipment_status);
