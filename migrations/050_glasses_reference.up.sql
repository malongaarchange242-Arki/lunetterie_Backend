-- Store the physical frame reference directly on the glass record.
-- Keep glass_analysis.reference for compatibility with historical records.
ALTER TABLE glasses
  ADD COLUMN IF NOT EXISTS reference VARCHAR(100);

CREATE INDEX IF NOT EXISTS idx_glasses_reference ON glasses(reference);
