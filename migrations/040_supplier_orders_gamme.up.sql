ALTER TABLE supplier_orders
    ADD COLUMN IF NOT EXISTS gamme VARCHAR(20) NOT NULL DEFAULT 'classique'
    CHECK (gamme IN ('classique', 'moyenne', 'luxe', 'panache'));
