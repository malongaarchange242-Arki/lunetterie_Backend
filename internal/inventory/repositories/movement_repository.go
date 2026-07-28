package repositories

import (
	"fmt"

	"github.com/jmoiron/sqlx"
	"github.com/lunetterie/backend/internal/inventory/models"
)

// MovementRepository gère les mouvements de montures
type MovementRepository struct {
	db *sqlx.DB
}

// NewMovementRepository crée une nouvelle instance
func NewMovementRepository(db *sqlx.DB) *MovementRepository {
	return &MovementRepository{db: db}
}

// Create crée un nouveau mouvement
func (r *MovementRepository) Create(movement *models.Movement) error {
	query := `
		INSERT INTO movements (
			glass_id, from_station_id, to_station_id,
			from_location_id, to_location_id, action, user_id, notes
		) VALUES (
			:glass_id, :from_station_id, :to_station_id,
			:from_location_id, :to_location_id, :action, :user_id, :notes
		) RETURNING id, created_at`

	rows, err := r.db.NamedQuery(query, movement)
	if err != nil {
		return fmt.Errorf("erreur création mouvement: %w", err)
	}
	defer rows.Close()

	if rows.Next() {
		return rows.Scan(&movement.ID, &movement.CreatedAt)
	}

	return fmt.Errorf("aucun ID retourné après création mouvement")
}
