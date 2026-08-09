package models

import "time"

const (
	ClaimStatusOuverte = "OUVERTE"
	ClaimStatusTraitee = "TRAITEE"
)

type Claim struct {
	ID         int64     `db:"id" json:"id"`
	StationID  int64     `db:"station_id" json:"station_id"`
	ClientName string    `db:"client_name" json:"client_name"`
	Barcode    string    `db:"barcode" json:"barcode,omitempty"`
	Motif      string    `db:"motif" json:"motif"`
	Detail     *string   `db:"detail" json:"detail,omitempty"`
	Status     string    `db:"status" json:"status"`
	CreatedBy  *int64    `db:"created_by" json:"created_by,omitempty"`
	CreatedAt  time.Time `db:"created_at" json:"created_at"`
	UpdatedAt  time.Time `db:"updated_at" json:"updated_at"`
}

type ClaimCreateRequest struct {
	StationID  int64  `json:"station_id" binding:"required"`
	ClientName string `json:"client_name" binding:"required"`
	Barcode    string `json:"barcode"`
	Motif      string `json:"motif" binding:"required"`
	Detail     string `json:"detail"`
}
