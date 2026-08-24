package services

import (
	"fmt"
	"strings"

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

// FindFreeLocation trouve et réserve le premier emplacement libre d'une station,
// dans l'ordre du code (POS-01, POS-02, ... POS-20 par bac).
//
// Plus de découpage par tranche de prix : ça mélangeait l'aperçu affiché pendant
// la saisie (sans prix connu, donc toujours "classique") et l'enregistrement réel
// (avec le prix, donc parfois reclassé "moyenne gamme") — deux emplacements
// différents montrés au magasinier pour la même monture.
func (s *AllocationService) FindFreeLocation(
	stationID int64,
	zone models.ZoneType,
) (*models.StorageLocation, error) {

	lookupStationID := stationID

	// Seule la zone STOCK est centralisée au Stock Général :
	// PRESENTOIR et LABORATOIRE restent propres à chaque station.
	if zone == models.ZoneStock {
		resolved, err := s.resolveStorageStationID(stationID)
		if err != nil {
			return nil, err
		}

		lookupStationID = resolved
	}

	var selected models.StorageLocation

	err := s.db.Get(&selected, `
		SELECT
			id,
			station_id,
			code,
			type,
			zone,
			status
		FROM storage_locations
		WHERE station_id = $1
		  AND zone = $2
		  AND status = 'LIBRE'
		  AND type = 'POSITION'
		ORDER BY code
		LIMIT 1
	`, lookupStationID, zone)

	if err != nil {
		return nil, fmt.Errorf(
			"aucun emplacement libre trouvé dans %s",
			zone,
		)
	}

	_, err = s.db.Exec(
		"UPDATE storage_locations SET status = 'OCCUPE' WHERE id = $1",
		selected.ID,
	)

	if err != nil {
		return nil, fmt.Errorf(
			"impossible de réserver l'emplacement: %w",
			err,
		)
	}

	return &selected, nil
}

// resolveStorageStationID traduit une station opérationnelle
// (boutique, caisse...) vers la station qui porte physiquement son stock.
//
// Le Stock Général est la station physique qui contient les emplacements
// de la zone STOCK.
//
// Si la station demandée est elle-même un Stock Général, elle est prioritaire.
func (s *AllocationService) resolveStorageStationID(
	stationID int64,
) (int64, error) {

	var storageStationID int64

	query := `
		SELECT id
		FROM stations
		WHERE city = (
			SELECT city
			FROM stations
			WHERE id = $1
		)
		AND name ILIKE 'Stock Général%'
		ORDER BY
			CASE
				WHEN id = $1 THEN 0
				ELSE 1
			END,
			id
		LIMIT 1
	`

	err := s.db.Get(
		&storageStationID,
		query,
		stationID,
	)

	if err != nil {
		return 0, fmt.Errorf(
			"impossible de trouver le stock physique pour la station %d: %w",
			stationID,
			err,
		)
	}

	return storageStationID, nil
}

func codePrefix(code string) string {
	parts := strings.Split(code, "-")

	if len(parts) >= 6 {
		return strings.Join(parts[:6], "-")
	}

	return code
}

// FindFreeLocationNearCode trouve un emplacement libre proche
// d'un emplacement de référence.
func (s *AllocationService) FindFreeLocationNearCode(
	stationID int64,
	zone models.ZoneType,
	baseCode string,
) (*models.StorageLocation, error) {

	prefix := codePrefix(baseCode)

	if prefix == "" {
		return nil, fmt.Errorf("code de référence invalide")
	}

	query := `
		SELECT
			id,
			station_id,
			code,
			type,
			zone,
			status
		FROM storage_locations
		WHERE station_id = $1
		  AND zone = $2
		  AND status = 'LIBRE'
		  AND type = 'POSITION'
		  AND code LIKE $3
		ORDER BY code
		LIMIT 1
	`

	var location models.StorageLocation

	err := s.db.Get(
		&location,
		query,
		stationID,
		zone,
		prefix+"-%",
	)

	if err != nil {
		return nil, fmt.Errorf(
			"aucun emplacement libre proche de %s: %w",
			baseCode,
			err,
		)
	}

	_, err = s.db.Exec(
		"UPDATE storage_locations SET status = 'OCCUPE' WHERE id = $1",
		location.ID,
	)

	if err != nil {
		return nil, fmt.Errorf(
			"impossible de réserver l'emplacement: %w",
			err,
		)
	}

	return &location, nil
}

// genericLocationCode génère un code au même format que les emplacements
// de stock.
//
// Exemple :
// RAYON-A-ETA-01-BAC-A-POS-03
func genericLocationCode(seq int) string {
	rayon := string(rune('A' + (seq-1)/50%10))
	etagere := (seq-1)/10%5 + 1
	bac := string(rune('A' + (seq-1)%10))
	position := (seq-1)%10 + 1

	return fmt.Sprintf(
		"RAYON-%s-ETA-%02d-BAC-%s-POS-%02d",
		rayon,
		etagere,
		bac,
		position,
	)
}

// presentoirPositionsPerUnit détermine à partir de combien d'emplacements
// on numérote un nouveau meuble présentoir.
const presentoirPositionsPerUnit = 20

// presentoirLocationCode génère un code "meuble-position".
//
// Exemple :
// PR01-1
// PR01-2
// ...
// PR01-20
// PR02-1
func presentoirLocationCode(seq int) string {
	unit := (seq-1)/presentoirPositionsPerUnit + 1
	position := (seq-1)%presentoirPositionsPerUnit + 1

	return fmt.Sprintf(
		"PR%02d-%d",
		unit,
		position,
	)
}

// findOrCreateLocation trouve un emplacement libre dans une zone donnée
// pour une station, ou en crée un automatiquement.
//
// Pour STOCK, les emplacements sont centralisés dans le Stock Général.
// Pour PRESENTOIR et LABORATOIRE, ils restent propres à la station.
func (s *AllocationService) findOrCreateLocation(
	stationID int64,
	zone models.ZoneType,
) (*models.StorageLocation, error) {

	// ------------------------------------------------------------
	// 1. Chercher d'abord un emplacement libre existant
	// ------------------------------------------------------------

	if location, err := s.FindFreeLocation(stationID, zone); err == nil {
		return location, nil
	}

	// ------------------------------------------------------------
	// 2. Déterminer la station physique utilisée
	// ------------------------------------------------------------

	lookupStationID := stationID

	if zone == models.ZoneStock {
		resolved, err := s.resolveStorageStationID(stationID)

		if err != nil {
			return nil, err
		}

		lookupStationID = resolved
	}

	// ------------------------------------------------------------
	// 3. Compter les emplacements existants
	// ------------------------------------------------------------

	var count int

	err := s.db.Get(
		&count,
		`
		SELECT COUNT(*)
		FROM storage_locations
		WHERE station_id = $1
		  AND zone = $2
		`,
		lookupStationID,
		zone,
	)

	if err != nil {
		return nil, fmt.Errorf(
			"impossible de compter les emplacements existants (zone %s): %w",
			zone,
			err,
		)
	}

	// ------------------------------------------------------------
	// 4. Créer un nouvel emplacement
	// ------------------------------------------------------------

	// Petite boucle de retry en cas de concurrence :
	// deux réceptions peuvent essayer de créer le même emplacement
	// simultanément.
	for attempt := 1; attempt <= 5; attempt++ {

		code := genericLocationCode(count + attempt)

		if zone == models.ZonePresentoir {
			code = presentoirLocationCode(count + attempt)
		}

		var location models.StorageLocation

		query := `
			INSERT INTO storage_locations (
				station_id,
				zone,
				code,
				type,
				status
			)
			VALUES (
				$1,
				$2,
				$3,
				'POSITION',
				'OCCUPE'
			)
			ON CONFLICT (station_id, zone, code)
			DO NOTHING
			RETURNING
				id,
				station_id,
				code,
				type,
				zone,
				status
		`

		// IMPORTANT :
		// Pour STOCK, on doit utiliser lookupStationID et non stationID.
		//
		// Avant :
		//     stationID
		//
		// Correction :
		//     lookupStationID
		//
		// Le COUNT et l'INSERT doivent travailler sur la même station
		// physique.
		err := s.db.Get(
			&location,
			query,
			lookupStationID,
			zone,
			code,
		)

		if err == nil {
			return &location, nil
		}
	}

	// ------------------------------------------------------------
	// 5. Échec après plusieurs tentatives
	// ------------------------------------------------------------

	return nil, fmt.Errorf(
		"impossible de créer un emplacement (zone %s) pour la station #%d",
		zone,
		stationID,
	)
}

// FindOrCreatePresentoirLocation trouve un emplacement libre
// en zone PRESENTOIR ou en crée un automatiquement.
func (s *AllocationService) FindOrCreatePresentoirLocation(
	stationID int64,
	barcode string,
) (*models.StorageLocation, error) {

	return s.findOrCreateLocation(
		stationID,
		models.ZonePresentoir,
	)
}

// FindOrCreateStockLocation trouve un emplacement libre
// en zone STOCK ou en crée un automatiquement.
//
// Les emplacements STOCK sont physiquement rattachés au
// Stock Général de la ville.
func (s *AllocationService) FindOrCreateStockLocation(
	stationID int64,
) (*models.StorageLocation, error) {

	return s.findOrCreateLocation(
		stationID,
		models.ZoneStock,
	)
}

// FreeLocation libère un emplacement.
func (s *AllocationService) FreeLocation(
	locationID int64,
) error {

	_, err := s.db.Exec(
		"UPDATE storage_locations SET status = 'LIBRE' WHERE id = $1",
		locationID,
	)

	return err
}
