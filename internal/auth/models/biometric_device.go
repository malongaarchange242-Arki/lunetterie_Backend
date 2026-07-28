package models

type BiometricDevice struct {
	ID         int64  `db:"id" json:"id"`
	DeviceID   string `db:"device_id" json:"device_id"`
	DeviceName string `db:"device_name" json:"device_name,omitempty"`
	StationID  *int64 `db:"station_id" json:"station_id,omitempty"`
	IsActive   bool   `db:"is_active" json:"is_active"`
	LastSeen   string `db:"last_seen" json:"last_seen,omitempty"`
	CreatedAt  string `db:"created_at" json:"created_at"`
}
