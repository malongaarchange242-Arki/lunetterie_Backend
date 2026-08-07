package repositories

import (
	"fmt"
	"strings"

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
		if isMissingTableErrorForCountryRepository(err) {
			return fallbackCountries(), nil
		}
		return nil, fmt.Errorf("impossible de récupérer les pays: %w", err)
	}
	if len(countries) == 0 {
		return fallbackCountries(), nil
	}
	return countries, nil
}

func fallbackCountries() []models.Country {
	return []models.Country{
		{ID: 1, Name: "Congo", Code: "CG"},
		{ID: 2, Name: "République démocratique du Congo", Code: "CD"},
	}
}

func isMissingTableErrorForCountryRepository(err error) bool {
	if err == nil {
		return false
	}
	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, "does not exist") || strings.Contains(msg, "relation") || strings.Contains(msg, "undefinedtable")
}
