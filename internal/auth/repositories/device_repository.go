package repositories

import (
	"fmt"

	"github.com/jmoiron/sqlx"
	"github.com/lunetterie/backend/internal/auth/models"
)

type DeviceRepository struct {
	db *sqlx.DB
}

func NewDeviceRepository(db *sqlx.DB) *DeviceRepository {
	return &DeviceRepository{db: db}
}

func (r *DeviceRepository) FindByDeviceID(deviceID string) (*models.BiometricDevice, error) {
	device := models.BiometricDevice{}
	query := `SELECT id, device_id, device_name, station_id, is_active, last_seen, created_at FROM biometric_devices WHERE device_id = $1`
	if err := r.db.Get(&device, query, deviceID); err != nil {
		return nil, fmt.Errorf("device non trouvé: %w", err)
	}
	return &device, nil
}
