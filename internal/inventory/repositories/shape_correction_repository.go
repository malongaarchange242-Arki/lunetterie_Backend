package repositories

import (
	"fmt"

	"github.com/jmoiron/sqlx"
	"github.com/lunetterie/backend/internal/inventory/models"
)

// ShapeCorrectionRepository gère l'historique des corrections de forme
type ShapeCorrectionRepository struct {
	db *sqlx.DB
}

// NewShapeCorrectionRepository crée une nouvelle instance
func NewShapeCorrectionRepository(db *sqlx.DB) *ShapeCorrectionRepository {
	return &ShapeCorrectionRepository{db: db}
}

// Create enregistre une correction de forme
func (r *ShapeCorrectionRepository) Create(correction *models.ShapeCorrection) error {
	query := `INSERT INTO shape_corrections (glass_id, detected_shape, corrected_shape, user_id)
		VALUES (:glass_id, :detected_shape, :corrected_shape, :user_id)
		RETURNING id, created_at`

	rows, err := r.db.NamedQuery(query, correction)
	if err != nil {
		return fmt.Errorf("erreur création correction de forme: %w", err)
	}
	defer rows.Close()

	if rows.Next() {
		return rows.Scan(&correction.ID, &correction.CreatedAt)
	}
	return fmt.Errorf("aucun ID retourné après création correction de forme")
}
