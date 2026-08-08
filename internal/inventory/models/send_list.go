package models

import "time"

// SendList est la liste des montures d'une session de réception, envoyée par la direction
// vers un magasin. Elle sert d'ordre de préparation au poste de scan du stock général.
type SendList struct {
	ID          int64     `db:"id" json:"id"`
	SessionCode string    `db:"session_code" json:"session_code"`
	City        string    `db:"city" json:"city"`
	ItemCount   int       `db:"item_count" json:"item_count"`
	Status      string    `db:"status" json:"status"`
	CreatedBy   *int64    `db:"created_by" json:"created_by,omitempty"`
	CreatedAt   time.Time `db:"created_at" json:"created_at"`
	UpdatedAt   time.Time `db:"updated_at" json:"updated_at"`
}

// SendListItem recopie les attributs de la monture au moment de l'envoi : la liste doit
// rester lisible même si la monture bouge ou est supprimée ensuite.
type SendListItem struct {
	ID           int64     `db:"id" json:"id"`
	ListID       int64     `db:"list_id" json:"list_id"`
	GlassID      *int64    `db:"glass_id" json:"glass_id,omitempty"`
	Barcode      *string   `db:"barcode" json:"barcode,omitempty"`
	Reference    *string   `db:"reference" json:"reference,omitempty"`
	Brand        *string   `db:"brand" json:"brand,omitempty"`
	LocationCode *string   `db:"location_code" json:"location_code,omitempty"`
	CreatedAt    time.Time `db:"created_at" json:"created_at"`
}

type SendListItemRequest struct {
	GlassID      *int64 `json:"glass_id"`
	Barcode      string `json:"barcode"`
	Reference    string `json:"reference"`
	Brand        string `json:"brand"`
	LocationCode string `json:"location_code"`
}

type SendListCreateRequest struct {
	SessionCode string                `json:"session_code" binding:"required"`
	City        string                `json:"city" binding:"required"`
	Items       []SendListItemRequest `json:"items"`
}
