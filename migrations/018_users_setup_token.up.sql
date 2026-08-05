-- Jeton à usage unique pour définir le mot de passe initial (voir /auth/set-password) :
-- sans ça, n'importe qui connaissant l'email d'un compte fraîchement créé pouvait lui
-- définir un mot de passe et en prendre le contrôle.
ALTER TABLE users ADD COLUMN IF NOT EXISTS setup_token VARCHAR(64);
ALTER TABLE users ADD COLUMN IF NOT EXISTS setup_token_expires_at TIMESTAMP;
