package services

import (
	"fmt"
	"time"

	"github.com/jmoiron/sqlx"
	"github.com/lunetterie/backend/internal/inventory/models"
)

// StorageGeneratorService gère la génération d'emplacements et la résolution des chemins
type StorageGeneratorService struct {
	db *sqlx.DB
}

// NewStorageGeneratorService crée une nouvelle instance
func NewStorageGeneratorService(db *sqlx.DB) *StorageGeneratorService {
	return &StorageGeneratorService{db: db}
}

// StationTemplate contient la configuration d'une station de stockage
type StationTemplate struct {
	Name             string `json:"name"`
	NumRayons        int    `json:"num_rayons"`
	EtageresParRayon int    `json:"etageres_par_rayon"`
	BacsParEtagere   int    `json:"bacs_par_etagere"`
	PositionsParBac  int    `json:"positions_par_bac"`
}

func (s *StorageGeneratorService) CreateLocation(stationID int64, parentID *int64, locationType string, capacity *int) (*models.StorageLocation, error) {
	var location models.StorageLocation
	if locationType != "VALISE" && locationType != "CARTON" {
		return nil, fmt.Errorf("type d'emplacement invalide pour le nouveau modèle: %s", locationType)
	}
	if capacity != nil && *capacity < 1 {
		return nil, fmt.Errorf("la capacité doit être positive ou nulle pour un carton illimité")
	}

	if locationType == "VALISE" {
		if parentID != nil {
			return nil, fmt.Errorf("une VALISE ne peut pas avoir de parent")
		}
		capacity = nil
	} else if locationType == "CARTON" {
		if parentID == nil {
			return nil, fmt.Errorf("un CARTON doit appartenir à une VALISE")
		}
		parentType := ""
		if err := s.db.Get(&parentType, `SELECT type FROM storage_locations WHERE id = $1 AND station_id = $2`, *parentID, stationID); err != nil {
			return nil, fmt.Errorf("parent introuvable dans la station")
		}
		if parentType != "VALISE" {
			return nil, fmt.Errorf("un CARTON doit avoir une VALISE comme parent")
		}
	}

	temporaryCode := fmt.Sprintf("TEMP-%d", time.Now().UnixNano())
	if err := s.db.Get(&location, `
		INSERT INTO storage_locations (station_id, parent_location_id, zone, code, name, type, capacity, status)
		VALUES ($1, $2, 'STOCK', $3, $4, $5, $6, 'LIBRE')
		RETURNING id, station_id, parent_location_id, zone, code, name, type, capacity, status, created_at`,
		stationID, parentID, temporaryCode, temporaryCode, locationType, capacity); err != nil {
		return nil, fmt.Errorf("impossible de créer l'emplacement: %w", err)
	}
	prefix := map[string]string{"VALISE": "VAL", "CARTON": "CAR"}[locationType]
	location.Code = fmt.Sprintf("%s-CNG-%d", prefix, location.ID)
	location.Name = fmt.Sprintf("%s %d", locationType, location.ID)
	if _, err := s.db.Exec(`UPDATE storage_locations SET code = $1, name = $2, barcode = $3 WHERE id = $4`, location.Code, location.Name, location.Code, location.ID); err != nil {
		return nil, fmt.Errorf("impossible de finaliser l'emplacement: %w", err)
	}
	location.Barcode = &location.Code
	return &location, nil
}

// GetLocationPath récupère le chemin lisible et le code d'un emplacement
func (s *StorageGeneratorService) GetLocationPath(locationID int64) (string, string, error) {
	var path, code string
	err := s.db.QueryRow(
		"SELECT full_name, code FROM v_location_tree WHERE id = $1",
		locationID,
	).Scan(&path, &code)
	if err != nil {
		return "", "", fmt.Errorf("emplacement introuvable: %w", err)
	}
	return path, code, nil
}

// PeekFreeLocation retourne le prochain emplacement libre pour une station SANS le réserver
// (pas de passage en 'OCCUPE') : utilisé pour un simple affichage prévisionnel avant
// l'enregistrement réel. L'emplacement effectivement attribué à la validation peut différer
// si un autre enregistrement concurrent le prend entre-temps.
//
// Premier emplacement libre dans l'ordre du code, sans tri par tranche de prix : c'est la
// même règle que AllocationService.FindFreeLocation, pour que cet aperçu montre exactement
// l'emplacement que l'enregistrement va réellement attribuer.
func (s *StorageGeneratorService) PeekFreeLocation(stationID int64, zone models.ZoneType) (int64, string, string, error) {
	lookupStationID := stationID
	if zone == models.ZoneStock {
		if err := s.db.Get(&lookupStationID, `
			SELECT id
			FROM stations
			WHERE city = (SELECT city FROM stations WHERE id = $1)
			  AND name ILIKE 'Stock Général%'
			ORDER BY CASE WHEN id = $1 THEN 0 ELSE 1 END, id
			LIMIT 1`, stationID); err != nil {
			return 0, "", "", fmt.Errorf("impossible de trouver le stock physique: %w", err)
		}
	}
	var location models.StorageLocation
	err := s.db.Get(&location, `
		SELECT id, station_id, code, type, zone, status
		FROM storage_locations
		WHERE station_id = $1
		  AND zone = $2
		  AND type = CASE WHEN $2 = 'STOCK' THEN 'CARTON' ELSE 'PRESENTOIR' END
		  AND status = 'LIBRE'
		ORDER BY code
		LIMIT 1`, lookupStationID, zone)
	if err != nil {
		return 0, "", "", fmt.Errorf("aucun emplacement libre disponible")
	}

	path, code, err := s.GetLocationPath(location.ID)
	if err != nil {
		return location.ID, "", "", nil
	}
	return location.ID, path, code, nil
}

// FindFreeLocation trouve et réserve le premier emplacement libre pour une station
func (s *StorageGeneratorService) FindFreeLocation(stationID int64, zone models.ZoneType) (int64, string, string, error) {
	var locationID int64
	err := s.db.QueryRow(
		"SELECT find_free_location($1, $2)",
		stationID,
		zone,
	).Scan(&locationID)
	if err != nil {
		return 0, "", "", fmt.Errorf("aucun emplacement libre: %w", err)
	}

	path, code, err := s.GetLocationPath(locationID)
	if err != nil {
		return locationID, "", "", nil
	}
	return locationID, path, code, nil
}
