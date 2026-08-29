package models

import "time"

// SupplierOrder représente un bon de commande envoyé par l'administration.
type SupplierOrder struct {
	ID          int64     `db:"id" json:"id"`
	Reference   string    `db:"reference" json:"reference"`
	Supplier    string    `db:"supplier" json:"supplier"`
	Provenance  string    `db:"provenance" json:"provenance"`
	Destination string    `db:"destination" json:"destination"`
	Quantity    int       `db:"quantity" json:"quantity"`
	Gender      string    `db:"gender" json:"gender"`
	Gamme       string    `db:"gamme" json:"gamme"`
	OrderDate   time.Time `db:"order_date" json:"order_date"`
	Transport   string    `db:"transport" json:"transport"`
	BarcodeNum  string    `db:"barcode_num" json:"barcode_num"`
	Status      string    `db:"status" json:"status"`
	Note        *string   `db:"note" json:"note,omitempty"`
	CreatedBy   *int64    `db:"created_by" json:"created_by,omitempty"`
	CreatedAt   time.Time `db:"created_at" json:"created_at"`
	UpdatedAt   time.Time `db:"updated_at" json:"updated_at"`
}

type SupplierOrderCreateRequest struct {
	Reference   string `json:"reference"`
	Supplier    string `json:"supplier" binding:"required"`
	Provenance  string `json:"provenance"`
	Destination string `json:"destination"`
	Quantity    int    `json:"quantity" binding:"required"`
	Gender      string `json:"gender"`
	Gamme       string `json:"gamme"`
	OrderDate   string `json:"order_date" binding:"required"`
	Transport   string `json:"transport"`
	BarcodeNum  string `json:"barcode_num"`
	Note        string `json:"note"`
}
