-- Ensure reference column exists on glasses table (fallback migration)
-- This runs after all previous migrations and guarantees the column is present

-- Step 1: Add column if it doesn't exist
DO $$
BEGIN
  ALTER TABLE glasses ADD COLUMN IF NOT EXISTS reference VARCHAR(100);
  RAISE NOTICE 'Column reference added to glasses table';
EXCEPTION WHEN duplicate_column THEN
  RAISE NOTICE 'Column reference already exists on glasses table';
END;
$$;

-- Step 2: Create index if it doesn't exist
CREATE INDEX IF NOT EXISTS idx_glasses_reference ON glasses(reference);

-- Step 3: Verify the column exists for troubleshooting
DO $$
DECLARE
  col_exists BOOLEAN;
BEGIN
  SELECT EXISTS (
    SELECT 1 FROM information_schema.columns 
    WHERE table_name = 'glasses' AND column_name = 'reference'
  ) INTO col_exists;
  
  IF col_exists THEN
    RAISE NOTICE 'Verified: reference column exists on glasses table';
  ELSE
    RAISE EXCEPTION 'CRITICAL: reference column does NOT exist on glasses table after migration!';
  END IF;
END;
$$;
