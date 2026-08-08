CREATE TABLE IF NOT EXISTS send_boxes (
    id BIGSERIAL PRIMARY KEY,
    list_id BIGINT NOT NULL REFERENCES send_lists(id) ON DELETE CASCADE,
    code VARCHAR(80) NOT NULL UNIQUE,
    reference VARCHAR(120) NOT NULL UNIQUE,
    city VARCHAR(120) NOT NULL,
    session_code VARCHAR(120) NOT NULL,
    item_count INTEGER NOT NULL DEFAULT 0,
    status VARCHAR(40) NOT NULL DEFAULT 'CREATED',
    created_by BIGINT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS send_box_items (
    id BIGSERIAL PRIMARY KEY,
    box_id BIGINT NOT NULL REFERENCES send_boxes(id) ON DELETE CASCADE,
    list_item_id BIGINT NOT NULL REFERENCES send_list_items(id) ON DELETE CASCADE,
    glass_id BIGINT NULL REFERENCES glasses(id) ON DELETE SET NULL,
    barcode VARCHAR(80) NULL,
    reference VARCHAR(120) NULL,
    location_code VARCHAR(80) NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_send_boxes_list ON send_boxes(list_id);
CREATE INDEX IF NOT EXISTS idx_send_boxes_code ON send_boxes(code);
CREATE INDEX IF NOT EXISTS idx_send_boxes_city ON send_boxes(city);
CREATE INDEX IF NOT EXISTS idx_send_box_items_box ON send_box_items(box_id);
CREATE INDEX IF NOT EXISTS idx_send_box_items_list_item ON send_box_items(list_item_id);
