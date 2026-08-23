package models

import "time"

// SupplierOrder représente une commande passée à un fournisseur (ex. Dubai) :
// la quantité commandée sert de référence pour comparer, plus tard, ce qui est
// réellement envoyé au stock général via les sessions de réception.
type SupplierOrder struct {
	ID        int64     `db:"id" json:"id"`
	Supplier  string    `db:"supplier" json:"supplier"`
	Quantity  int       `db:"quantity" json:"quantity"`
	Gender    string    `db:"gender" json:"gender"`
	Gamme     string    `db:"gamme" json:"gamme"`
	OrderDate time.Time `db:"order_date" json:"order_date"`
	Note      *string   `db:"note" json:"note,omitempty"`
	CreatedBy *int64    `db:"created_by" json:"created_by,omitempty"`
	CreatedAt time.Time `db:"created_at" json:"created_at"`
	UpdatedAt time.Time `db:"updated_at" json:"updated_at"`
}

type SupplierOrderCreateRequest struct {
	Supplier  string `json:"supplier" binding:"required"`
	Quantity  int    `json:"quantity" binding:"required"`
	Gender    string `json:"gender" binding:"required"`
	Gamme     string `json:"gamme"`
	OrderDate string `json:"order_date" binding:"required"`
	Note      string `json:"note"`
}
