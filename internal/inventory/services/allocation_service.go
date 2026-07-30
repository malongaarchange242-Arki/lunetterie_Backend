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

// presentoirLocationCode génère un code au même format que les emplacements de stock
// (RAYON-A-ETA-01-BAC-A-POS-03), à partir d'un rang séquentiel (1, 2, 3...).
func presentoirLocationCode(seq int) string {
	rayon := string(rune('A' + (seq-1)/50%10))
	etagere := (seq-1)/10%5 + 1
	bac := string(rune('A' + (seq-1)%10))
	position := (seq-1)%10 + 1
	return fmt.Sprintf("RAYON-%s-ETA-%02d-BAC-%s-POS-%02d", rayon, etagere, bac, position)
}

// FindOrCreatePresentoirLocation trouve un emplacement libre en zone PRESENTOIR pour la station,
// ou en crée un à la volée si aucun n'a encore été généré (le générateur d'emplacements ne
// produit aujourd'hui que la zone STOCK). Le code créé suit le même format que les emplacements
// de stock (RAYON-A-ETA-01-BAC-A-POS-03) pour rester cohérent visuellement.
func (s *AllocationService) FindOrCreatePresentoirLocation(stationID int64, barcode string) (*models.StorageLocation, error) {
	if location, err := s.FindFreeLocation(stationID, models.ZonePresentoir); err == nil {
		return location, nil
	}

	var count int
	if err := s.db.Get(&count, `SELECT COUNT(*) FROM storage_locations WHERE station_id = $1 AND zone = $2`, stationID, models.ZonePresentoir); err != nil {
		return nil, fmt.Errorf("impossible de compter les emplacements présentoir existants: %w", err)
	}

	// Petite boucle de retry en cas de course (deux créations concurrentes sur le même rang).
	for attempt := 1; attempt <= 5; attempt++ {
		code := presentoirLocationCode(count + attempt)

		var location models.StorageLocation
		query := `
			INSERT INTO storage_locations (station_id, zone, code, type, status)
			VALUES ($1, $2, $3, 'POSITION', 'OCCUPE')
			ON CONFLICT (station_id, zone, code) DO NOTHING
			RETURNING id, station_id, code, type, zone, status`
		err := s.db.Get(&location, query, stationID, models.ZonePresentoir, code)
		if err == nil {
			return &location, nil
		}
	}

	return nil, fmt.Errorf("impossible de créer un emplacement présentoir pour la station #%d", stationID)
}

// FreeLocation libère un emplacement
func (s *AllocationService) FreeLocation(locationID int64) error {
	_, err := s.db.Exec(
		"UPDATE storage_locations SET status = 'LIBRE' WHERE id = $1",
		locationID,
	)
	return err
}
