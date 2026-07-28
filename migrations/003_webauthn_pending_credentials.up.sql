-- Allow temporary WebAuthn registration data before the user is created.
ALTER TABLE webauthn_credentials ALTER COLUMN user_id DROP NOT NULL;
ALTER TABLE webauthn_challenges ALTER COLUMN user_id DROP NOT NULL;
