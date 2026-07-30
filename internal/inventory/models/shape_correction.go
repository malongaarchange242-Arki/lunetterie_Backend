package models

import "time"

// ShapeCorrection trace le cas où l'utilisateur corrige la forme détectée par l'IA
// à l'écran de vérification. Sert à constituer un jeu de données pour le futur
// ré-entraînement du classificateur de formes.
type ShapeCorrection struct {
	ID             int64     `db:"id" json:"id"`
	GlassID        int64     `db:"glass_id" json:"glass_id"`
	DetectedShape  string    `db:"detected_shape" json:"detected_shape"`
	CorrectedShape string    `db:"corrected_shape" json:"corrected_shape"`
	UserID         *int64    `db:"user_id" json:"user_id,omitempty"`
	CreatedAt      time.Time `db:"created_at" json:"created_at"`
}
