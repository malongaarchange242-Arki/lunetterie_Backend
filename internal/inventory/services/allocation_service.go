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

// genericLocationCode génère un code au même format que les emplacements de stock
// (RAYON-A-ETA-01-BAC-A-POS-03), à partir d'un rang séquentiel (1, 2, 3...).
func genericLocationCode(seq int) string {
	rayon := string(rune('A' + (seq-1)/50%10))
	etagere := (seq-1)/10%5 + 1
	bac := string(rune('A' + (seq-1)%10))
	position := (seq-1)%10 + 1
	return fmt.Sprintf("RAYON-%s-ETA-%02d-BAC-%s-POS-%02d", rayon, etagere, bac, position)
}

// presentoirPositionsPerUnit détermine à partir de combien d'emplacements on numérote un
// nouveau meuble présentoir (PR01, PR02, ...) plutôt que d'ajouter une position au même meuble.
const presentoirPositionsPerUnit = 20

// presentoirLocationCode génère un code "meuble-position" pour les emplacements de la zone
// PRESENTOIR, ex: PR01-1, PR01-2, ..., PR01-20, PR02-1, ... — cohérent avec le format affiché
// à l'écran vendeur (Présentoir : PR03, Position : 12 -> Emplacement : PR03-12).
func presentoirLocationCode(seq int) string {
	unit := (seq-1)/presentoirPositionsPerUnit + 1
	position := (seq-1)%presentoirPositionsPerUnit + 1
	return fmt.Sprintf("PR%02d-%d", unit, position)
}

// findOrCreateLocation trouve un emplacement libre dans une zone donnée pour une station, ou en
// crée un à la volée (même format de code que le générateur d'emplacements) si aucun n'existe
// encore pour cette station/zone — utile car le générateur n'est pas systématiquement lancé pour
// toutes les zones de chaque nouvelle station.
func (s *AllocationService) findOrCreateLocation(stationID int64, zone models.ZoneType) (*models.StorageLocation, error) {
	if location, err := s.FindFreeLocation(stationID, zone); err == nil {
		return location, nil
	}

	var count int
	if err := s.db.Get(&count, `SELECT COUNT(*) FROM storage_locations WHERE station_id = $1 AND zone = $2`, stationID, zone); err != nil {
		return nil, fmt.Errorf("impossible de compter les emplacements existants (zone %s): %w", zone, err)
	}

	// Petite boucle de retry en cas de course (deux créations concurrentes sur le même rang).
	for attempt := 1; attempt <= 5; attempt++ {
		code := genericLocationCode(count + attempt)
		if zone == models.ZonePresentoir {
			code = presentoirLocationCode(count + attempt)
		}

		var location models.StorageLocation
		query := `
			INSERT INTO storage_locations (station_id, zone, code, type, status)
			VALUES ($1, $2, $3, 'POSITION', 'OCCUPE')
			ON CONFLICT (station_id, zone, code) DO NOTHING
			RETURNING id, station_id, code, type, zone, status`
		err := s.db.Get(&location, query, stationID, zone, code)
		if err == nil {
			return &location, nil
		}
	}

	return nil, fmt.Errorf("impossible de créer un emplacement (zone %s) pour la station #%d", zone, stationID)
}

// FindOrCreatePresentoirLocation trouve un emplacement libre en zone PRESENTOIR pour la station,
// ou en crée un à la volée si aucun n'a encore été généré.
func (s *AllocationService) FindOrCreatePresentoirLocation(stationID int64, barcode string) (*models.StorageLocation, error) {
	return s.findOrCreateLocation(stationID, models.ZonePresentoir)
}

// FindOrCreateStockLocation trouve un emplacement libre en zone STOCK pour la station, ou en crée
// un à la volée si aucun n'a encore été généré (ex: le générateur d'emplacements n'a jamais été
// lancé pour cette station) — sans ce repli, la réception d'un transfert reste bloquée en
// silence dès qu'aucun emplacement de stock n'est disponible.
func (s *AllocationService) FindOrCreateStockLocation(stationID int64) (*models.StorageLocation, error) {
	return s.findOrCreateLocation(stationID, models.ZoneStock)
}

// FreeLocation libère un emplacement
func (s *AllocationService) FreeLocation(locationID int64) error {
	_, err := s.db.Exec(
		"UPDATE storage_locations SET status = 'LIBRE' WHERE id = $1",
		locationID,
	)
	return err
}
