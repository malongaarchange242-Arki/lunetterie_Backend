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
	query := `SELECT s.id, s.name, s.type, s.city, p.nom AS country, s.address, s.phone, s.is_active, s.created_at
		FROM stations s
		LEFT JOIN pays p ON p.id = s.pays_id
		WHERE is_active = true
		ORDER BY name ASC`
	if err := r.db.Select(&stations, query); err != nil {
		return nil, fmt.Errorf("impossible de récupérer les stations: %w", err)
	}
	return stations, nil
}

func (r *StationRepository) CreateStore(country, city string) (*models.Station, error) {
	var station models.Station
	tx, err := r.db.Beginx()
	if err != nil {
		return nil, fmt.Errorf("impossible de préparer la création du magasin: %w", err)
	}
	defer tx.Rollback()

	var countryID int64
	if err := tx.Get(&countryID, `
		INSERT INTO pays (nom) VALUES ($1)
		ON CONFLICT (nom) DO UPDATE SET nom = EXCLUDED.nom
		RETURNING id`, country); err != nil {
		return nil, fmt.Errorf("impossible d'enregistrer le pays: %w", err)
	}

	var cityID int64
	if err := tx.Get(&cityID, `SELECT id FROM villes WHERE nom = $1 AND pays_id = $2 LIMIT 1`, city, countryID); err != nil {
		if err := tx.Get(&cityID, `INSERT INTO villes (nom, pays_id) VALUES ($1, $2) RETURNING id`, city, countryID); err != nil {
			return nil, fmt.Errorf("impossible d'enregistrer la ville: %w", err)
		}
	}

	query := `INSERT INTO stations (name, type, city, pays_id, ville_id, is_active, created_at)
		VALUES ($1, 'SOUS_STATION', $2, $3, $4, true, NOW())
		RETURNING id, name, type, city, address, phone, is_active, created_at`
	if err := tx.Get(&station, query, "Station "+city, city, countryID, cityID); err != nil {
		return nil, fmt.Errorf("impossible de créer le magasin: %w", err)
	}
	station.Country = &country
	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("impossible de valider le magasin: %w", err)
	}
	return &station, nil
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
