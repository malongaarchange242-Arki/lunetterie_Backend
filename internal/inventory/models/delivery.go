package models

import "time"

// Delivery représente un arrivage / livraison créé depuis le poste Laboratoire
type Delivery struct {
	ID         int64     `db:"id" json:"id"`
	SupplierID int64     `db:"supplier_id" json:"supplier_id"`
	Reference  *string   `db:"reference" json:"reference,omitempty"`
	ReceivedBy *int64    `db:"received_by" json:"received_by,omitempty"`
	StationID  int64     `db:"station_id" json:"station_id"`
	Notes      *string   `db:"notes" json:"notes,omitempty"`
	ReceivedAt time.Time `db:"received_at" json:"received_at"`
	CreatedAt  time.Time `db:"created_at" json:"created_at"`
}

// DeliveryItem représente la relation entre delivery et monture
type DeliveryItem struct {
	ID         int64 `db:"id" json:"id"`
	DeliveryID int64 `db:"delivery_id" json:"delivery_id"`
	GlassID    int64 `db:"glass_id" json:"glass_id"`
}
