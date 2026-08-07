package repositories

import (
	"fmt"

	"github.com/jmoiron/sqlx"
	"github.com/lunetterie/backend/internal/inventory/models"
)

type CountryRepository struct {
	db *sqlx.DB
}

func NewCountryRepository(db *sqlx.DB) *CountryRepository {
	return &CountryRepository{db: db}
}

func (r *CountryRepository) ListCountries() ([]models.Country, error) {
	countries := []models.Country{}
	err := r.db.Select(&countries, `
		SELECT id, nom, code
		FROM pays
		ORDER BY nom ASC
	`)
	if err != nil {
		return nil, fmt.Errorf("impossible de récupérer les pays: %w", err)
	}
	return countries, nil
}
