-- Compatibility rollback for databases that still need the legacy branch photo URL.
ALTER TABLE glasses
    ADD COLUMN IF NOT EXISTS photo_branche_url TEXT;
