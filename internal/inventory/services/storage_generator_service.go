package services

import (
	"fmt"

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

// GenerateLocations invoque la fonction SQL de génération d'emplacements
func (s *StorageGeneratorService) GenerateLocations(stationID int64, template StationTemplate) (int, error) {
	var total int
	err := s.db.QueryRow(
		"SELECT generate_station_locations($1, $2, $3, $4, $5)",
		stationID,
		template.NumRayons,
		template.EtageresParRayon,
		template.BacsParEtagere,
		template.PositionsParBac,
	).Scan(&total)
	if err != nil {
		return 0, fmt.Errorf("erreur génération emplacements: %w", err)
	}
	return total, nil
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
func (s *StorageGeneratorService) PeekFreeLocation(stationID int64, zone models.ZoneType) (int64, string, string, error) {
	return s.PeekFreeLocationForPrice(stationID, zone, nil, "")
}

// PeekFreeLocationForPrice retourne le prochain emplacement libre d'un pool spécifique selon la gamme.
func (s *StorageGeneratorService) PeekFreeLocationForPrice(stationID int64, zone models.ZoneType, price *float64, gamme string) (int64, string, string, error) {
	query := `
		SELECT id, station_id, code, type, zone, status
		FROM storage_locations
		WHERE station_id = $1 AND zone = $2 AND type = 'POSITION' AND status = 'LIBRE'
		ORDER BY code`

	var locations []models.StorageLocation
	err := s.db.Select(&locations, query, stationID, zone)
	if err != nil {
		return 0, "", "", fmt.Errorf("impossible de lister les emplacements libres: %w", err)
	}
	if len(locations) == 0 {
		return 0, "", "", fmt.Errorf("aucun emplacement libre: aucune position disponible")
	}

	selected := selectLocationByTier(locations, classifyPriceTier(price, gamme))
	if selected == nil {
		selected = &locations[0]
	}

	path, code, err := s.GetLocationPath(selected.ID)
	if err != nil {
		return selected.ID, "", "", nil
	}
	return selected.ID, path, code, nil
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
