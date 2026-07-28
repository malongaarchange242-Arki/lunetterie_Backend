package repositories

import (
	"fmt"

	"github.com/jmoiron/sqlx"
	"github.com/lunetterie/backend/internal/inventory/models"
)

// LocationRepository gère les emplacements de stockage
type LocationRepository struct {
	db *sqlx.DB
}

// NewLocationRepository crée une nouvelle instance
func NewLocationRepository(db *sqlx.DB) *LocationRepository {
	return &LocationRepository{db: db}
}

// GetByID récupère un emplacement par ID
func (r *LocationRepository) GetByID(id int64) (*models.StorageLocation, error) {
	var loc models.StorageLocation
	query := `SELECT * FROM storage_locations WHERE id = $1`
	err := r.db.Get(&loc, query, id)
	if err != nil {
		return nil, fmt.Errorf("emplacement introuvable: %w", err)
	}
	return &loc, nil
}

// UpdateStatus met à jour le statut d'un emplacement
func (r *LocationRepository) UpdateStatus(locationID int64, status string) error {
	query := `
		UPDATE storage_locations 
		SET status = $1 
		WHERE id = $2`
	_, err := r.db.Exec(query, status, locationID)
	return err
}
