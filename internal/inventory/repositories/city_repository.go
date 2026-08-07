package repositories

import (
	"fmt"

	"github.com/jmoiron/sqlx"
	"github.com/lunetterie/backend/internal/inventory/models"
)

type CityRepository struct {
	db *sqlx.DB
}

func NewCityRepository(db *sqlx.DB) *CityRepository {
	return &CityRepository{db: db}
}

func (r *CityRepository) ListCitiesByCountry(countryID int64) ([]models.City, error) {
	cities := []models.City{}
	err := r.db.Select(&cities, `
		SELECT id, nom, pays_id
		FROM villes
		WHERE pays_id = $1
		ORDER BY nom ASC
	`, countryID)
	if err != nil {
		return nil, fmt.Errorf("impossible de récupérer les villes: %w", err)
	}
	return cities, nil
}
