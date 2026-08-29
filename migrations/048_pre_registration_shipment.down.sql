ALTER TABLE pre_registration_cases
    DROP COLUMN IF EXISTS shipment_scanned_at,
    DROP COLUMN IF EXISTS shipment_scanned;

ALTER TABLE reception_commands
    DROP COLUMN IF EXISTS arrived_at,
    DROP COLUMN IF EXISTS dispatched_at,
    DROP COLUMN IF EXISTS shipment_status;

DROP INDEX IF EXISTS idx_reception_commands_shipment_status;
