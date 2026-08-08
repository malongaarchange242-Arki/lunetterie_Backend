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

// ExpiredReserve : une monture encore RESERVEE dont la mise de côté a dépassé le délai.
// UserID est celui qui avait posé la réservation : une tâche automatique n'a pas
// d'utilisateur courant, et un mouvement sans auteur est refusé par la base.
type ExpiredReserve struct {
	GlassID    int64     `db:"glass_id"`
	Barcode    string    `db:"barcode"`
	StationID  int64     `db:"station_id"`
	LocationID *int64    `db:"location_id"`
	UserID     int64     `db:"user_id"`
	ReservedAt time.Time `db:"reserved_at"`
}
