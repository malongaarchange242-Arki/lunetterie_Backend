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

// FindOrCreatePresentoirByCode retrouve un casier de présentoir par son code (« PR01-1 »),
// et le crée s'il n'existe pas encore.
//
// La création est volontaire : les casiers ne sont pas tous générés d'avance — l'allocation
// automatique les fabrique elle-même au fil de l'eau (allocation_service.go
// findOrCreateLocation). Refuser un code absent obligerait à peupler la table à la main
// avant de pouvoir désigner un casier physiquement existant.
//
// Le statut n'est pas touché ici : c'est l'appelant qui décide de l'occuper, une fois qu'il
// a vérifié que le casier était libre.
func (r *LocationRepository) FindOrCreatePresentoirByCode(stationID int64, code string) (*models.StorageLocation, error) {
	var location models.StorageLocation

	query := `
		SELECT id, station_id, code, type, zone, status
		FROM storage_locations
		WHERE station_id = $1 AND zone = 'PRESENTOIR' AND UPPER(TRIM(code)) = $2`
	if err := r.db.Get(&location, query, stationID, code); err == nil {
		return &location, nil
	}

	insert := `
		INSERT INTO storage_locations (station_id, zone, code, type, status)
		VALUES ($1, 'PRESENTOIR', $2, 'POSITION', 'LIBRE')
		ON CONFLICT (station_id, zone, code) DO UPDATE SET code = EXCLUDED.code
		RETURNING id, station_id, code, type, zone, status`
	if err := r.db.Get(&location, insert, stationID, code); err != nil {
		return nil, fmt.Errorf("impossible de créer le casier %s: %w", code, err)
	}
	return &location, nil
}

// FindGlassBarcodeAtLocation dit quelle monture occupe un casier. Chaîne vide s'il est libre.
// Sert à refuser un casier déjà pris en nommant l'occupant : « PR01-1 est déjà occupé par
// LUN-CNG-0004 » oriente, là où « casier occupé » oblige à aller regarder.
func (r *LocationRepository) FindGlassBarcodeAtLocation(locationID int64) (string, error) {
	barcodes := []string{}
	query := `SELECT barcode FROM glasses WHERE location_id = $1 LIMIT 1`
	if err := r.db.Select(&barcodes, query, locationID); err != nil {
		return "", fmt.Errorf("impossible de lire l'occupant du casier: %w", err)
	}
	if len(barcodes) == 0 {
		return "", nil
	}
	return barcodes[0], nil
}

// FindEmptyPresentoirSlotsToday liste les emplacements de la zone PRESENTOIR d'une station qui
// sont actuellement libres ET ont été libérés aujourd'hui suite à une vente, une réserve ou un
// envoi en caisse (pas un simple emplacement jamais utilisé), avec la monture qui les occupait
// — pour savoir quels emplacements physiques remplir en fin de journée, et avec quoi.
//
// MISE_EN_CAISSE compte : depuis que « Envoyer » pousse la monture au comptoir, c'est ce
// mouvement-là qui vide le casier, pas la vente qui vient après. Une monture que le client
// refuse et qu'on repose au présentoir réoccupe son emplacement, donc le filtre sur
// status = 'LIBRE' la fait ressortir d'elle-même de la liste.
func (r *LocationRepository) FindEmptyPresentoirSlotsToday(stationID int64) ([]models.EmptySlot, error) {
	slots := []models.EmptySlot{}
	// DISTINCT ON (sl.id) + ORDER BY m.created_at DESC : si un emplacement a été libéré plusieurs
	// fois aujourd'hui (rempli puis vidé à nouveau), on ne garde que la dernière monture en date.
	query := `
		SELECT code, barcode, reference, brand FROM (
			SELECT DISTINCT ON (sl.id)
				sl.id AS location_id, sl.code, g.barcode,
				ga.reference, ga.brand
			FROM storage_locations sl
			JOIN movements m ON m.from_location_id = sl.id
			JOIN glasses g ON g.id = m.glass_id
			LEFT JOIN glass_analysis ga ON ga.id = g.analysis_id
			WHERE sl.station_id = $1
			  AND sl.zone = 'PRESENTOIR'
			  AND sl.status = 'LIBRE'
			  AND m.action IN ('RETRAIT_PRESENTOIR', 'RESERVATION', 'MISE_EN_CAISSE')
			  AND m.created_at::date = CURRENT_DATE
			ORDER BY sl.id, m.created_at DESC
		) latest
		ORDER BY code`
	if err := r.db.Select(&slots, query, stationID); err != nil {
		return nil, fmt.Errorf("impossible de récupérer les emplacements vides: %w", err)
	}
	return slots, nil
}
