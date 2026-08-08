package repositories

import (
	"fmt"

	"github.com/jmoiron/sqlx"
	"github.com/lunetterie/backend/internal/auth/models"
)

type StationRepository struct {
	db *sqlx.DB
}

func NewStationRepository(db *sqlx.DB) *StationRepository {
	return &StationRepository{db: db}
}

func (r *StationRepository) FindAll() ([]models.Station, error) {
	stations := []models.Station{}
	query := `SELECT id, name, type, city, address, phone, is_active, created_at
		FROM stations
		WHERE is_active = true
		ORDER BY name ASC`
	if err := r.db.Select(&stations, query); err != nil {
		return nil, fmt.Errorf("impossible de récupérer les stations: %w", err)
	}
	return stations, nil
}
func (r *StationRepository) GetByID(id int64) (*models.Station, error) {
	var station models.Station
	query := `SELECT id, name, type, city, address, phone, is_active, created_at FROM stations WHERE id = $1`
	if err := r.db.Get(&station, query, id); err != nil {
		return nil, fmt.Errorf("station introuvable: %w", err)
	}
	return &station, nil
}

// FindLocalStationsByCity récupère les magasins d'une ville — les stations locales vers
// lesquelles le Stock Général expédie les listes reçues.
//
// Le filtre sur le nom reprend la convention déjà appliquée côté scan.html : une sous-station
// dont le nom commence par « Station » est un magasin (ex: « Station Pointe-Noire »), les
// autres sont des postes dédiés (« Présentoir », « Laboratoire ») qui partagent la ville du
// siège sans être une destination d'expédition.
//
// Renvoie la liste complète plutôt qu'une seule station : l'appelant doit pouvoir distinguer
// « aucun magasin pour cette ville » de « plusieurs candidats », deux erreurs différentes pour
// le magasinier.
func (r *StationRepository) FindLocalStationsByCity(city string) ([]models.Station, error) {
	stations := []models.Station{}
	query := `SELECT id, name, type, city, address, phone, is_active, created_at
		FROM stations
		WHERE is_active = true
		  AND type = 'SOUS_STATION'
		  AND name ILIKE 'station%'
		  AND LOWER(TRIM(city)) = LOWER(TRIM($1))
		ORDER BY name ASC`
	if err := r.db.Select(&stations, query, city); err != nil {
		return nil, fmt.Errorf("impossible de récupérer les stations de la ville %q: %w", city, err)
	}
	return stations, nil
}

// FindByName récupère une station par son nom exact (ex: "Laboratoire", "Présentoir")
func (r *StationRepository) FindByName(name string) (*models.Station, error) {
	var station models.Station
	query := `SELECT id, name, type, city, address, phone, is_active, created_at FROM stations WHERE name = $1`
	if err := r.db.Get(&station, query, name); err != nil {
		return nil, fmt.Errorf("station introuvable: %w", err)
	}
	return &station, nil
}
