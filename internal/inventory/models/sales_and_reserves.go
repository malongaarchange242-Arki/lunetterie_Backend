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

// ReserveLine : une mise en réserve telle qu'on la lit, la monture jointe.
//
// `ReservedAt` est la vraie date de mise de côté. Elle a son importance : faute de cette
// lecture, les écrans calculaient l'ancienneté d'une réserve depuis `updated_at` de la
// monture — c'est-à-dire depuis n'importe quelle modification, et non depuis la mise de
// côté. Le délai de dix jours se comptait donc à partir de la mauvaise date.
type ReserveLine struct {
	ID        int64 `db:"id" json:"id"`
	ReserveID int64 `db:"reserve_id" json:"reserve_id"`
	GlassID   int64 `db:"glass_id" json:"glass_id"`

	StationID  int64     `db:"station_id" json:"station_id"`
	UserID     int64     `db:"user_id" json:"user_id"`
	ReservedAt time.Time `db:"reserved_at" json:"reserved_at"`

	Barcode string   `db:"barcode" json:"barcode"`
	Status  string   `db:"status" json:"status"`
	Price   *float64 `db:"price" json:"price,omitempty"`

	Reference *string `db:"reference" json:"reference,omitempty"`
	Brand     *string `db:"brand" json:"brand,omitempty"`
	Shape     *string `db:"shape" json:"shape,omitempty"`
	Color     *string `db:"color" json:"color,omitempty"`
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
