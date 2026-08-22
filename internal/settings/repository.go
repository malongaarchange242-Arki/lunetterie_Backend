package settings

import (
	"database/sql"
	"encoding/json"
	"fmt"

	"github.com/jmoiron/sqlx"
)

type Config struct {
	FeatureFlags       map[string]map[string]bool `json:"featureFlags"`
	PosteEnabled       map[string]bool            `json:"posteEnabled"`
	CountryEnabled     map[string]bool            `json:"countryEnabled"`
	CityEnabled        map[string]bool            `json:"cityEnabled"`
	SousStationEnabled map[string]map[string]bool `json:"sousStationEnabled"`
}

type Repository struct {
	db *sqlx.DB
}

func NewRepository(db *sqlx.DB) *Repository {
	return &Repository{db: db}
}

func (r *Repository) Get() (Config, error) {
	var raw []byte
	err := r.db.Get(&raw, `SELECT value FROM app_settings WHERE key = 'feature_flags'`)
	if err != nil {
		if err == sql.ErrNoRows {
			return Config{}, nil
		}
		return Config{}, fmt.Errorf("impossible de lire les réglages: %w", err)
	}

	var config Config
	if err := json.Unmarshal(raw, &config); err != nil {
		return Config{}, fmt.Errorf("réglages invalides en base: %w", err)
	}
	return config, nil
}

func (r *Repository) Save(config Config) error {
	raw, err := json.Marshal(config)
	if err != nil {
		return fmt.Errorf("impossible d'encoder les réglages: %w", err)
	}

	_, err = r.db.Exec(`
		INSERT INTO app_settings (key, value, updated_at)
		VALUES ('feature_flags', $1::jsonb, NOW())
		ON CONFLICT (key) DO UPDATE SET value = EXCLUDED.value, updated_at = NOW()
	`, raw)
	if err != nil {
		return fmt.Errorf("impossible d'enregistrer les réglages: %w", err)
	}
	return nil
}
