ALTER TABLE proforma_items DROP COLUMN IF EXISTS offerte;
ALTER TABLE proforma_items DROP COLUMN IF EXISTS color;
ALTER TABLE proforma_items DROP COLUMN IF EXISTS shape;

DROP INDEX IF EXISTS idx_proforma_prescriptions_foyer;

DROP TABLE IF EXISTS proforma_prescriptions;
