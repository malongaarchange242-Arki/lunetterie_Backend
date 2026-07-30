package services

import (
	"fmt"

	"github.com/jmoiron/sqlx"
	"github.com/lunetterie/backend/internal/inventory/models"
)

// AllocationService gère l'allocation d'emplacements
type AllocationService struct {
	db *sqlx.DB
}

// NewAllocationService crée une nouvelle instance
func NewAllocationService(db *sqlx.DB) *AllocationService {
	return &AllocationService{db: db}
}

// FindFreeLocation trouve un emplacement libre dans une station
func (s *AllocationService) FindFreeLocation(stationID int64, zone models.ZoneType) (*models.StorageLocation, error) {
	query := `
		SELECT id, station_id, code, type, zone, status
		FROM storage_locations
		WHERE station_id = $1 
		  AND zone = $2 
		  AND status = 'LIBRE'
		  AND type = 'POSITION'
		ORDER BY code
		LIMIT 1`

	var location models.StorageLocation
	err := s.db.Get(&location, query, stationID, zone)
	if err != nil {
		return nil, fmt.Errorf("aucun emplacement libre trouvé dans %s: %w", zone, err)
	}

	// Marquer l'emplacement comme occupé
	_, err = s.db.Exec(
		"UPDATE storage_locations SET status = 'OCCUPE' WHERE id = $1",
		location.ID,
	)
	if err != nil {
		return nil, fmt.Errorf("impossible de réserver l'emplacement: %w", err)
	}

	return &location, nil
}

// FindOrCreatePresentoirLocation trouve un emplacement libre en zone PRESENTOIR pour la station,
// ou en crée un à la volée si aucun n'a encore été généré (le générateur d'emplacements ne
// produit aujourd'hui que la zone STOCK). Le code est dérivé du code-barres pour rester unique.
func (s *AllocationService) FindOrCreatePresentoirLocation(stationID int64, barcode string) (*models.StorageLocation, error) {
	if location, err := s.FindFreeLocation(stationID, models.ZonePresentoir); err == nil {
		return location, nil
	}

	var location models.StorageLocation
	query := `
		INSERT INTO storage_locations (station_id, zone, code, type, status)
		VALUES ($1, $2, $3, 'POSITION', 'OCCUPE')
		RETURNING id, station_id, code, type, zone, status`
	err := s.db.Get(&location, query, stationID, models.ZonePresentoir, "PRESENTOIR-"+barcode)
	if err != nil {
		return nil, fmt.Errorf("impossible de créer un emplacement présentoir: %w", err)
	}
	return &location, nil
}

// FreeLocation libère un emplacement
func (s *AllocationService) FreeLocation(locationID int64) error {
	_, err := s.db.Exec(
		"UPDATE storage_locations SET status = 'LIBRE' WHERE id = $1",
		locationID,
	)
	return err
}
