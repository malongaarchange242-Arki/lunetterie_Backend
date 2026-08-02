ALTER TABLE reception_commands
    ADD COLUMN IF NOT EXISTS supplier_order_id BIGINT NULL REFERENCES supplier_orders(id) ON DELETE SET NULL;

CREATE INDEX IF NOT EXISTS idx_reception_commands_supplier_order_id ON reception_commands(supplier_order_id);
