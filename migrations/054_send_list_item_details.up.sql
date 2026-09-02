ALTER TABLE send_list_items
    ADD COLUMN IF NOT EXISTS shape VARCHAR(120),
    ADD COLUMN IF NOT EXISTS color VARCHAR(120);

CREATE INDEX IF NOT EXISTS idx_send_list_items_brand ON send_list_items(brand);
CREATE INDEX IF NOT EXISTS idx_send_list_items_shape ON send_list_items(shape);
CREATE INDEX IF NOT EXISTS idx_send_list_items_color ON send_list_items(color);
