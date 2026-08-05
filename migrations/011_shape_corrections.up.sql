-- Historique des corrections manuelles de la forme détectée par l'IA
-- (alimente le futur ré-entraînement du classificateur de formes).
CREATE TABLE IF NOT EXISTS shape_corrections (
    id BIGSERIAL PRIMARY KEY,
    glass_id BIGINT NOT NULL REFERENCES glasses(id) ON DELETE CASCADE,
    detected_shape VARCHAR(50) NOT NULL,
    corrected_shape VARCHAR(50) NOT NULL,
    user_id BIGINT REFERENCES users(id) ON DELETE SET NULL,
    created_at TIMESTAMP DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_shape_corrections_glass ON shape_corrections(glass_id);
