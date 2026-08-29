ALTER TABLE supplier_orders
    ADD COLUMN IF NOT EXISTS reference VARCHAR(64),
    ADD COLUMN IF NOT EXISTS provenance VARCHAR(120) NOT NULL DEFAULT '',
    ADD COLUMN IF NOT EXISTS destination VARCHAR(120) NOT NULL DEFAULT 'Stock général',
    ADD COLUMN IF NOT EXISTS transport VARCHAR(80) NOT NULL DEFAULT '',
    ADD COLUMN IF NOT EXISTS barcode_num VARCHAR(120) NOT NULL DEFAULT '',
    ADD COLUMN IF NOT EXISTS status VARCHAR(32) NOT NULL DEFAULT 'attente';

UPDATE supplier_orders
SET reference = COALESCE(NULLIF(reference, ''), 'BC-HIST-' || id::text),
    barcode_num = COALESCE(NULLIF(barcode_num, ''), COALESCE(NULLIF(reference, ''), 'BC-HIST-' || id::text));

ALTER TABLE supplier_orders ALTER COLUMN reference SET NOT NULL;
CREATE UNIQUE INDEX IF NOT EXISTS idx_supplier_orders_reference ON supplier_orders(reference);