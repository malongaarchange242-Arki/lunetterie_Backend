ALTER TABLE send_lists DROP CONSTRAINT IF EXISTS send_lists_status_check;
ALTER TABLE send_lists ADD CONSTRAINT send_lists_status_check CHECK (status IN ('NOUVELLE', 'VUE', 'TRAITEE', 'ANNULEE'));

