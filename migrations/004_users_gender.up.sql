-- Ajoute la colonne genre sur users, utilisée par le formulaire "Ajouter un employé"
ALTER TABLE users ADD COLUMN IF NOT EXISTS gender VARCHAR(20) CHECK (gender IN ('Homme', 'Femme', 'Autre'));
