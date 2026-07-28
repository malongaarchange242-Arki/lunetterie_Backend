-- Prix de vente saisi lors de la réception (par exemplaire physique)
ALTER TABLE glasses ADD COLUMN IF NOT EXISTS price NUMERIC(12,2);
