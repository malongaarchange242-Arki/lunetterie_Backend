-- Rollback is only valid after pending rows have been linked or removed.
DELETE FROM webauthn_credentials WHERE user_id IS NULL;
DELETE FROM webauthn_challenges WHERE user_id IS NULL;
ALTER TABLE webauthn_credentials ALTER COLUMN user_id SET NOT NULL;
ALTER TABLE webauthn_challenges ALTER COLUMN user_id SET NOT NULL;
