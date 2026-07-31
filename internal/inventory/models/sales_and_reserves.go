package models

import "time"

type Sale struct {
	ID        int64     `db:"id" json:"id"`
	StationID int64     `db:"station_id" json:"station_id"`
	UserID    int64     `db:"user_id" json:"user_id"`
	Notes     *string   `db:"notes" json:"notes,omitempty"`
	CreatedAt time.Time `db:"created_at" json:"created_at"`
}

type SaleItem struct {
	ID      int64 `db:"id" json:"id"`
	SaleID  int64 `db:"sale_id" json:"sale_id"`
	GlassID int64 `db:"glass_id" json:"glass_id"`
}

type Reserve struct {
	ID        int64     `db:"id" json:"id"`
	StationID int64     `db:"station_id" json:"station_id"`
	UserID    int64     `db:"user_id" json:"user_id"`
	Notes     *string   `db:"notes" json:"notes,omitempty"`
	CreatedAt time.Time `db:"created_at" json:"created_at"`
}

type ReserveItem struct {
	ID        int64 `db:"id" json:"id"`
	ReserveID int64 `db:"reserve_id" json:"reserve_id"`
	GlassID   int64 `db:"glass_id" json:"glass_id"`
}
