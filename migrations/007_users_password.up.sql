-- Ajoute le hash du mot de passe pour la connexion par e-mail/mot de passe
ALTER TABLE users ADD COLUMN IF NOT EXISTS password_hash VARCHAR(255);
