DROP INDEX IF EXISTS idx_proforma_prescriptions_societe;
ALTER TABLE proforma_prescriptions DROP COLUMN IF EXISTS societe_id;

DROP INDEX IF EXISTS idx_societes_active;
DROP INDEX IF EXISTS idx_societes_name_unique;
DROP TABLE IF EXISTS societes;
