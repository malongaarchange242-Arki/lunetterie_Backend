package services

import (
	"fmt"
	"strings"

	"github.com/jmoiron/sqlx"
	"github.com/lunetterie/backend/internal/inventory/models"
)

const (
	priceTierClassique = "classique"
	priceTierMoyenne   = "moyenne"
	priceTierLuxe      = "luxe"
)

// AllocationService gère l'allocation d'emplacements
type AllocationService struct {
	db *sqlx.DB
}

// NewAllocationService crée une nouvelle instance
func NewAllocationService(db *sqlx.DB) *AllocationService {
	return &AllocationService{db: db}
}

// FindFreeLocation trouve un emplacement libre dans une station.
// Les emplacements sont désormais partagés en trois pools selon la gamme :
// classique, moyenne, luxe. Si aucun emplacement du pool demandé n'est libre,
// on retombe sur le premier emplacement libre disponible de la zone.
func (s *AllocationService) FindFreeLocation(stationID int64, zone models.ZoneType) (*models.StorageLocation, error) {
	return s.findFreeLocationForTier(stationID, zone, "")
}

// resolveStorageStationID traduit une station opérationnelle (boutique, caisse...) vers la
// station qui porte physiquement son stock — le "Stock Général" de sa ville. Ces stations
// n'ont pas leur propre arborescence storage_locations en zone STOCK : les emplacements
// physiques n'existent que sur le Stock Général, quelle que soit la station à laquelle un
// compte ou une monture est rattaché administrativement.
//
// CASE WHEN id = $1 THEN 0 : si la station demandée EST déjà un Stock Général, elle se
// choisit elle-même plutôt qu'un homonyme d'une autre ville partageant la même liste de
// candidats — la requête ne filtre que par ville, pas par égalité stricte de nom.
func (s *AllocationService) resolveStorageStationID(stationID int64) (int64, error) {
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

	err := s.db.Get(&storageStationID, query, stationID)
	if err != nil {
		return 0, fmt.Errorf(
			"impossible de trouver le stock physique pour la station %d: %w",
			stationID,
			err,
		)
	}

	return storageStationID, nil
}

func (s *AllocationService) findFreeLocationForTier(stationID int64, zone models.ZoneType, tier string) (*models.StorageLocation, error) {
	lookupStationID := stationID
	// Seule la zone STOCK est centralisée au Stock Général : PRESENTOIR et LABORATOIRE restent
	// propres à chaque station (le présentoir d'une boutique est chez elle, pas à l'entrepôt).
	if zone == models.ZoneStock {
		resolved, err := s.resolveStorageStationID(stationID)
		if err != nil {
			return nil, err
		}
		lookupStationID = resolved
	}

	query := `
		SELECT id, station_id, code, type, zone, status
		FROM storage_locations
		WHERE station_id = $1
		  AND zone = $2
		  AND status = 'LIBRE'
		  AND type = 'POSITION'
		ORDER BY code`

	var locations []models.StorageLocation
	err := s.db.Select(&locations, query, lookupStationID, zone)
	if err != nil {
		return nil, fmt.Errorf("impossible de lister les emplacements libres: %w", err)
	}
	if len(locations) == 0 {
		return nil, fmt.Errorf("aucun emplacement libre trouvé dans %s", zone)
	}

	selected := selectLocationByTier(locations, tier)
	if selected == nil {
		selected = &locations[0]
	}

	_, err = s.db.Exec(
		"UPDATE storage_locations SET status = 'OCCUPE' WHERE id = $1",
		selected.ID,
	)
	if err != nil {
		return nil, fmt.Errorf("impossible de réserver l'emplacement: %w", err)
	}

	return selected, nil
}

func classifyPriceTier(price *float64, gamme string) string {
	if strings.TrimSpace(strings.ToLower(gamme)) != "" {
		return strings.ToLower(strings.TrimSpace(gamme))
	}
	if price == nil {
		return priceTierClassique
	}
	value := *price
	if value > 100000 {
		return priceTierLuxe
	}
	if value > 50000 {
		return priceTierMoyenne
	}
	return priceTierClassique
}

func selectLocationByTier(locations []models.StorageLocation, tier string) *models.StorageLocation {
	normalizedTier := strings.ToLower(strings.TrimSpace(tier))
	if normalizedTier == "" {
		return nil
	}

	var preferred []models.StorageLocation
	var fallback []models.StorageLocation
	for _, location := range locations {
		code := strings.ToUpper(location.Code)
		switch normalizedTier {
		case priceTierClassique:
			if strings.Contains(code, "POS-0") || strings.Contains(code, "POS-01") || strings.Contains(code, "POS-02") || strings.Contains(code, "POS-03") || strings.Contains(code, "POS-04") || strings.Contains(code, "POS-05") {
				preferred = append(preferred, location)
			} else {
				fallback = append(fallback, location)
			}
		case priceTierMoyenne:
			if strings.Contains(code, "POS-06") || strings.Contains(code, "POS-07") || strings.Contains(code, "POS-08") || strings.Contains(code, "POS-09") || strings.Contains(code, "POS-10") {
				preferred = append(preferred, location)
			} else {
				fallback = append(fallback, location)
			}
		case priceTierLuxe:
			if strings.Contains(code, "POS-11") || strings.Contains(code, "POS-12") || strings.Contains(code, "POS-13") || strings.Contains(code, "POS-14") || strings.Contains(code, "POS-15") {
				preferred = append(preferred, location)
			} else {
				fallback = append(fallback, location)
			}
		}
	}

	if len(preferred) > 0 {
		return &preferred[0]
	}
	if len(fallback) > 0 {
		return &fallback[0]
	}
	return nil
}

func codePrefix(code string) string {
	parts := strings.Split(code, "-")
	if len(parts) >= 6 {
		return strings.Join(parts[:6], "-")
	}
	return code
}

func (s *AllocationService) FindFreeLocationNearCode(stationID int64, zone models.ZoneType, baseCode string) (*models.StorageLocation, error) {
	prefix := codePrefix(baseCode)
	if prefix == "" {
		return nil, fmt.Errorf("code de référence invalide")
	}

	query := `
		SELECT id, station_id, code, type, zone, status
		FROM storage_locations
		WHERE station_id = $1
		  AND zone = $2
		  AND status = 'LIBRE'
		  AND type = 'POSITION'
		  AND code LIKE $3
		ORDER BY code
		LIMIT 1`

	var location models.StorageLocation
	err := s.db.Get(&location, query, stationID, zone, prefix+"-%")
	if err != nil {
		return nil, fmt.Errorf("aucun emplacement libre proche de %s: %w", baseCode, err)
	}

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

// FindFreeLocationForPrice allocates a stock location from a dedicated tier pool.
func (s *AllocationService) FindFreeLocationForPrice(stationID int64, zone models.ZoneType, price *float64, gamme string) (*models.StorageLocation, error) {
	tier := classifyPriceTier(price, gamme)
	return s.findFreeLocationForTier(stationID, zone, tier)
}

// FreeLocation libère un emplacement
func (s *AllocationService) FreeLocation(locationID int64) error {
	_, err := s.db.Exec(
		"UPDATE storage_locations SET status = 'LIBRE' WHERE id = $1",
		locationID,
	)
	return err
}
