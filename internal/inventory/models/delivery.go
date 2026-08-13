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

// DeliveryLine : une ligne de livraison telle qu'on la lit, la monture jointe.
//
// Le poste affiche des lunettes, pas des identifiants : sans la référence, la marque et le
// prix, la liste ne dirait pas à l'écran laquelle des paires prêtes on tient en main.
// `HandedOverAt` distingue les deux moments que la même ligne porte : créée quand le
// laboratoire termine le montage, horodatée quand le client repart avec.
type DeliveryLine struct {
	ID           int64      `db:"id" json:"id"`
	DeliveryID   int64      `db:"delivery_id" json:"delivery_id"`
	GlassID      int64      `db:"glass_id" json:"glass_id"`
	HandedOverAt *time.Time `db:"handed_over_at" json:"handed_over_at,omitempty"`

	Barcode string   `db:"barcode" json:"barcode"`
	Status  string   `db:"status" json:"status"`
	Price   *float64 `db:"price" json:"price,omitempty"`

	Reference *string `db:"reference" json:"reference,omitempty"`
	Brand     *string `db:"brand" json:"brand,omitempty"`
	Shape     *string `db:"shape" json:"shape,omitempty"`
	Color     *string `db:"color" json:"color,omitempty"`

	StationID  int64     `db:"station_id" json:"station_id"`
	ReceivedAt time.Time `db:"received_at" json:"received_at"`
}
