ALTER TABLE supplier_orders
    DROP CONSTRAINT IF EXISTS supplier_orders_gamme_check;

ALTER TABLE supplier_orders
    ADD CONSTRAINT supplier_orders_gamme_check
    CHECK (gamme IN ('classique', 'moyenne', 'luxe', 'panache', 'lecture', 'solaire', 'securite'));