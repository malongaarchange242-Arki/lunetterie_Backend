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

// FindEmptyPresentoirSlotsToday liste les codes d'emplacement de la zone PRESENTOIR d'une
// station qui sont actuellement libres ET ont été libérés aujourd'hui suite à une vente ou une
// réserve (pas un simple emplacement jamais utilisé) — pour savoir quels emplacements physiques
// remplir en fin de journée.
func (r *LocationRepository) FindEmptyPresentoirSlotsToday(stationID int64) ([]string, error) {
	var codes []string
	query := `
		SELECT DISTINCT sl.code
		FROM storage_locations sl
		JOIN movements m ON m.from_location_id = sl.id
		WHERE sl.station_id = $1
		  AND sl.zone = 'PRESENTOIR'
		  AND sl.status = 'LIBRE'
		  AND m.action IN ('RETRAIT_PRESENTOIR', 'RESERVATION')
		  AND m.created_at::date = CURRENT_DATE
		ORDER BY sl.code`
	if err := r.db.Select(&codes, query, stationID); err != nil {
		return nil, fmt.Errorf("impossible de récupérer les emplacements vides: %w", err)
	}
	return codes, nil
}
