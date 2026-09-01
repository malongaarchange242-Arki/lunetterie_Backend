package services

import (
	"fmt"
	"log"
	"strings"

	"github.com/jmoiron/sqlx"
	"github.com/lib/pq"
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
// dans l'ordre du code des cartons.
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
		SELECT sl.id, sl.station_id, sl.code, sl.name, sl.barcode, sl.type, sl.zone, sl.capacity, sl.status
		FROM storage_locations sl
		WHERE sl.station_id = $1
		  AND sl.zone = $2
		  AND (sl.status = 'LIBRE' OR sl.type = 'CARTON')
		  AND sl.type = CASE WHEN sl.zone = 'STOCK' THEN 'CARTON' ELSE 'PRESENTOIR' END
		  AND (sl.capacity IS NULL OR (SELECT COUNT(*) FROM glasses g WHERE g.location_id = sl.id) < sl.capacity)
		ORDER BY code
		LIMIT 1
	`, lookupStationID, zone)

	if err != nil {
		return nil, fmt.Errorf(
			"aucun emplacement libre trouvé dans %s",
			zone,
		)
	}

	if zone != models.ZoneStock {
		_, err = s.db.Exec("UPDATE storage_locations SET status = 'OCCUPE' WHERE id = $1", selected.ID)
		if err != nil {
			return nil, fmt.Errorf("impossible de réserver l'emplacement: %w", err)
		}
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

	// Stratégie de recherche :
	// 1. Si la station demandée est "Stock Général", la retourner directement
	// 2. Chercher "Stock Général" dans la même ville (si la ville existe)
	// 3. Sinon, chercher "Stock Général" n'importe où
	query := `
		SELECT id
		FROM stations
		WHERE name ILIKE 'Stock Général%'
		ORDER BY
			CASE
				WHEN id = $1 THEN 0
				WHEN city = (SELECT city FROM stations WHERE id = $1 AND city IS NOT NULL) THEN 1
				ELSE 2
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

func nextPreRegistrationBoxCode(sequence int64) string {
	return fmt.Sprintf("CTN-%04d", sequence)
}

func (s *AllocationService) createNextPreRegistrationCarton(
	lookupStationID int64,
	valise *models.StorageLocation,
	carton struct {
		BoxID    int64          `db:"box_id"`
		BoxCode  string         `db:"box_code"`
		Quantity int            `db:"quantity"`
		CaseID   int64          `db:"case_id"`
		CaseCode string         `db:"case_code"`
		Formes   []byte         `db:"formes"`
		Marques  pq.StringArray `db:"marques"`
		Couleurs pq.StringArray `db:"couleurs"`
		Matieres pq.StringArray `db:"matieres"`
		Photos   []byte         `db:"photos"`
		Gamme    string         `db:"gamme"`
		BoxType  string         `db:"type_lunette"`
		Prix     float64        `db:"prix"`
	},
) (*models.StorageLocation, error) {
	var sequence int64
	if err := s.db.Get(&sequence, `SELECT nextval('carton_code_seq')`); err != nil {
		return nil, fmt.Errorf("impossible de générer le code du carton suivant: %w", err)
	}
	newCode := nextPreRegistrationBoxCode(sequence)

	var newBoxID int64
	if err := s.db.Get(&newBoxID, `
		INSERT INTO pre_registration_boxes (
			case_id, code, quantity, formes, marques, couleurs, matieres, photos, gamme, type_lunette, prix
		)
		VALUES ($1, $2, $3, $4::jsonb, $5, $6, $7, $8::jsonb, $9, $10, $11)
		RETURNING id`,
		carton.CaseID,
		newCode,
		carton.Quantity,
		string(carton.Formes),
		pq.Array(carton.Marques),
		pq.Array(carton.Couleurs),
		pq.Array(carton.Matieres),
		string(carton.Photos),
		carton.Gamme,
		carton.BoxType,
		carton.Prix,
	); err != nil {
		return nil, fmt.Errorf("impossible de créer un nouveau carton pour la même commande: %w", err)
	}

	var location models.StorageLocation
	if err := s.db.Get(&location, `
		INSERT INTO storage_locations (station_id, parent_location_id, zone, code, name, type, barcode, capacity, status)
		VALUES ($1, $2, 'STOCK', $3, $4, 'CARTON', $3, $5, 'LIBRE')
		ON CONFLICT (station_id, zone, code) DO UPDATE SET
			parent_location_id = EXCLUDED.parent_location_id,
			name = EXCLUDED.name,
			barcode = COALESCE(storage_locations.barcode, EXCLUDED.barcode),
			capacity = EXCLUDED.capacity
		RETURNING id, station_id, parent_location_id, zone, code, name, barcode, type, capacity, status, created_at`,
		lookupStationID,
		valise.ID,
		newCode,
		"Carton "+newCode,
		carton.Quantity,
	); err != nil {
		return nil, fmt.Errorf("impossible de préparer le carton %s: %w", newCode, err)
	}

	_ = newBoxID
	return &location, nil
}

func (s *AllocationService) FindOrCreatePreRegistrationCartonLocation(
	stationID int64,
	commandCode *string,
	boxID *int64,
	boxCode *string,
) (*models.StorageLocation, error) {
	if boxID == nil && (boxCode == nil || strings.TrimSpace(*boxCode) == "") {
		return nil, fmt.Errorf("carton de pre-enregistrement requis")
	}

	lookupStationID, err := s.resolveStorageStationID(stationID)
	if err != nil {
		return nil, err
	}

	type preRegistrationCarton struct {
		BoxID    int64          `db:"box_id"`
		BoxCode  string         `db:"box_code"`
		Quantity int            `db:"quantity"`
		CaseID   int64          `db:"case_id"`
		CaseCode string         `db:"case_code"`
		Formes   []byte         `db:"formes"`
		Marques  pq.StringArray `db:"marques"`
		Couleurs pq.StringArray `db:"couleurs"`
		Matieres pq.StringArray `db:"matieres"`
		Photos   []byte         `db:"photos"`
		Gamme    string         `db:"gamme"`
		BoxType  string         `db:"type_lunette"`
		Prix     float64        `db:"prix"`
	}

	var idArg any
	if boxID != nil {
		idArg = *boxID
	}
	codeArg := ""
	if boxCode != nil {
		codeArg = strings.TrimSpace(*boxCode)
	}
	commandArg := ""
	if commandCode != nil {
		commandArg = strings.TrimSpace(*commandCode)
	}

	var carton preRegistrationCarton
	if err := s.db.Get(&carton, `
		SELECT b.id AS box_id, b.code AS box_code, b.quantity,
		       c.id AS case_id, c.code AS case_code,
		       b.formes, b.marques, b.couleurs, b.matieres, b.photos,
		       b.gamme, b.type_lunette, b.prix
		FROM pre_registration_boxes b
		JOIN pre_registration_cases c ON c.id = b.case_id
		JOIN reception_commands rc ON rc.id = c.reception_command_id
		WHERE (($1::bigint IS NOT NULL AND b.id = $1)
		   OR ($1::bigint IS NULL AND UPPER(TRIM(b.code)) = UPPER(TRIM($2::text))))
		  AND (NULLIF(TRIM($3::text), '') IS NULL OR rc.code = TRIM($3::text))
		LIMIT 1`, idArg, codeArg, commandArg); err != nil {
		return nil, fmt.Errorf("carton de pre-enregistrement introuvable: %w", err)
	}

	var valise models.StorageLocation
	if err := s.db.Get(&valise, `
		INSERT INTO storage_locations (station_id, parent_location_id, zone, code, name, type, barcode, status)
		VALUES ($1, NULL, 'STOCK', $2, $3, 'VALISE', $2, 'LIBRE')
		ON CONFLICT (station_id, zone, code) DO UPDATE SET
			name = EXCLUDED.name,
			barcode = COALESCE(storage_locations.barcode, EXCLUDED.barcode)
		RETURNING id, station_id, parent_location_id, zone, code, name, barcode, type, capacity, status, created_at`,
		lookupStationID, carton.CaseCode, "Valise "+carton.CaseCode); err != nil {
		return nil, fmt.Errorf("impossible de preparer la valise %s: %w", carton.CaseCode, err)
	}

	var location models.StorageLocation
	if err := s.db.Get(&location, `
		INSERT INTO storage_locations (station_id, parent_location_id, zone, code, name, type, barcode, capacity, status)
		VALUES ($1, $2, 'STOCK', $3, $4, 'CARTON', $3, $5, 'LIBRE')
		ON CONFLICT (station_id, zone, code) DO UPDATE SET
			parent_location_id = EXCLUDED.parent_location_id,
			name = EXCLUDED.name,
			barcode = COALESCE(storage_locations.barcode, EXCLUDED.barcode),
			capacity = EXCLUDED.capacity
		RETURNING id, station_id, parent_location_id, zone, code, name, barcode, type, capacity, status, created_at`,
		lookupStationID, valise.ID, carton.BoxCode, "Carton "+carton.BoxCode, carton.Quantity); err != nil {
		return nil, fmt.Errorf("impossible de preparer le carton %s: %w", carton.BoxCode, err)
	}

	if location.Capacity != nil {
		var count int
		if err := s.db.Get(&count, `SELECT COUNT(*) FROM glasses WHERE location_id = $1`, location.ID); err != nil {
			return nil, fmt.Errorf("impossible de verifier la capacite du carton %s: %w", carton.BoxCode, err)
		}
		if count >= *location.Capacity {
			log.Printf("⚠️ Capacité carton atteinte: box_id=%d code=%s station=%d location_id=%d count=%d capacity=%d", carton.BoxID, carton.BoxCode, lookupStationID, location.ID, count, *location.Capacity)
			nextLocation, nextErr := s.createNextPreRegistrationCarton(lookupStationID, &valise, carton)
			if nextErr != nil {
				return nil, nextErr
			}
			log.Printf("✅ Nouveau carton créé pour la même commande: %s", nextLocation.Code)
			return nextLocation, nil
		}
	}

	return &location, nil
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
	lookupStationID := stationID
	if zone == models.ZoneStock {
		resolved, err := s.resolveStorageStationID(stationID)
		if err != nil {
			return nil, err
		}
		lookupStationID = resolved
	}

	prefix := codePrefix(baseCode)

	if prefix == "" {
		return nil, fmt.Errorf("code de référence invalide")
	}

	query := `
		SELECT sl.id, sl.station_id, sl.code, sl.name, sl.barcode, sl.type, sl.zone, sl.capacity, sl.status
		FROM storage_locations sl
		WHERE sl.station_id = $1
		  AND sl.zone = $2
		  AND sl.type = 'CARTON'
		  AND (sl.capacity IS NULL OR (SELECT COUNT(*) FROM glasses g WHERE g.location_id = sl.id) < sl.capacity)
		  AND code LIKE $3
		ORDER BY code
		LIMIT 1
	`

	var location models.StorageLocation

	err := s.db.Get(
		&location,
		query,
		lookupStationID,
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

	if zone != models.ZoneStock {
		_, err = s.db.Exec("UPDATE storage_locations SET status = 'OCCUPE' WHERE id = $1", location.ID)
		if err != nil {
			return nil, fmt.Errorf("impossible de réserver l'emplacement: %w", err)
		}
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
	if location, err := s.FindFreeLocation(stationID, zone); err == nil {
		return location, nil
	}
	if zone == models.ZoneStock {
		return nil, fmt.Errorf("aucun carton disponible : créez d'abord un carton dans une valise")
	}
	return nil, fmt.Errorf("aucun emplacement disponible pour la zone %s", zone)
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
	return s.FindFreeLocation(stationID, models.ZoneStock)
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
