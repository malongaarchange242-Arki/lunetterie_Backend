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
