-- Rollback: Remove reference column from glasses table
DROP INDEX IF EXISTS idx_glasses_reference;
ALTER TABLE glasses DROP COLUMN IF EXISTS reference;
