-- The glasses profile now stores only the monture photo.
ALTER TABLE glasses
    DROP COLUMN IF EXISTS photo_branche_url;
