-- URLs des photos (monture + branche + arrière) stockées dans le bucket Supabase "glasses-photos"
ALTER TABLE glasses ADD COLUMN IF NOT EXISTS photo_monture_url TEXT;
ALTER TABLE glasses ADD COLUMN IF NOT EXISTS photo_branche_url TEXT;
ALTER TABLE glasses ADD COLUMN IF NOT EXISTS photo_arriere_url TEXT;
