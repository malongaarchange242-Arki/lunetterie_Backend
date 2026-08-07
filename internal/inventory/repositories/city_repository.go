package repositories

import (
	"fmt"
	"strings"

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
		if isMissingTableErrorForCityRepository(err) {
			return fallbackCities(countryID), nil
		}
		return nil, fmt.Errorf("impossible de récupérer les villes: %w", err)
	}
	if len(cities) == 0 {
		return fallbackCities(countryID), nil
	}
	return cities, nil
}

func fallbackCities(countryID int64) []models.City {
	switch countryID {
	case 1:
		return []models.City{{ID: 1, Name: "Brazzaville", CountryID: 1}, {ID: 2, Name: "Pointe-Noire", CountryID: 1}, {ID: 3, Name: "Dolisie", CountryID: 1}}
	case 2:
		return []models.City{{ID: 4, Name: "Kinshasa", CountryID: 2}, {ID: 5, Name: "Lubumbashi", CountryID: 2}, {ID: 6, Name: "Goma", CountryID: 2}}
	default:
		return []models.City{{ID: 100, Name: "Autre", CountryID: countryID}}
	}
}

func isMissingTableErrorForCityRepository(err error) bool {
	if err == nil {
		return false
	}
	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, "does not exist") || strings.Contains(msg, "relation") || strings.Contains(msg, "undefinedtable")
}
